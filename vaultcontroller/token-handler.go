// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vaultcontroller

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func tokenRequestHandler(w io.Writer, r *http.Request) (int, error) {
	log.Printf("token request from %s", r.RemoteAddr)
	name := r.FormValue("name")
	if name == "" {
		return 400, fmt.Errorf("missing or empty name parameter from %s", r.RemoteAddr)
	}

	namespace := r.FormValue("namespace")
	if namespace == "" {
		log.Infoln("token request: namespace missing or empty using default")
		namespace = "default"
	}

	//use kube client set to lookup the pod details by name
	pod, err := kubernetesClientSet.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return 500, fmt.Errorf("error during namespace/pod (%s)/(%s) lookup %s", namespace, name, err)
	}

	policies := pod.Annotations["vaultproject.io/policies"]

	if policies == "" {
		return 500, fmt.Errorf("error missing or empty pod vaultproject.io/role annotation (%s)", name)
	}

	ttl := pod.Annotations["vaultproject.io/ttl"]
	if ttl == "" {
		ttl = "72h"
	}

	tcr := &api.TokenCreateRequest{
		Policies: strings.Split(policies, ","),
		Metadata: map[string]string{
			"host_ip":   pod.Status.HostIP,
			"namespace": pod.Namespace,
			"pod_ip":    pod.Status.PodIP,
			"pod_name":  pod.Name,
			"pod_uid":   string(pod.UID),
		},
		DisplayName: pod.Name,
		Period:      ttl,
		NoParent:    true,
		TTL:         ttl,
	}
	secret, err := vaultClient.Auth().Token().Create(tcr)
	if err != nil {
		return 500, fmt.Errorf("error creating wrapped token for pod (%s), error: %s", name, err)
	}

	var wrappedToken bytes.Buffer
	err = json.NewEncoder(&wrappedToken).Encode(&secret.WrapInfo)
	if err != nil {
		return 500, fmt.Errorf("error parsing wrapped token for pod (%s), error: %s", name, err)
	}

	if len(pod.Spec.InitContainers) == 0 {
		return 500, errors.New("pod has no init containers")
	}

	initContainer := pod.Spec.InitContainers[0]
	if len(initContainer.Ports) == 0 {
		return 500, fmt.Errorf("init container %s has no ports section", initContainer.Name)
	}

	portStr := strconv.Itoa(int(initContainer.Ports[0].ContainerPort))

	log.Printf("Container Port in Init Container: %s", portStr)

	go pushWrappedTokenTo(pod.Status.PodIP, portStr, &wrappedToken)

	return 202, nil
}

func pushWrappedTokenTo(ip string, port string, token io.Reader) {
	url := fmt.Sprintf("https://%s:%s", ip, port)
	// initializa the client with insecure verify because vaul-init uses a self-signed certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Post(url, "", token)
	if err != nil {
		log.Printf("error pushing wrapped token to %s: %s", url, err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("error pushing wrapped token to %s: %s", url, resp.Status)
		return
	}
	log.Printf("successfully pushed wrapped token to %s", url)
}
