package main

import (
	"context"
	"fmt"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	acappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/utils/pointer"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/tabwriter"
	"time"
)

func main() {
	ctx := context.Background()

	// get kubeconfig
	config, err := getConfigOnDisk()
	if err != nil {
		panic(err)
	}

	ExampleClientSet(ctx, config)
	//ExampleRESTClient(ctx, config)
	//ExampleDiscoveryClient(ctx, config)

}

func ExampleClientSet(ctx context.Context, config *rest.Config) {
	// get clientset from kubeconfig
	clientset, _ := kubernetes.NewForConfig(config)

	// create a resource
	createdPod, _ := createResource(ctx, clientset)
	_ = createdPod

	// get info about a resource
	pod, _ := getResource(ctx, clientset, "nginx-pod")
	_ = pod

	// get a list of resources
	// namespaced
	getResourceList(ctx, clientset)

	// filter the result of a list
	filterResourceListByLabel(ctx, clientset)

	// set field selector
	filterResourceListByField(ctx, clientset)

	// delete a resource
	deleteResource(ctx, clientset, createdPod)

	// delete a collection of resources
	deleteResourceCollection(ctx, clientset)

	// update a resource
	updateResource(ctx, clientset)

	// watch a resource
	watchResource(ctx, clientset)
}

func watchResource(ctx context.Context, clientset *kubernetes.Clientset) {
	watcher, err := clientset.AppsV1().
		Deployments("default").
		Watch(
			ctx,
			metav1.ListOptions{},
		)
	if err != nil {
		panic(err)
	}

	fmt.Printf("==============================\nWatching, press Ctrl-c to exit\n==============================\n")
	for ev := range watcher.ResultChan() {
		switch v := ev.Object.(type) {
		case *appsv1.Deployment:
			fmt.Printf("%s %s\n", ev.Type, v.GetName())
		case *metav1.Status:
			fmt.Printf("%s\n", v.Status)
			watcher.Stop()
		}
	}
}

