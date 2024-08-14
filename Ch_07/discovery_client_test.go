package Ch_07

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func Test_checkMinimalServerVersion(t *testing.T) {
	type server struct {
		major string
		minor string
	}
	type args struct {
		clientset kubernetes.Interface
		minMinor  int
	}
	tests := []struct {
		name   string
		args   args
		server server
		min    int
		want   bool
		err    bool
	}{
		{
			name: "minimal not respected",
			args: args{
				clientset: fake.NewSimpleClientset(),
				minMinor:  10,
			},
			server: server{
				major: "1",
				minor: "9",
			},
			min:  10,
			want: false,
			err:  false,
		},
		{
			name: "minimal respected",
			args: args{
				clientset: fake.NewSimpleClientset(),
				minMinor:  10,
			},
			server: server{
				major: "1",
				minor: "11",
			},
			min:  10,
			want: true,
			err:  false,
		},
		{
			name: "version of server is unreadable",
			args: args{
				clientset: fake.NewSimpleClientset(),
				minMinor:  10,
			},
			server: server{
				major: "aze",
				minor: "11",
			},
			err: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDiscovery, ok := tt.args.clientset.Discovery().(*fakediscovery.FakeDiscovery)
			if !ok {
				t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
			}
			fakeDiscovery.FakedServerVersion = &version.Info{
				Major: tt.server.major,
				Minor: tt.server.minor,
			}
			got, err := checkMinimalServerVersion(tt.args.clientset, tt.args.minMinor)
			assert.Equal(t, tt.err, err != nil)
			assert.Equalf(t, tt.want, got, "checkMinimalServerVersion(%v, %v)", tt.args.clientset, tt.args.minMinor)
		})
	}
}
