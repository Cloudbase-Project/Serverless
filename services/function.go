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
	corev1 "k8s.io/api/core/v1"
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
) error {
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	label, _ := kw.BuildLabel("app", []string{function.ID.String()}) // TODO:
	deploymentWatch, err := kw.GetDeploymentWatcher(
		watchContext,
		label.String(),
		namespace,
	)
	if err != nil {
		return err
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
	return nil
}

func (fs *FunctionService) WatchImageBuilder(
	kw *kuberneteswrapper.KubernetesWrapper,
	function *models.Function,
	namespace string,
) error {

	// watch for 1 min and then close everything
	watchContext, cancelFunc := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelFunc()

	label, err := kw.BuildLabel("builder", []string{function.ID.String()}) // TODO:
	podWatch, err := kw.GetImageBuilderWatcher(watchContext, label.String())

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
				fs.UpdateBuildStatus(
					UpdateBuildStatusOptions{
						Function: function,
						Status:   string(constants.BuildSuccess),
						Reason:   &p.Status.Message,
					},
				)
				break
			case corev1.PodFailed:
				// TODO: Commit status to DB with message
				fmt.Println("Image build failed. Reason : ", p.Status.Message)
				podWatch.Stop()
				fs.UpdateBuildStatus(
					UpdateBuildStatusOptions{
						Function: function,
						Status:   string(constants.BuildFailed),
						Reason:   &p.Status.Message,
					},
				)
				break
			}
		}
	}()
	<-watchContext.Done()
	var reason *string
	*reason = "Watch Timeout"
	function.BuildStatus = string(constants.BuildFailed)
	function.BuildFailReason = "Watch Timeout"
	fs.SaveFunction(function)

	return err

}
