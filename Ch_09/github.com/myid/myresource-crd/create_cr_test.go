package main

import (
	"context"
	"errors"
	"github.com/myid/myresource-crd/pkg/apis/mygroup.example.com/v1alpha1"
	"github.com/myid/myresource-crd/pkg/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func TestCreateCR(t *testing.T) {
	type args struct {
		ctx       context.Context
		clientset *fake.Clientset
		name      string
		namespace string
		image     string
		memory    string
	}
	tests := []struct {
		name    string
		args    args
		wantCr  *v1alpha1.MyResource
		wantErr error
	}{
		{
			name: "case 1: create cr",
			args: args{
				ctx:       context.TODO(),
				clientset: fake.NewSimpleClientset(),
				name:      "myresource-crd",
				namespace: "default",
				image:     "myresource-crd",
				memory:    "1024Mi",
			},
			wantCr: &v1alpha1.MyResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "mygroup.example.com/v1alpha1",
					Kind:       "MyResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myresource-crd",
					Namespace: "default",
				},
				Spec: v1alpha1.MyResourceSpec{
					Image:  "myresource-crd",
					Memory: resource.MustParse("1024Mi"),
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCr, err := CreateCR(tt.args.ctx, tt.args.clientset, tt.args.name, tt.args.namespace, tt.args.image, tt.args.memory)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("CreateCR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotCr, tt.wantCr) {
				t.Errorf("CreateCR() gotCr = %v, want %v", gotCr, tt.wantCr)
			}
		})
	}
}
