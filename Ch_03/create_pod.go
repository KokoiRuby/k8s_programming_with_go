package main

import (
	"bytes"
	bjson "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"net/http"
)

func main() {
	err := createPod()
	if err != nil {
		return
	}
}

func createPod() error {
	pod := NewPod()
	serializer := getJSONSerializer()
	postBody, err := serializePodObject(serializer, pod)
	if err != nil {
		return err
	}
	reqCreate, err := buildPostRequest(postBody)
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(reqCreate)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 300 {
		createdPod, err := deserializePodBody(serializer, body)
		if err != nil {
			return err
		}
		json, err := bjson.MarshalIndent(createdPod, "", " ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", json)
	} else {
		status, err := deserializeStatusBody(serializer, body)
		if err != nil {
			return err
		}
		json, err := bjson.MarshalIndent(status, "", " ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", json)
	}

	return nil
}

// NewPod constructor
func NewPod() *corev1.Pod {
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "runtime",
					Image: "nginx",
				},
			},
		},
	}
	pod.SetName("my-pod")
	pod.SetLabels(map[string]string{
		"app.kubernetes.io/name":    "my-component",
		"app.kubernetes.io/part-of": "my-pod",
	})
	return &pod
}

// serializePodObject pod struct → JSON
func serializePodObject(serializer runtime.Serializer, pod *corev1.Pod) (io.Reader, error) {
	var buf bytes.Buffer
	err := serializer.Encode(pod, &buf)
	if err != nil {
		return nil, err
	}
	return &buf, nil
}

// buildPostRequest build post request & ++ headers
func buildPostRequest(body io.Reader) (*http.Request, error) {
	reqCreate, err := http.NewRequest("POST", "http://127.0.0.1:8001/api/v1/namespaces/default/pods", body)
	if err != nil {
		return nil, err
	}
	reqCreate.Header.Add("Accept", "application/json")
	reqCreate.Header.Add("Content-Type", "application/json")
	return reqCreate, nil
}

// deserializePodBody JSON → Pod struct
func deserializePodBody(serializer runtime.Serializer, body []byte) (*corev1.Pod, error) {
	var res corev1.Pod
	_, _, err := serializer.Decode(body, nil, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// deserializeStatusBody JSON → meta struct
func deserializeStatusBody(serializer runtime.Serializer, body []byte) (*metav1.Status, error) {
	var status metav1.Status
	_, _, err := serializer.Decode(body, nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// getJSONSerializer create a Serializer with scheme so that it can resolve
func getJSONSerializer() runtime.Serializer {
	scheme := runtime.NewScheme()
	// add type to scheme
	scheme.AddKnownTypes(schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}, &corev1.Pod{}, &metav1.Status{})
	return json.NewSerializerWithOptions(json.SimpleMetaFactory{}, nil, scheme, json.SerializerOptions{})
}
