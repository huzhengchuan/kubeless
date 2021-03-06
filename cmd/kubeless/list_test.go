/*
Copyright (c) 2016-2017 Bitnami

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"

	"github.com/kubeless/kubeless/pkg/spec"
)

func fakeTprClient(f func(req *http.Request) (*http.Response, error)) *fake.RESTClient {
	return &fake.RESTClient{
		APIRegistry:          api.Registry,
		NegotiatedSerializer: api.Codecs,
		Client:               fake.CreateHTTPClient(f),
	}
}

func listOutput(t *testing.T, client rest.Interface, ns, output string, args []string) string {
	var buf bytes.Buffer

	if err := doList(&buf, client, ns, output, args); err != nil {
		t.Fatalf("doList returned error: %v", err)
	}

	return buf.String()
}

func objBody(object interface{}) io.ReadCloser {
	output, err := json.Marshal(object)
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewReader([]byte(output)))
}

func TestList(t *testing.T) {
	funcMem, _ := parseMemory("128Mi")
	listObj := spec.FunctionList{
		Items: []*spec.Function{
			{
				Metadata: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "myns",
				},
				Spec: spec.FunctionSpec{
					Handler:  "fhandler",
					Function: "ffunction",
					Runtime:  "fruntime",
					Type:     "ftype",
					Topic:    "ftopic",
					Deps:     "fdeps",
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{}},
						},
					},
				},
			},
			{
				Metadata: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "myns",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: spec.FunctionSpec{
					Handler:  "bhandler",
					Function: "bfunction",
					Runtime:  "bruntime",
					Type:     "btype",
					Topic:    "btopic",
					Deps:     "bdeps",
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Env: []v1.EnvVar{
										{
											Name:  "foo",
											Value: "bar",
										},
										{
											Name:  "foo2",
											Value: "bar2",
										},
									},
									Resources: v1.ResourceRequirements{
										Limits: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: funcMem,
										},
										Requests: map[v1.ResourceName]resource.Quantity{
											v1.ResourceMemory: funcMem,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	client := fakeTprClient(func(req *http.Request) (*http.Response, error) {
		header := http.Header{}
		header.Set("Content-Type", runtime.ContentTypeJSON)
		switch req.URL.Path {
		case "/namespaces/myns/functions":
			return &http.Response{
				StatusCode: 200,
				Header:     header,
				Body:       objBody(&listObj),
			}, nil
		case "/namespaces/myns/functions/foo":
			return &http.Response{
				StatusCode: 200,
				Header:     header,
				Body:       objBody(listObj.Items[0]),
			}, nil
		default:
			t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
			return nil, nil
		}
	})

	// No arg -> list everything in namespace
	output := listOutput(t, client, "myns", "", []string{})
	t.Log("output is", output)

	if !strings.Contains(output, "foo") || !strings.Contains(output, "bar") {
		t.Errorf("table output didn't mention both functions")
	}

	// Explicit arg(s)
	output = listOutput(t, client, "myns", "", []string{"foo"})
	t.Log("output is", output)

	if !strings.Contains(output, "foo") {
		t.Errorf("table output didn't mention explicit function foo")
	}
	if strings.Contains(output, "bar") {
		t.Errorf("table output mentions unrequested function bar")
	}

	// TODO: Actually validate the output of the following.
	// Probably need to fix output framing first.

	// json output
	output = listOutput(t, client, "myns", "json", []string{})
	t.Log("output is", output)
	if !strings.Contains(output, "foo") || !strings.Contains(output, "bar") {
		t.Errorf("table output didn't mention both functions")
	}

	// yaml output
	output = listOutput(t, client, "myns", "yaml", []string{})
	t.Log("output is", output)
	if !strings.Contains(output, "128Mi") {
		t.Errorf("table output didn't mention proper memory of function")
	}

	// wide output
	output = listOutput(t, client, "myns", "wide", []string{})
	t.Log("output is", output)
	if !strings.Contains(output, "foo = bar") {
		t.Errorf("table output didn't mention proper env of function")
	}
}
