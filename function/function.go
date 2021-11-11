package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	kuberneteswrapper "github.com/Cloudbase-Project/serverless/KubernetesWrapper"
	"github.com/Cloudbase-Project/serverless/constants"
	v1 "k8s.io/api/apps/v1"

	// appsv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PostCodeDTO struct {
	code     string
	language constants.Language
}

type Function struct {
	l  *log.Logger
	kw *kuberneteswrapper.KubernetesWrapper
}

func NewFunction(client *kubernetes.Clientset, l *log.Logger) *Function {
	kw := kuberneteswrapper.NewWrapper(client)
	return &Function{l: l, kw: kw}
}

func fromJSON(body io.Reader, value interface{}) interface{} {
	d := json.NewDecoder(body)
	return d.Decode(value)
}

func (f *Function) ListFunctions(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *Function) UpdateFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *Function) DeleteFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *Function) GetFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *Function) GetFunctionLogs(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *Function) DeployFunction(rw http.ResponseWriter, r *http.Request) {
	// http.Error(rw, "Not Implemented", 500)

	// TODO: Get function from db.
	// check if status is complete and only then try to deploy

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	deploymentLabel := map[string]string{"app": "codeId"}

	var replicas int32
	replicas = 1

	deployment, err := clientset.AppsV1().
		Deployments(constants.Namespace).
		Create(context.Background(),
			&v1.Deployment{
				TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				// TODO:
				ObjectMeta: metav1.ObjectMeta{Name: "codeId"},
				Spec: v1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						// TODO:
						MatchLabels: deploymentLabel,
					},
					Replicas: &replicas, // TODO: Have to do more here
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: deploymentLabel},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyAlways,
							Containers: []corev1.Container{{
								// TODO:
								Name:  "codeId",
								Image: "image name from db", // should be ghcr.io/projectname/codeId:latest
								Ports: []corev1.ContainerPort{{ContainerPort: 3000}},
							}},
						},
					},
				},
			}, metav1.CreateOptions{})

	// create a clusterIP service for the deployment

	service, err := clientset.CoreV1().
		Services(constants.Namespace).
		Create(r.Context(), &corev1.Service{
			TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "codeid" + "srv",
			},
			Spec: corev1.ServiceSpec{
				Selector: deploymentLabel,
				Type:     corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{Port: 3000, TargetPort: intstr.FromInt(3000)},
				},
			},
		}, metav1.CreateOptions{})

	// TODO: Update status in db
	// TODO: Watch status and update in db
	// TODO: register with the custom router

	rw.Write([]byte("Deploying your system."))

}

func (f *Function) CreateFunction(rw http.ResponseWriter, r *http.Request) {

	// TODO: 1. authenicate
	// TODO: 2. check if the service is enabled
	// TODO: 3. save code to db

	body := &PostCodeDTO{}

	err := fromJSON(r.Body, body)
	if err != nil {
		http.Error(rw, "cannot read json", 400)
	}

	// TODO: get these from env variables
	Registry := "ghcr.io"
	Project := ""
	ImageName := "uhquehqweoiqjeoqiwwhqodiqejd" // Code id

	imageTag := Registry + "/" + Project + "/" + ImageName + ":latest"

	namespace, err := f.kw.CreateNamespace(r.Context(), constants.Namespace)

	// create namespace if not exist
	if err != nil {
		// namespace already exists. ignore
		fmt.Println("namespace already exists. ignore")
		fmt.Printf("err: %v\n", err)
	}
	fmt.Printf("namespace: %v\n", namespace)

	// create kaniko pod

	pod, err := f.kw.CreateImageBuilder(
		&kuberneteswrapper.ImageBuilder{
			Ctx:        r.Context(),
			Namespace:  constants.Namespace,
			FunctionId: "aweqwe",         // TODO:
			Language:   constants.NODEJS, // TODO:
			ImageTag:   imageTag,
		})

	// podLogs = clientset.CoreV1().Pods("serverless").GetLogs("kaniko-worker", &v1.PodLogOptions{})

	rw.Write([]byte("Building Image for your code"))

	// watch for 1 min and then close everything
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	label, err := f.kw.BuildLabel("builder", []string{"codeID"}) // TODO:
	podWatch, err := f.kw.WatchImageBuilder(watchContext, label.String())

	go func() {
		for event := range podWatch.ResultChan() {
			p, ok := event.Object.(*corev1.Pod)
			if !ok {
				fmt.Println("unexpected type")
			}
			// Check Pod Phase. If its failed or succeeded.
			switch p.Status.Phase {
			case corev1.PodSucceeded:
				// TODO: Commit status to DB
				fmt.Println("image build success. pushed to db")
				podWatch.Stop()
				break
			case corev1.PodFailed:
				// TODO: Commit status to DB with message
				fmt.Println("Image build failed. Reason : ", p.Status.Message)
				podWatch.Stop()
				break
			}
		}
	}()

}
