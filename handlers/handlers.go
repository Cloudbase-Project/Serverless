package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Cloudbase-Project/serverless/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PostCodeDTO struct {
	code     string
	language constants.Language
}

func fromJSON(body io.Reader, value interface{}) interface{} {
	d := json.NewDecoder(body)
	return d.Decode(value)
}

func ListFunctions(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func UpdateFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func DeleteFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func GetFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func GetFunctionLogs(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func DeployFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func CreateFunction(rw http.ResponseWriter, r *http.Request) {

	// TODO: 1. authenicate
	// TODO: 2. check if the service is enabled
	// TODO: 3. save code to db

	body := &PostCodeDTO{}

	err := fromJSON(r.Body, body)
	if err != nil {
		http.Error(rw, "cannot read json", 400)
	}

	/*
		Build image
	*/

	// builder := ImageBuilder{}
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	if body.language == constants.NODEJS {

	}

	// create namespace if not exist
	namespace, err := clientset.CoreV1().
		Namespaces().
		Create(r.Context(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "serverless"}}, metav1.CreateOptions{})
	if err != nil {
		// namespace already exists. ignore
		fmt.Println("namespace already exists. ignore")
		fmt.Printf("err: %v\n", err)
	}
	fmt.Printf("namespace: %v\n", namespace)

	// create kaniko pod

	Registry := "ghcr.io"
	Project := ""
	ImageName := "uhquehqweoiqjeoqiwwhqodiqejd" // Code id

	imageTag := Registry + "/" + Project + "/" + ImageName + ":latest"

	pod, err := clientset.CoreV1().Pods("serverless").Create(r.Context(), &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kaniko-worker",
			Labels: map[string]string{
				"builder": "codeId", // the code id
			},
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{{
				Name:  "setup-kaniko",
				Image: "yauritux/busybox-curl",
				Command: []string{
					"/bin/sh",
					"-c",
					"curl -XGET http://cloudbase-serverless-srv.default:3000/worker/queue -o /workspace/index.js && echo -e " + constants.NodejsDockerfile + " >> /workspace/Dockerfile && echo -e " + constants.NodejsPackageJSON + " >> /workspace/package.json && echo -e " + constants.RegistryCredentials + " >> /kaniko/.docker/config.json ",
				},
				VolumeMounts: []v1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerConfig",
					MountPath: "/kaniko/.docker",
				}},
			}},
			Containers: []v1.Container{{
				Name:  "kaniko-executor",
				Image: "gcr.io/kaniko-project/executor:latest",
				Args: []string{
					"--dockerfile=/workspace/Dockerfile",
					"--context=dir:///workspace",
					// "--no-push",
					"--destination=" + imageTag,
				},
				VolumeMounts: []v1.VolumeMount{{
					Name:      "shared",
					MountPath: "/workspace",
				}, {
					Name:      "dockerConfig",
					MountPath: "/kaniko/.docker",
				}},
			}},
			RestartPolicy: v1.RestartPolicyNever,
			Volumes: []v1.Volume{{
				Name: "shared", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
			}, {
				Name: "dockerConfig", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}},
			}},
		},
	}, metav1.CreateOptions{})

	// podLogs = clientset.CoreV1().Pods("serverless").GetLogs("kaniko-worker", &v1.PodLogOptions{})

	rw.Write([]byte("Building Image for your code"))

	label := ""
	for k := range pod.GetLabels() {
		label = k
		break
	}

	// watch for 1 min and then close everything
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	podWatch, err := clientset.CoreV1().
		Pods("serverless").
		Watch(
			// TODO: Donno if the request context should be used here or a custom timeout context should be used here.
			// r.Context(),
			watchContext,
			metav1.ListOptions{LabelSelector: label})

	go func() {
		for event := range podWatch.ResultChan() {
			p, ok := event.Object.(*v1.Pod)
			if !ok {
				fmt.Println("unexpected type")
			}
			// Check Pod Phase. If its failed or succeeded.
			switch p.Status.Phase {
			case v1.PodSucceeded:
				// TODO: Commit status to DB
				fmt.Println("image build success. pushed to db")
				podWatch.Stop()
				break
			case v1.PodFailed:
				// TODO: Commit status to DB with message
				fmt.Println("Image build failed. Reason : ", p.Status.Message)
				podWatch.Stop()
				break
			}
		}
	}()

}
