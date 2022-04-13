package kuberneteswrapper

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Cloudbase-Project/serverless/constants"
	"github.com/Cloudbase-Project/serverless/utils"
	"k8s.io/client-go/kubernetes"

	// appsv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
)

type KubernetesWrapper struct {
	KClient *kubernetes.Clientset
}

type ImageBuilder struct {
	Ctx        context.Context
	Namespace  string
	FunctionId string
	Language   constants.Language
	ImageName  string
	Code       string
}

type DeploymentOptions struct {
	Ctx             context.Context
	Namespace       string
	FunctionId      string
	DeploymentLabel map[string]string
	ImageName       string
	Replicas        int32
}

type ServiceOptions struct {
	Ctx             context.Context
	Namespace       string
	FunctionId      string
	DeploymentLabel map[string]string
}

type UpdateOptions struct {
	Ctx       context.Context
	Namespace string
	Name      string
}

type DeleteOptions struct {
	Ctx       context.Context
	Name      string
	Namespace string
}

func NewWrapper(client *kubernetes.Clientset) *KubernetesWrapper {
	return &KubernetesWrapper{KClient: client}
}

func (kw *KubernetesWrapper) BuildLabel(key string, value []string) (*labels.Requirement, error) {
	return labels.NewRequirement(key, selection.Equals, value)
}

func (kw *KubernetesWrapper) GetImageBuilderWatcher(
	ctx context.Context,
	label string,
) (watch.Interface, error) {
	return kw.KClient.CoreV1().
		Pods(constants.Namespace).
		Watch(
			// TODO: Donno if the request context should be used here or a custom timeout context should be used here.
			// r.Context(),
			ctx,
			metav1.ListOptions{LabelSelector: label})
}

func (kw *KubernetesWrapper) GetDeploymentWatcher(
	ctx context.Context,
	label string,
	namespace string,
) (watch.Interface, error) {
	return kw.KClient.AppsV1().
		Deployments(namespace).
		Watch(ctx, metav1.ListOptions{LabelSelector: label})
}

// Build an image for the given functionId and image name
func (kw *KubernetesWrapper) CreateImageBuilder(ib *ImageBuilder) (*corev1.Pod, error) {

	var Dockerfile string
	if ib.Language == constants.NODEJS {
		Dockerfile = constants.NodejsDockerfile

	}
	// else if ib.Language == constants.GOLANG {
	// 	Dockerfile = constants.GolangDockerfile
	// }

	m1 := regexp.MustCompile(`"`)
	packagejson := m1.ReplaceAllString(constants.NodejsPackageJSON, `\"`)
	dockerfile := m1.ReplaceAllString(Dockerfile, `\"`)

	REGISTRY := os.Getenv("REGISTRY")
	BASE64_CREDENTIALS := os.Getenv("BASE64_CREDENTIALS")

	pod, err := kw.KClient.CoreV1().Pods(ib.Namespace).Create(ib.Ctx, &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kaniko-worker",
			Labels: map[string]string{
				"builder": ib.FunctionId, // the code id
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:  "setup-kaniko",
				Image: "yauritux/busybox-curl",
				Command: []string{
					"/bin/sh",
					"-c",
					// "curl -XGET http://cloudbase-serverless-srv.default:3000/worker/queue -o /workspace/index.js && echo -e " + Dockerfile + " >> /workspace/Dockerfile && echo -e " + constants.NodejsPackageJSON + " >> /workspace/package.json && echo -e " + constants.RegistryCredentials + " >> /kaniko/.docker/config.json ",
					// "echo -e " + ib.Code + " >> /workspace/index.js && echo -e " + Dockerfile + " >> /workspace/Dockerfile && echo -e " + constants.NodejsPackageJSON + " >> /workspace/package.json",
					`echo -e  "` + ib.Code + `" >> /workspace/index.js && echo -e "` + dockerfile + `" >> /workspace/Dockerfile && echo -e "` + packagejson + `" >> /workspace/package.json && echo -e "{\"auths\":{\"` + REGISTRY + `\":{\"auth\": \"` + BASE64_CREDENTIALS + `\" }}}" > /kaniko/.docker/config.json`,
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerconfig",
					MountPath: "/kaniko/.docker",
				}},
			}},
			Containers: []corev1.Container{{
				Name:  "kaniko-executor",
				Image: "gcr.io/kaniko-project/executor:latest",
				Args: []string{
					"--dockerfile=/workspace/Dockerfile",
					"--context=dir:///workspace",
					// "--no-push",
					"--destination=" + ib.ImageName,
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerconfig",
					MountPath: "/kaniko/.docker",
				}},
			}},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{{
				Name: "shared", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			},
				{
					Name: "dockerconfig", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
				// {Name: "dockerconfig",
				// 	VolumeSource: corev1.VolumeSource{
				// 		Secret: &corev1.SecretVolumeSource{
				// 			SecretName: "regcred",
				// 			Items: []corev1.KeyToPath{
				// 				{Key: ".dockerconfigjson", Path: "config.json"},
				// 			},
				// 		},
				// 	}},
			},
		},
	}, metav1.CreateOptions{})
	fmt.Printf("err: %v\n", err)
	return pod, err
}

