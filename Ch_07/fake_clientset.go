package Ch_07

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreatePod(ctx context.Context, clientset kubernetes.Interface, name, namespace, image string) (pod *corev1.Pod, err error) {
	podToCreate := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "runtime",
					Image: image,
				},
			},
		},
	}
	return clientset.CoreV1().Pods(namespace).Create(ctx, &podToCreate, metav1.CreateOptions{})
}