func updateResource(ctx context.Context, clientset *kubernetes.Clientset) {
	wantedDep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "app1",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}

	createdDep, err := clientset.AppsV1().Deployments("default").Create(ctx, &wantedDep, metav1.CreateOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Namespace %q not found\n", "project1")
			os.Exit(1)
		} else if errors.IsAlreadyExists(err) {
			fmt.Printf("Deployment %q already exists\n", "nginx-pod")
			os.Exit(1)
		} else if errors.IsInvalid(err) {
			fmt.Printf("Deployment specification is invalid: %v\n", err)
			os.Exit(1)
		}
		panic(err)
	}
	_ = createdDep

	wantedDep.Spec.Template.Spec.Containers[0].Name = "main"
	updatedDep, err := clientset.
		AppsV1().
		Deployments("default").
		Update(
			ctx,
			&wantedDep,
			metav1.UpdateOptions{},
		)
	if err != nil {
		if errors.IsInvalid(err) {
			fmt.Printf("Deployment specification is invalid: %v\n", err)
			os.Exit(1)
		} else if errors.IsConflict(err) {
			fmt.Printf("Conflict updating deployment %q\n", "nginx")
			os.Exit(1)
		}
		panic(err)
	}

	time.Sleep(3 * time.Second)

	// update by strategic merge patch
	// retry until no conflict
	for {
		conflict := false

		existingDep, err := clientset.
			AppsV1().
			Deployments("project1").
			Get(ctx, "nginx", metav1.GetOptions{})

		if err != nil {
			if errors.IsNotFound(err) {
				fmt.Printf("Deployment %q is not found\n", "nginx")
				os.Exit(1)
			}
			panic(err)
		}

		patch := client.StrategicMergeFrom(
			existingDep,
			client.MergeFromWithOptimisticLock{},
		)
		updatedDep2 := updatedDep.DeepCopy()
		updatedDep2.Spec.Replicas = pointer.Int32(2)
		patchData, err := patch.Data(updatedDep2)
		if err != nil {
			panic(err)
		}
		patchedDep, err := clientset.
			AppsV1().Deployments("project1").Patch(
			ctx,
			"nginx",
			patch.Type(),
			patchData,
			metav1.PatchOptions{},
		)
		if err != nil {
			if errors.IsInvalid(err) {
				fmt.Printf("Deployment specification is invalid: %v\n", err)
				os.Exit(1)
			} else if errors.IsConflict(err) {
				fmt.Printf("Conflict patching deployment %q: %v\nRetrying...\n", "nginx", err)
				conflict = true
			} else {
				panic(err)
			}
		}
		_ = patchedDep
		if !conflict {
			break
		}
		time.Sleep(1 * time.Second)
	}

	time.Sleep(3 * time.Second)

	// # Applying resources server-side with Patch
	ssaDep := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			// there is no way to indicate a default value for this field,
			// the nil value being valid,
			// so we need to indicate it
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "app1",
				},
			},
			Replicas: pointer.Int32(1), // This value is changed
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// there is no way to indicate a default value for this field,
					// the nil value being valid,
					// so we need to indicate it
					Labels: map[string]string{
						"app": "app1",
					},
				},
			},
		},
	}
	ssaDep.SetName("nginx")

	ssaDep.APIVersion, ssaDep.Kind =
		appsv1.SchemeGroupVersion.
			WithKind("Deployment").
			ToAPIVersionAndKind()

	patch := client.Apply
	patchData, err := patch.Data(&ssaDep)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(patchData))

	patchedDep, err := clientset.
		AppsV1().Deployments("project1").Patch(
		ctx,
		"nginx",
		patch.Type(),
		patchData,
		metav1.PatchOptions{
			FieldManager: "my-program",
			Force:        pointer.Bool(true),
		},
	)
	if err != nil {
		if errors.IsInvalid(err) {
			fmt.Printf("Deployment specification is invalid: %v\n", err)
			os.Exit(1)
		} else if errors.IsConflict(err) {
			fmt.Printf("Conflict server-side patching deployment %q: %v\n", "nginx", err)
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	_ = patchedDep

	time.Sleep(3 * time.Second)

	// # Server-Side Apply using Apply Configurations
	deployConfig := acappsv1.Deployment(
		"nginx",
		"project1",
	)
	deployConfig.WithSpec(acappsv1.DeploymentSpec())
	deployConfig.Spec.WithReplicas(2)

	patchedDep, err = clientset.AppsV1().
		Deployments("project1").Apply(
		ctx,
		deployConfig,
		metav1.ApplyOptions{
			FieldManager: "my-program",
			Force:        true,
		},
	)
	if err != nil {
		if errors.IsInvalid(err) {
			fmt.Printf("Deployment specification is invalid: %v\n", err)
			os.Exit(1)
		} else if errors.IsConflict(err) {
			fmt.Printf("Conflict server-side applying deployment %q: %v\n", "nginx", err)
			os.Exit(1)
		} else {
			panic(err)
		}
	}
	_ = patchedDep

}

func deleteResourceCollection(ctx context.Context, clientset *kubernetes.Clientset) {
	err := clientset.
		CoreV1().
		Pods("default").
		DeleteCollection(
			ctx,
			metav1.DeleteOptions{},
			metav1.ListOptions{},
		)
	if err != nil {
		panic(err)
	}
}

func deleteResource(ctx context.Context, clientset *kubernetes.Clientset, createdPod *corev1.Pod) {
	err := clientset.CoreV1().Pods("default").Delete(ctx, "nginx-pod", metav1.DeleteOptions{})
	if err != nil {
		panic(err)
	}
	// with grace period
	err = clientset.CoreV1().Pods("default").Delete(ctx, "nginx-pod", *metav1.NewDeleteOptions(5))
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("pod %q already deleted\n", "nginx-pod")
		} else {
			panic(err)
		}
	}
	// by uid
	uid := createdPod.GetUID()
	err = clientset.CoreV1().Pods("default").Delete(ctx, "nginx-pod", *metav1.NewPreconditionDeleteOptions(
		string(uid),
	))
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("pod %q already deleted\n", "nginx-pod")
		} else if errors.IsConflict(err) {
			fmt.Printf("Conflicting UID %q\n", string(uid))
		} else {
			panic(err)
		}
	}
	// by resource version
	rv := createdPod.GetResourceVersion()
	err = clientset.CoreV1().Pods("default").Delete(ctx, "nginx-pod", *metav1.NewRVDeletionPrecondition(
		rv,
	))
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("pod %q already deleted\n", "nginx-pod")
		} else if errors.IsConflict(err) {
			// This error will be raised, as the resource has changed due to previous deletion
			fmt.Printf("Conflicting resourceVersion %q\n", string(rv))
		} else {
			panic(err)
		}
	}

	// with propagation policy
	options := *metav1.NewDeleteOptions(5)
	policy := metav1.DeletePropagationForeground
	options.PropagationPolicy = &policy
	err = clientset.
		CoreV1().
		Pods("default").
		Delete(ctx, "nginx-pod", options)
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("pod %q already deleted\n", "nginx-pod")
		} else {
			panic(err)
		}
	}
}

