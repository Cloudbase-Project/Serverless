package kuberneteswrapper

import (
	"context"

	"k8s.io/client-go/kubernetes"

	// appsv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesWrapper struct {
	KClient *kubernetes.Clientset
}

func NewWrapper(client *kubernetes.Clientset) *KubernetesWrapper {
	return &KubernetesWrapper{KClient: client}
}

func (kw *KubernetesWrapper) CreateImageBuilder(ctx context.Context, namespace string) {
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
