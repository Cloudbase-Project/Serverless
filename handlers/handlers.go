package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Cloudbase-Project/serverless/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PostCodeDTO struct {
	code     string
	language Language
}

func fromJSON(body io.Reader, value interface{}) interface{} {
	d := json.NewDecoder(body)
	return d.Decode(value)
}

type Language string

const (
	NODEJS Language = "NODEJS"
	GOLANG Language = "GOLANG"
)

func CodeHandler(rw http.ResponseWriter, r *http.Request) {

	// 1. authenicate
	// 2. check if the service is enabled
	// 3. save code to db

	body := &PostCodeDTO{}

	err := fromJSON(r.Body, body)
	if err != nil {
		http.Error(rw, "cannot read json", 400)
	}

	/*
		Build image
			stream the data to the frontend
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

	if body.language == NODEJS {

	}

	// create namespace if not exist
	namespace, err := clientset.CoreV1().
		Namespaces().
		Create(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "serverless"}}, metav1.CreateOptions{})
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

	pod, err := clientset.CoreV1().Pods("serverless").Create(context.Background(), &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "kaniko-worker",
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

}
