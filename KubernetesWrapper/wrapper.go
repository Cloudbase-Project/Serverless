package kuberneteswrapper

import (
	"context"

	"github.com/Cloudbase-Project/serverless/constants"
	"k8s.io/client-go/kubernetes"

	// appsv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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
	ImageTag   string
}

func NewWrapper(client *kubernetes.Clientset) *KubernetesWrapper {
	return &KubernetesWrapper{KClient: client}
}

func (kw *KubernetesWrapper) BuildLabel(key string, value []string) (*labels.Requirement, error) {
	return labels.NewRequirement(key, selection.Equals, value)
}

func (kw *KubernetesWrapper) WatchImageBuilder(
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

func (kw *KubernetesWrapper) CreateImageBuilder(ib *ImageBuilder) (*corev1.Pod, error) {

	var Dockerfile string
	if ib.Language == constants.NODEJS {
		Dockerfile = constants.NodejsDockerfile

	}
	// else if ib.Language == constants.GOLANG {
	// 	Dockerfile = constants.GolangDockerfile
	// }

	return kw.KClient.CoreV1().Pods(ib.Namespace).Create(ib.Ctx, &corev1.Pod{
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
					"curl -XGET http://cloudbase-serverless-srv.default:3000/worker/queue -o /workspace/index.js && echo -e " + Dockerfile + " >> /workspace/Dockerfile && echo -e " + constants.NodejsPackageJSON + " >> /workspace/package.json && echo -e " + constants.RegistryCredentials + " >> /kaniko/.docker/config.json ",
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerConfig",
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
					"--destination=" + ib.ImageTag,
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerConfig",
					MountPath: "/kaniko/.docker",
				}},
			}},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{{
				Name: "shared", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}, {
				Name: "dockerConfig", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			}},
		},
	}, metav1.CreateOptions{})

}

func (kw *KubernetesWrapper) CreateNamespace(
	ctx context.Context,
	namespace string,
) (*corev1.Namespace, error) {

	return kw.KClient.CoreV1().
		Namespaces().
		Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{})

}

func (kw *KubernetesWrapper) CreateDeployment(ctx context.Context, namespace string) {}

func (kw *KubernetesWrapper) CreateService(ctx context.Context, namespace string) {}
