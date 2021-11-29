package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	kuberneteswrapper "github.com/Cloudbase-Project/serverless/KubernetesWrapper"
	"github.com/Cloudbase-Project/serverless/constants"
	"github.com/Cloudbase-Project/serverless/dtos"
	"github.com/Cloudbase-Project/serverless/services"
	"github.com/Cloudbase-Project/serverless/utils"
	"github.com/gorilla/mux"

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
	// set status to readyToDeploy
	// set LastAction to update

	vars := mux.Vars(r)

	var data *dtos.UpdateCodeDTO
	fromJSON(r.Body, data)

	if _, err := dtos.Validate(data); err != nil {
		http.Error(rw, "Validation error", 400)
		return
	}

	// get the function.
	function, err := f.service.GetFunction(vars["codeId"])
	if err != nil {
		http.Error(rw, "DB error", 500)
	}

	// update the code.
	function.Code = data.Code
	function.BuildStatus = string(constants.Building)
	// save it
	f.service.SaveFunction(function)

	imageName := utils.BuildImageName(function.ID.String())

	// build image
	f.kw.CreateImageBuilder(&kuberneteswrapper.ImageBuilder{
		Ctx:        r.Context(),
		Namespace:  constants.Namespace,
		FunctionId: function.ID.String(),
		Language:   constants.Language(function.Language),
		ImageName:  imageName,
	})

	rw.Write([]byte("Building new image for your updated code"))

	result := f.service.WatchImageBuilder(f.kw, function, constants.Namespace)
	if result.Err != nil {
		http.Error(rw, "Error watching image builder", 500)
	}

	function.BuildFailReason = result.Reason
	function.BuildStatus = result.Status
	// TODO: Should Come back to this. maybe have to add lastAction  = update
	function.LastAction = string(constants.UpdateAction)
	function.DeployStatus = string(constants.RedeployRequired)
	f.service.SaveFunction(function)

}

func (f *FunctionHandler) DeleteFunction(rw http.ResponseWriter, r *http.Request) {
	// get function from db
	vars := mux.Vars(r)

	codeId := vars["codeId"]

	function, err := f.service.GetFunction(codeId)
	if err != nil {
		http.Error(rw, "DB error", 500)
	}
	// delete it.
	err = f.service.DeleteFunction(codeId)
	if err != nil {
		f.l.Print(err)
		http.Error(rw, "DB error", 500)
	}
	// TODO: delete resources
	serviceName := utils.BuildServiceName(codeId)

	err = f.service.DeleteFunctionResources(
		f.kw,
		context.Background(),
		constants.Namespace,
		codeId,
		serviceName,
	)
	if err != nil {
		f.l.Print(err)
		http.Error(rw, "Err deleting resources", 500)
	}
	// TODO: Remove from router
}

func (f *FunctionHandler) GetFunctionLogs(rw http.ResponseWriter, r *http.Request) {
	// get function
	// check if its deployStatus is deployed
	// check if lastAction is deploy
	// check if
}

// Create a deployment and a clusterIP service for the function.
// Errors if no image is found for the function
func (f *FunctionHandler) DeployFunction(rw http.ResponseWriter, r *http.Request) {

	//  Get function from db.
	vars := mux.Vars(r)

	function, err := f.service.GetFunction(vars["codeId"])
	if err != nil {
		http.Error(rw, "DB error", 500)
	}

	if function.BuildStatus == string(constants.BuildSuccess) &&
		function.DeployStatus == string(constants.NotDeployed) &&
		function.LastAction == string(constants.BuildAction) {
		// proceed

		deploymentLabel := map[string]string{"app": function.ID.String()}

		// TODO: Should change to constant
		replicas := int32(1)

		imageName := utils.BuildImageName(function.ID.String())

		err = f.service.DeployFunction(
			f.kw,
			r.Context(),
			constants.Namespace,
			function.ID.String(),
			deploymentLabel,
			imageName,
			replicas,
		)
		if err != nil {
			http.Error(rw, "Error deploying your image.", 500)
			return
		}

		// update status in db
		function.DeployStatus = string(constants.Deploying)
		f.service.SaveFunction(function)

		rw.Write([]byte("Deploying your function..."))

		// Watch status
		// watch for 1 min and then close everything

		result := f.service.WatchDeployment(f.kw, function, constants.Namespace)
		if result.Err != nil {
			http.Error(rw, "Error watching deployment", 500)
		}

		function.DeployFailReason = result.Reason
		function.DeployStatus = result.Status
		f.service.SaveFunction(function)

		// TODO: register with the custom router

	} else {
		http.Error(rw, "Cannot perform this action currently", 400)
	}

	// if function.LastAction == string(constants.UpdateAction) ||
	// 	function.LastAction == string(constants.DeployAction) {
	// 	http.Error(rw, "Function Already deployed or must be redeployed", 400)
	// }

	// // check if status is complete and only then try to deploy
	// if (function.BuildStatus == string(constants.BuildFailed)) ||
	// 	(function.DeployStatus == string(constants.RedeployRequired)) ||
	// 	(function.DeployStatus == string(constants.Deployed)) ||
	// 	(function.LastAction == string(constants.UpdateAction)) {
	// 	http.Error(rw, "Cannot perform this action currently.", 400)
	// 	return
	// }

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

	_, err = f.kw.CreateImageBuilder(
		&kuberneteswrapper.ImageBuilder{
			Ctx:        r.Context(),
			Namespace:  constants.Namespace,
			FunctionId: function.ID.String(),
			Language:   constants.Language(function.Language),
			ImageName:  imageName,
		})

	// podLogs = clientset.CoreV1().Pods("serverless").GetLogs("kaniko-worker", &v1.PodLogOptions{})

	rw.Write([]byte("Building Image for your code"))

	result := f.service.WatchImageBuilder(f.kw, function, constants.Namespace)
	if result.Err != nil {
		http.Error(rw, "Error watching image builder", 500)
	}

	function.BuildFailReason = result.Reason
	function.BuildStatus = result.Status
	f.service.SaveFunction(function)

}