func filterResourceListByField(ctx context.Context, clientset *kubernetes.Clientset) {
	// option 1: assemble one term
	fieldSelector := fields.AndSelectors(
		fields.OneTermEqualSelector("status.phase", "Running"),
		fields.OneTermNotEqualSelector("spec.restartPolicy", "Always"),
	)

	// option 2: parse
	fieldSelector, _ = fields.ParseSelector(
		"status.phase=Running,spec.restartPolicy!=Always",
	)

	// option 3: k/v set
	fieldSelector = fields.SelectorFromSet(fields.Set{
		"status.phase":       "Running",
		"spec.restartPolicy": "Never",
	})

	podList, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: fieldSelector.String(),
	})
	for _, pod := range podList.Items {
		fmt.Printf("%s\n", pod.GetName())
	}
}

func filterResourceListByLabel(ctx context.Context, clientset *kubernetes.Clientset) {
	// option 1: requirements
	req, _ := labels.NewRequirement("myKey", selection.Equals, []string{"myVal"})
	labelSelector := labels.NewSelector().Add(*req)

	// option 2: parse
	labelSelector, _ = labels.Parse(
		"myKey=myVal",
	)

	// option 3: k/v set
	labelSelector, _ = labels.ValidatedSelectorFromSet(labels.Set{
		"myKey": "myVal",
	})

	podList, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	for _, pod := range podList.Items {
		fmt.Printf("%s\n", pod.GetName())
	}
}

func getResourceList(ctx context.Context, clientset *kubernetes.Clientset) {
	podList, _ := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{})
	for _, pod := range podList.Items {
		fmt.Printf("%s\n", pod.GetName())
	}
	fmt.Println()
	// cluster-wide
	podList, _ = clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	for _, pod := range podList.Items {
		fmt.Printf("%s\n", pod.GetName())
	}
	fmt.Println()
}

func getResource(ctx context.Context, clientset *kubernetes.Clientset, name string) (*corev1.Pod, error) {
	pod, err := clientset.CoreV1().Pods("default").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Pod %q not found\n", "default")
			os.Exit(1)
		}
	}
	return pod, nil
}

func createResource(ctx context.Context, clientset *kubernetes.Clientset) (*corev1.Pod, error) {
	wantedPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-pod",
			Labels: map[string]string{
				"myKey": "myVal",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		},
	}
	createdPod, err := clientset.CoreV1().Pods("default").Create(ctx, &wantedPod, metav1.CreateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("Namespace %q not found\n", "project1")
			os.Exit(1)
		} else if errors.IsAlreadyExists(err) {
			fmt.Printf("Pod %q already exists\n", "nginx-pod")
			os.Exit(1)
		} else if errors.IsInvalid(err) {
			fmt.Printf("Pod specification is invalid\n")
			os.Exit(1)
		}
		panic(err)
	}
	_ = createdPod
	return createdPod, err
}

