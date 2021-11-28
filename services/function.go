package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	kuberneteswrapper "github.com/Cloudbase-Project/serverless/KubernetesWrapper"
	"github.com/Cloudbase-Project/serverless/constants"
	"github.com/Cloudbase-Project/serverless/models"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
)

type FunctionService struct {
	db *gorm.DB
	l  *log.Logger
}

func NewFunctionService(db *gorm.DB, l *log.Logger) *FunctionService {
	return &FunctionService{db: db, l: l}
}

func (fs *FunctionService) GetAllFunctions() (*models.Functions, error) {

	var functions models.Functions

	if err := fs.db.Where("userId = ?").Find(&functions).Error; err != nil {
		return nil, err
	}

	return &functions, nil
}

func (fs *FunctionService) GetFunction(codeId string) (*models.Function, error) {
	var function models.Function
	if err := fs.db.First(&function, "id = ?", codeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &function, nil
}

func (fs *FunctionService) CreateFunction(
	code string,
	language constants.Language,
	userId string,
) (*models.Function, error) {

	var function models.Function
	if err := fs.db.Create(&models.Function{Code: code, Language: string(language), UserId: userId}).Error; err != nil {
		return nil, err
	}
	return &function, nil
}

type UpdateBuildStatusOptions struct {
	Function *models.Function
	Status   string
	Reason   *string
}

func (fs *FunctionService) UpdateBuildStatus(data UpdateBuildStatusOptions) {
	data.Function.BuildStatus = data.Status
	if data.Reason != nil {
		data.Function.BuildFailReason = *data.Reason
	}
	fs.db.Save(data.Function)
}

func (fs *FunctionService) SaveFunction(function *models.Function) {
	fs.db.Save(function)
}

// Deploys a function. Creates a deployment and a clusterIP service
func (fs *FunctionService) DeployFunction(
	kw *kuberneteswrapper.KubernetesWrapper,
	ctx context.Context,
	namespace string,
	functionId string,
	label map[string]string,
	imageName string,
	replicas int32,
) error {
	// (ctx, funtionid, namespace, imagename, replicas, label)

	_, err := kw.CreateDeployment(&kuberneteswrapper.DeploymentOptions{
		Ctx:             ctx,
		Namespace:       namespace,
		FunctionId:      functionId,
		DeploymentLabel: label,
		ImageName:       imageName,
		Replicas:        replicas,
	})
	if err != nil {
		return err
	}

	// create a clusterIP service for the deployment
	_, err = kw.CreateService(&kuberneteswrapper.ServiceOptions{
		Ctx:             ctx,
		Namespace:       namespace,
		FunctionId:      functionId,
		DeploymentLabel: label,
	})
	if err != nil {
		return err
	}
	return nil
}

func (fs *FunctionService) WatchDeployment(
	kw *kuberneteswrapper.KubernetesWrapper,
	function *models.Function,
	namespace string,
) {
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	label, _ := kw.BuildLabel("app", []string{function.ID.String()}) // TODO:
	deploymentWatch, err := kw.GetDeploymentWatcher(
		watchContext,
		label.String(),
		namespace,
	)
	if err != nil {
		fs.l.Print("Error watching deployment")
	}

	go func() {
		for event := range deploymentWatch.ResultChan() {
			p, ok := event.Object.(*appsv1.Deployment)
			if !ok {
				fmt.Println("unexpected type")
			}

			if p.Status.UpdatedReplicas == *p.Spec.Replicas &&
				p.Status.Replicas == *p.Spec.Replicas &&
				p.Status.AvailableReplicas == *p.Spec.Replicas &&
				p.Status.ObservedGeneration >= p.GetObjectMeta().GetGeneration() {
				// deployment complete
				fs.l.Print("Deployment available replicas = required replicas")
				if p.Status.Conditions[0].Type == appsv1.DeploymentAvailable {
					fs.l.Print("Deployment Available")
				}
				function.DeployStatus = string(constants.Deployed)
				fs.SaveFunction(function)
				break
			} else if p.Status.Conditions[0].Type == appsv1.DeploymentProgressing {
				fs.l.Print("Deployment in Progress")
			} else if p.Status.Conditions[0].Type == appsv1.DeploymentReplicaFailure {
				fs.l.Print("Replica failure. Reason : ", p.Status.Conditions[0].Message)
				function.DeployStatus = string(constants.DeploymentFailed)
				function.DeployFailReason = p.Status.Conditions[0].Message
				fs.SaveFunction(function)
				break

			}

		}
	}()
	<-watchContext.Done()
	// Update status in db
	function.DeployStatus = string(constants.DeploymentFailed)
	function.DeployFailReason = "Watch Timeout"
	fs.SaveFunction(function)

}