func (kw *KubernetesWrapper) CreateNamespace(
	ctx context.Context,
	namespace string,
) (*corev1.Namespace, error) {

	return kw.KClient.CoreV1().
		Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})

}

func (kw *KubernetesWrapper) CreateDeployment(options *DeploymentOptions) (*v1.Deployment, error) {
	return kw.KClient.AppsV1().
		Deployments(options.Namespace).
		Create(options.Ctx,
			&v1.Deployment{
				TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				// TODO:
				ObjectMeta: metav1.ObjectMeta{
					Name:   options.FunctionId,
					Labels: map[string]string{"app": options.FunctionId},
				},
				Spec: v1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						// TODO:
						MatchLabels: options.DeploymentLabel,
					},
					Replicas: &options.Replicas, // TODO: Have to do more here
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: options.DeploymentLabel},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyAlways,
							Containers: []corev1.Container{{
								// TODO:
								Name:  options.FunctionId,
								Image: options.ImageName, // "image name from db", // should be ghcr.io/projectname/codeId:latest
								Ports: []corev1.ContainerPort{{ContainerPort: 3000}},
							}},
							ImagePullSecrets: []corev1.LocalObjectReference{{Name: "regcred"}},
						},
					},
				},
			}, metav1.CreateOptions{})
}

func (kw *KubernetesWrapper) CreateService(options *ServiceOptions) (*corev1.Service, error) {

	serviceName := utils.BuildServiceName(options.FunctionId)

	return kw.KClient.CoreV1().
		Services(options.Namespace).
		Create(options.Ctx, &corev1.Service{
			TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
			},
			Spec: corev1.ServiceSpec{
				Selector: options.DeploymentLabel,
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Port: 4000, TargetPort: intstr.FromInt(4000)},
				},
			},
		}, metav1.CreateOptions{})
}

// Delete the deployment
func (kw *KubernetesWrapper) DeleteDeployment(options *DeleteOptions) error {
	return kw.KClient.AppsV1().
		Deployments(options.Namespace).
		Delete(options.Ctx, options.Name, metav1.DeleteOptions{})
}

// Delete the service
func (kw *KubernetesWrapper) DeleteService(options *DeleteOptions) error {
	return kw.KClient.CoreV1().
		Services(options.Namespace).
		Delete(options.Ctx, options.Name, metav1.DeleteOptions{})
}

// updates the deployment label with current timestamp to trigger a redeploy
func (kw *KubernetesWrapper) UpdateDeployment(options *UpdateOptions) error {

	deployment, err := kw.KClient.AppsV1().
		Deployments(options.Namespace).
		Get(options.Ctx, options.Name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	deployment.Spec.Template.ObjectMeta.Annotations["date"] = time.Now().String()

	_, err = kw.KClient.AppsV1().
		Deployments(options.Namespace).
		Update(options.Ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
