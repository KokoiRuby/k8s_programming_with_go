package main

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type MyResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec struct {
		// Msg says hello world!
		Msg string `json:"msg"`
		// Msg1 provides verbose information
		Msg1 string `json:"msg1"`
	} `json:"spec"`
}