func ExampleRESTClient(ctx context.Context, config *rest.Config) {
	// build clientset from kubeconfig
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// get rest client from clientset
	restClient := clientset.CoreV1().RESTClient()

	// build req
	req := restClient.Get().
		Namespace("kube-system").
		Resource("pods").
		SetHeader(
			"Accept",
			fmt.Sprintf(
				"application/json;as=Table;v=%s;g=%s",
				metav1.SchemeGroupVersion.Version,
				metav1.GroupName,
			),
		)

	var res metav1.Table
	err = req.Do(ctx).Into(&res)
	if err != nil {
		panic(err)
	}

	// tab writer to format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

	// col header
	for _, colDef := range res.ColumnDefinitions {
		_, _ = fmt.Fprintf(w, "%v\t", colDef.Name)
	}
	_, _ = fmt.Fprintln(w, "")

	// cell per row
	for _, row := range res.Rows {
		for _, cell := range row.Cells {
			_, _ = fmt.Fprintf(w, "%v\t", cell)
		}
		_, _ = fmt.Fprintln(w, "")

	}

	// flush buffer to output
	err = w.Flush()
	if err != nil {
		return
	}
}

func ExampleDiscoveryClient(ctx context.Context, config *rest.Config) {
	// get discovery client from kubeconfig
	client, _ := discovery.NewDiscoveryClientForConfig(config)
	// get discovery rest mapper from discovery client
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(client))

	mapping, _ := restMapper.RESTMapping(
		schema.GroupKind{
			Group: "apps",
			Kind:  "Deployment",
		})
	fmt.Printf("single mapping: %+v\n", *mapping)
	fmt.Println()

	mappings, _ := restMapper.RESTMappings(
		schema.GroupKind{Group: "apps", Kind: "Deployment"},
	)
	for _, mapping := range mappings {
		fmt.Printf("mapping: %+v\n", *mapping)
		fmt.Println()
	}

	kinds, _ := restMapper.KindsFor(
		schema.GroupVersionResource{Group: "", Version: "", Resource: "deployment"},
	)
	fmt.Printf("kinds: %+v\n", kinds)
	fmt.Println()

	resources, _ := restMapper.ResourcesFor(
		schema.GroupVersionResource{Group: "", Version: "", Resource: "deployment"},
	)
	fmt.Printf("resources: %+v\n", resources)
	fmt.Println()
}

func getConfigInCluster() (*rest.Config, error) {
	return rest.InClusterConfig()
}

func getConfigInmem() (*rest.Config, error) {
	configBytes, err := os.ReadFile(
		"./config",
	)
	if err != nil {
		return nil, err
	}
	return clientcmd.RESTConfigFromKubeConfig(
		configBytes,
	)
}

func getConfigOnDisk() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", "./config")
}

func getConfigPersonalized() (*rest.Config, error) {
	return clientcmd.BuildConfigFromKubeconfigGetter(
		"",
		func() (*api.Config, error) {
			apiConfig, err := clientcmd.LoadFromFile(
				"/home/user/.kube/config",
			)
			if err != nil {
				return nil, nil
			}
			// TODO: manipulate apiConfig
			return apiConfig, nil
		},
	)
}

// KUBECONFIG, with a list of paths
func getConfigFromSeveral() (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		nil,
	).ClientConfig()
}

func getConfigOverrideCliFlag() (*rest.Config, error) {
	var (
		flags     pflag.FlagSet
		overrides clientcmd.ConfigOverrides
		of        = clientcmd.RecommendedConfigOverrideFlags("")
	)
	clientcmd.BindOverrideFlags(&overrides, &flags, of)
	err := flags.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&overrides,
	).ClientConfig()

}
