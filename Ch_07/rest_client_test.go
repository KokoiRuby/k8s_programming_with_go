package Ch_07

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"net/http"
	"testing"
)

func Test_getPods(t *testing.T) {
	type args struct {
		ctx        context.Context
		restClient rest.Interface
		ns         string
	}
	tests := []struct {
		name           string
		args           args
		wantErr        error
		wantStatusCode int32
	}{
		{
			name: "test1",
			args: args{
				ctx: context.TODO(),
				restClient: &fake.RESTClient{
					GroupVersion:         corev1.SchemeGroupVersion,
					NegotiatedSerializer: scheme.Codecs,
					Err:                  errors.New("an error from the rest client"),
				},
				ns: "default",
			},
			wantErr: errors.New("an error from the rest client"),
		},
		{
			name: "test2",
			args: args{
				ctx: context.TODO(),
				restClient: &fake.RESTClient{
					GroupVersion:         corev1.SchemeGroupVersion,
					NegotiatedSerializer: scheme.Codecs,
					Err:                  nil,
					Resp: &http.Response{
						StatusCode: http.StatusNotFound,
					},
				},
				ns: "default",
			},
			wantErr:        nil,
			wantStatusCode: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getPods(tt.args.ctx, tt.args.restClient, tt.args.ns)
			if errors.Is(err, tt.wantErr) {
				t.Errorf("getPods() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.name == "test2" {
				var status *kerrors.StatusError
				ok := errors.As(err, &status)
				if !ok {
					t.Errorf("err should be of type errors.StatusError")
				}
				code := status.Status().Code
				assert.Equal(t, tt.wantStatusCode, code)
			}
		})
	}
}
