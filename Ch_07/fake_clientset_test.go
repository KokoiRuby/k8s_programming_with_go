package Ch_07

import (
	"context"
	"errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	cgotesting "k8s.io/client-go/testing"
	"reflect"
	"testing"
)

func TestCreatePod(t *testing.T) {
	type args struct {
		ctx       context.Context
		clientset kubernetes.Interface
		name      string
		namespace string
		image     string
	}
	tests := []struct {
		name    string
		args    args
		wantPod *corev1.Pod
		wantErr error
	}{
		{
			name: "case1: check the result of function",
			args: args{
				ctx:       context.TODO(),
				clientset: fake.NewSimpleClientset(),
				name:      "a-name",
				namespace: "a-namespace",
				image:     "a-image",
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a-name",
					Namespace: "a-namespace",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "runtime",
							Image: "a-image",
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "case2: react to actions",
			args: args{
				ctx: context.TODO(),
				clientset: func() kubernetes.Interface {
					client := fake.NewSimpleClientset()
					// mutate pod Spec.NodeName
					client.Fake.PrependReactor("create", "pods", func(action cgotesting.Action) (handled bool, ret runtime.Object, err error) {
						act := action.(cgotesting.CreateAction)
						ret = act.GetObject()
						pod := ret.(*corev1.Pod)
						pod.Spec.NodeName = "node1"
						return false, pod, nil
					})
					return client
				}(),
				name:      "a-name",
				namespace: "a-namespace",
				image:     "a-image",
			},
			wantPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "a-name",
					Namespace: "a-namespace",
				},
				Spec: corev1.PodSpec{
					NodeName: "node1",
					Containers: []corev1.Container{
						{
							Name:  "runtime",
							Image: "a-image",
						},
					},
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPod, err := CreatePod(tt.args.ctx, tt.args.clientset, tt.args.name, tt.args.namespace, tt.args.image)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreatePod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPod, tt.wantPod) {
				t.Errorf("CreatePod() gotPod = %v, want %v", gotPod, tt.wantPod)
			}
		})
	}
}

func TestCreatePodCheckAction(t *testing.T) {
	var (
		name      = "a-name"
		namespace = "a-namespace"
		image     = "an-image"

		wantPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a-name",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "runtime",
						Image: "an-image",
					},
				},
			},
		}
		wantActions = 1
	)
	clientset := fake.NewSimpleClientset()
	_, _ = CreatePod(
		context.TODO(),
		clientset,
		name,
		namespace,
		image,
	)

	actions := clientset.Actions()
	if len(actions) != wantActions {
		t.Errorf("CreatePodCheckAction() got = %v, want %v", len(actions), wantActions)
	}
	action := actions[0]

	actionNamespace := action.GetNamespace()
	if actionNamespace != namespace {
		t.Errorf("action namespace = %s, want %s",
			actionNamespace,
			namespace,
		)
	}

	if !action.Matches("create", "pods") {
		t.Errorf("action verb = %s, want create",
			action.GetVerb(),
		)
		t.Errorf("action resource = %s, want pods",
			action.GetResource().Resource,
		)
	}

	createAction := action.(cgotesting.CreateAction)
	obj := createAction.GetObject()
	if !reflect.DeepEqual(obj, wantPod) {
		t.Errorf("create action object = %v, want %v",
			obj,
			wantPod,
		)
	}
}
