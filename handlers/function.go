package handlers

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
	"github.com/Cloudbase-Project/serverless/dtos"
	"github.com/Cloudbase-Project/serverless/services"
	"github.com/gorilla/mux"

	// appsv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type FunctionHandler struct {
	l       *log.Logger
	service *services.FunctionService
	kw      *kuberneteswrapper.KubernetesWrapper
}

// create new function
func NewFunctionHandler(
	client *kubernetes.Clientset,
	l *log.Logger,
	s *services.FunctionService,
) *FunctionHandler {
	kw := kuberneteswrapper.NewWrapper(client)
	return &FunctionHandler{l: l, service: s, kw: kw}
}

func fromJSON(body io.Reader, value interface{}) interface{} {
	d := json.NewDecoder(body)
	return d.Decode(value)
}

// Get all functions created by this user.
func (f *FunctionHandler) ListFunctions(rw http.ResponseWriter, r *http.Request) {

	functions, err := f.service.GetAllFunctions()
	if err != nil {
		http.Error(rw, "DB error", 500)
	}

	err = functions.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to marshal JSON", http.StatusInternalServerError)
	}
}

// Get a function given a "codeId" in the route params
func (f *FunctionHandler) GetFunction(rw http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	function, err := f.service.GetFunction(vars["codeId"])
	if err != nil {
		http.Error(rw, "DB error", 500)
	}

	err = function.ToJSON(rw)
	if err != nil {
		http.Error(rw, "Unable to marshal JSON", http.StatusInternalServerError)
	}
}

func (f *FunctionHandler) UpdateFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *FunctionHandler) DeleteFunction(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

func (f *FunctionHandler) GetFunctionLogs(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Not Implemented", 500)
}

// Create a deployment and a clusterIP service for the function.
// Errors if no image is found for the function
func (f *FunctionHandler) DeployFunction(rw http.ResponseWriter, r *http.Request) {

	// TODO: Get function from db.
	// check if status is complete and only then try to deploy

	deploymentLabel := map[string]string{"app": "codeId"}

	var replicas int32
	replicas = 1

	imageName := "qweqwe" // TODO:

	deployment, err := f.kw.CreateDeployment(&kuberneteswrapper.DeploymentOptions{
		Ctx:             r.Context(),
		Namespace:       constants.Namespace,
		FunctionId:      "qweqwe",
		ImageName:       imageName,
		DeploymentLabel: deploymentLabel,
		Replicas:        replicas,
	})

	// create a clusterIP service for the deployment

	service, err := f.kw.CreateService(&kuberneteswrapper.ServiceOptions{
		Ctx:             r.Context(),
		Namespace:       constants.Namespace,
		FunctionId:      "qweqwe",
		DeploymentLabel: deploymentLabel,
	})

	// TODO: Update status in db
	// TODO: Watch status and update in db
	// TODO: register with the custom router

	rw.Write([]byte("Deploying your function..."))

}

func (f *FunctionHandler) CreateFunction(rw http.ResponseWriter, r *http.Request) {

	// TODO: 1. authenicate and get userId
	// TODO: 2. check if the service is enabled
	// TODO: 3. save code to db

	var data *dtos.PostCodeDTO
	fromJSON(r.Body, data)
	if _, err := dtos.Validate(data); err != nil {
		http.Error(rw, "Validation error", 400)
		return
	}

	// Commit to db
	// TODO:
	function, err := f.service.CreateFunction(data.Code, data.Language, "userId")
	if err != nil {
		http.Error(rw, "DB error", 500)
	}

	// err := fromJSON(r.Body, body)
	// if err != nil {
	// 	http.Error(rw, "cannot read json", 400)
	// }

	// TODO: get these from env variables
	Registry := "ghcr.io"
	Project := ""

	imageName := Registry + "/" + Project + "/" + function.ID.String() + ":latest"

	namespace, err := f.kw.CreateNamespace(r.Context(), constants.Namespace)

	// create namespace if not exist
	if err != nil {
		// namespace already exists. ignore
		fmt.Println("namespace already exists. ignoring...")
		fmt.Printf("err: %v\n", err)
	}
	fmt.Printf("namespace: %v\n", namespace)

	// create kaniko pod

	pod, err := f.kw.CreateImageBuilder(
		&kuberneteswrapper.ImageBuilder{
			Ctx:        r.Context(),
			Namespace:  constants.Namespace,
			FunctionId: function.ID.String(),
			Language:   constants.Language(function.Language),
			ImageName:  imageName,
		})

	// podLogs = clientset.CoreV1().Pods("serverless").GetLogs("kaniko-worker", &v1.PodLogOptions{})

	rw.Write([]byte("Building Image for your code"))

	// watch for 1 min and then close everything
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	label, err := f.kw.BuildLabel("builder", []string{function.ID.String()}) // TODO:
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
				f.service.UpdateBuildStatus(
					services.UpdateBuildStatusOptions{
						Function: function,
						Status:   "success",
						Reason:   &p.Status.Message,
					},
				)
				break
			case corev1.PodFailed:
				// TODO: Commit status to DB with message
				fmt.Println("Image build failed. Reason : ", p.Status.Message)
				podWatch.Stop()
				f.service.UpdateBuildStatus(
					services.UpdateBuildStatusOptions{
						Function: function,
						Status:   "failed",
						Reason:   &p.Status.Message,
					},
				)
				break
			}
		}
	}()
	<-watchContext.Done()
	f.service.UpdateBuildStatus(
		services.UpdateBuildStatusOptions{Function: function, Status: "failed"},
	)

}
