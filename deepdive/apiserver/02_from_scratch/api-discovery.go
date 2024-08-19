package main

import (
	"encoding/json"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

const kStatusTemplate = `{
	"apiVersion": "v1",
	"kind": "Status",
	"metadata": {},
	"status": "Failure",
	"message": "%s",
	"reason": "%s",
	"details": {"group": "mygroup.com", "kind": "MyResource", "name": "%s"},
	"code": %d
}`

// /apis returns APIGroupList or APIGroupDiscoveryList (since v1.26+)
var apiGroupList = metav1.APIGroupList{
	TypeMeta: metav1.TypeMeta{
		Kind:       "APIGroupList",
		APIVersion: "v1",
	},
	// /apis/mygroup.com
	Groups: []metav1.APIGroup{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "APIGroup",
				APIVersion: "v1",
			},
			Name: "mygroup.com",
			Versions: []metav1.GroupVersionForDiscovery{
				{GroupVersion: "mygroup.com/v1", Version: "v1"},
			},
			PreferredVersion: metav1.GroupVersionForDiscovery{
				GroupVersion: "mygroup.com/v1",
				Version:      "v1",
			},
		},
	},
}

// curl -H 'Accept: application/yaml;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList' localhost:8001/apis
// yaml → json
// no apimachinery?!
var apiGroupDiscoveryList = `{
	"apiVersion": "apidiscovery.k8s.io/v2beta1",
	"kind": "APIGroupDiscoveryList",
	"metadata": {},
	"items": [
	  {
		"metadata": {
		  "name": "mygroup.com"
		},
		"versions": [
		  {
			"version": "v1",
			"resources": [
			  {
				"resource": "myresources",
				"responseKind": {
				  "group": "mygroup.com",
				  "kind": "MyResource",
				  "version": "v1"
				},
				"scope": "Namespaced",
				"shortNames": [
				  "myres"
				],
				"singularResource": "myresource",
				"verbs": [
				  "delete",
				  "get",
				  "list",
				  "patch",
				  "create",
				  "update"
				]
			  }
			]
		  }
		]
	  }
	]
  }`

// /apis/mygroup.com/v1

var apiResourceList = metav1.APIResourceList{
	TypeMeta: metav1.TypeMeta{
		Kind:       "APIResourceList",
		APIVersion: "v1",
	},
	GroupVersion: "mygroup.com/v1",
	APIResources: []metav1.APIResource{
		{
			Name:         "myresources",
			SingularName: "myresource",
			Namespaced:   true,
			Kind:         "MyResource",
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"update",
				"patch"},
			ShortNames: []string{"myres"},
			Categories: []string{"all"},
		},
	},
}

func apis(w http.ResponseWriter, r *http.Request) {
	var gvk [3]string

	// 1.27+ kubectl discovery APIGroups and APIResourceList only by /apis with Header
	//    Accept: application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList
	// 1.27- kubectl discovery APIGroups and APIResourceList by /apis, /apis/{group}, /apis/{group}/{version}

	// resolve Accept header
	for _, acceptPart := range strings.Split(r.Header.Get("Accept"), ";") {
		// g=apidiscovery.k8s.io      g
		// v=v2beta1                  v
		// as=APIGroupDiscoveryList   k
		if pair := strings.Split(acceptPart, "="); len(pair) == 2 {
			switch pair[0] {
			case "g":
				gvk[0] = pair[1]
			case "v":
				gvk[1] = pair[1]
			case "as":
				gvk[2] = pair[1]
			}
		}
	}

	if gvk[0] == "apidiscovery.k8s.io" && gvk[2] == "APIGroupDiscoveryList" {
		w.Header().Set("Content-Type", "application/json;g=apidiscovery.k8s.io;v=v2beta1;as=APIGroupDiscoveryList")
		_, err := w.Write([]byte(apiGroupDiscoveryList))
		if err != nil {
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		renderJSON(w, apiGroupList)
	}
}

func apisGroup(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	renderJSON(w, apiGroupList.Groups[0])
}

func apisGroupVersion(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	renderJSON(w, apiResourceList)
}

func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		writeErrStatus(w, "", http.StatusInternalServerError, err.Error())
		return
	}
	_, err = w.Write(js)
	if err != nil {
		return
	}
}

func writeErrStatus(w http.ResponseWriter, name string, status int, msg string) {
	var errStatus string
	switch status {
	case http.StatusNotFound:
		errStatus = fmt.Sprintf(kStatusTemplate, fmt.Sprintf(`foos '%s' not found`, name), http.StatusText(http.StatusNotFound), name, http.StatusNotFound)
	case http.StatusConflict:
		errStatus = fmt.Sprintf(kStatusTemplate, fmt.Sprintf(`foos '%s' already exists`, name), http.StatusText(http.StatusConflict), name, http.StatusConflict)
	default:
		errStatus = fmt.Sprintf(kStatusTemplate, msg, http.StatusText(status), name, status)
	}
	w.WriteHeader(status)
	_, err := w.Write([]byte(errStatus))
	if err != nil {
		return
	}
}
