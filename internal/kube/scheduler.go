// Package kube is a small namespace-scoped Job reconciler client. Kubernetes
// handles pod lifecycle; the control plane independently owns authorization.
package kube

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Sanskarzz/tsunami/internal/platform"
	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

type Scheduler struct {
	Client                                       *http.Client
	Host, Namespace, Image, Broker, RuntimeClass string
	ToolImage                                    string
	tokenPath                                    string
}

// Pod termination is a Kubernetes observation. Missing API objects alone never
// prove that a process on a partitioned node has exited.
func (s *Scheduler) Observe(ctx context.Context, r platform.Run) (platform.ExecutionObservation, error) {
	var out platform.ExecutionObservation
	status, _, err := s.call(ctx, "GET", "/apis/batch/v1/namespaces/"+s.Namespace+"/jobs/run-"+r.ID, nil)
	if err != nil {
		return out, err
	}
	jobMissing := status == 404
	_, data, err := s.call(ctx, "GET", "/api/v1/namespaces/"+s.Namespace+"/pods?labelSelector="+url.QueryEscape("tsunami-run="+r.ID), nil)
	if err != nil {
		return out, err
	}
	var pods struct {
		Items []struct {
			Metadata struct {
				UID string `json:"uid"`
			} `json:"metadata"`
			Status struct {
				ContainerStatuses []struct {
					State struct {
						Running    *struct{} `json:"running"`
						Terminated *struct{} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err = json.Unmarshal(data, &pods); err != nil {
		return out, err
	}
	out.ResourcesAbsent = jobMissing && len(pods.Items) == 0
	allExited := len(pods.Items) > 0
	for _, pod := range pods.Items {
		out.PodUID = pod.Metadata.UID
		if len(pod.Status.ContainerStatuses) == 0 {
			allExited = false
		}
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Running != nil {
				out.Running = true
			}
			if container.State.Terminated == nil {
				allExited = false
			}
		}
	}
	out.Exited = allExited
	return out, nil
}

func InCluster(namespace, image, broker, runtime string) (*Scheduler, error) {
	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, errors.New("invalid Kubernetes CA")
	}
	host := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST") + ":" + os.Getenv("KUBERNETES_SERVICE_PORT")
	return &Scheduler{Client: &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}, Host: host, Namespace: namespace, Image: image, Broker: broker, RuntimeClass: runtime, ToolImage: os.Getenv("TSUNAMI_TOOL_IMAGE"), tokenPath: "/var/run/secrets/kubernetes.io/serviceaccount/token"}, nil
}

func (s *Scheduler) call(ctx context.Context, method, path string, in any) (int, []byte, error) {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return 0, nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.Host+path, body)
	if err != nil {
		return 0, nil, err
	}
	token, err := os.ReadFile(s.tokenPath)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	req.Header.Set("Content-Type", "application/json")
	client := *s.Client
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	accepted := resp.StatusCode >= 200 && resp.StatusCode < 300
	if method == http.MethodGet && resp.StatusCode == http.StatusNotFound {
		accepted = true
	}
	if method == http.MethodPost && resp.StatusCode == http.StatusConflict {
		accepted = true
	}
	if method == http.MethodDelete && resp.StatusCode == http.StatusNotFound {
		accepted = true
	}
	if !accepted {
		return resp.StatusCode, data, fmt.Errorf("Kubernetes API returned %d", resp.StatusCode)
	}
	return resp.StatusCode, data, err
}

func (s *Scheduler) Ensure(ctx context.Context, r platform.Run, token string) error {
	// Bound Kubernetes RPCs, not the lifetime of an asynchronous local worker.
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	image := s.Image
	if r.Executor != "" {
		if r.Executor != toolcontract.CurlHTTP || s.RuntimeClass != "gvisor" || s.ToolImage == "" || r.Application == nil || r.Execution != "scripted" {
			return errors.New("reviewed executor requires configured application, tool image and gvisor")
		}
		image = s.ToolImage
	}
	name := "run-" + r.ID
	secretPath := "/api/v1/namespaces/" + s.Namespace + "/secrets"
	jobsPath := "/apis/batch/v1/namespaces/" + s.Namespace + "/jobs"
	if !r.AllowJobCreation {
		status, body, err := s.call(ctx, "GET", jobsPath+"/"+name, nil)
		if err != nil {
			return err
		}
		if status == 404 {
			return platform.ErrWorkerFailed
		}
		var existing struct {
			Status struct {
				Failed int `json:"failed"`
			} `json:"status"`
		}
		if err = json.Unmarshal(body, &existing); err != nil {
			return err
		}
		if existing.Status.Failed > 0 {
			return platform.ErrWorkerFailed
		}
		return nil
	}
	labels := map[string]string{"app.kubernetes.io/part-of": "tsunami", "tsunami-role": "runner", "tsunami-run": r.ID}
	secret := map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]any{"name": name, "labels": labels}, "type": "Opaque", "stringData": map[string]string{"token": token}}
	if _, _, err := s.call(ctx, "POST", secretPath, secret); err != nil {
		return err
	}
	security := map[string]any{"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true, "capabilities": map[string]any{"drop": []string{"ALL"}}}
	pod := map[string]any{
		"restartPolicy": "Never", "serviceAccountName": "tsunami-runner", "automountServiceAccountToken": false,
		"securityContext": map[string]any{"runAsNonRoot": true, "runAsUser": 65532, "runAsGroup": 65532, "seccompProfile": map[string]string{"type": "RuntimeDefault"}},
		"containers": []any{map[string]any{"name": "agent", "image": image, "imagePullPolicy": "IfNotPresent", "args": []string{"worker"}, "securityContext": security,
			"env":          []any{map[string]string{"name": "TSUNAMI_BROKER_URL", "value": s.Broker}, map[string]string{"name": "TSUNAMI_RUN_ID", "value": r.ID}, map[string]string{"name": "TSUNAMI_TOKEN_FILE", "value": "/run/tsunami/token"}},
			"volumeMounts": []any{map[string]any{"name": "run-identity", "mountPath": "/run/tsunami", "readOnly": true}},
			"resources":    map[string]any{"requests": map[string]string{"cpu": "50m", "memory": "64Mi"}, "limits": map[string]string{"cpu": "500m", "memory": "256Mi", "ephemeral-storage": "32Mi"}}}},
		"volumes": []any{map[string]any{"name": "run-identity", "secret": map[string]any{"secretName": name, "defaultMode": 0444}}},
	}
	if r.Executor == toolcontract.CurlHTTP {
		container := pod["containers"].([]any)[0].(map[string]any)
		container["volumeMounts"] = append(container["volumeMounts"].([]any), map[string]any{"name": "adapter-tmp", "mountPath": "/tmp"})
		pod["volumes"] = append(pod["volumes"].([]any), map[string]any{"name": "adapter-tmp", "emptyDir": map[string]string{"medium": "Memory", "sizeLimit": "16Mi"}})
		pod["securityContext"].(map[string]any)["fsGroup"] = 65532
	}
	if s.RuntimeClass != "" {
		pod["runtimeClassName"] = s.RuntimeClass
	}
	job := map[string]any{"apiVersion": "batch/v1", "kind": "Job", "metadata": map[string]any{"name": name, "labels": labels}, "spec": map[string]any{"backoffLimit": 0, "activeDeadlineSeconds": 90, "ttlSecondsAfterFinished": 300, "template": map[string]any{"metadata": map[string]any{"labels": labels}, "spec": pod}}}
	status, _, err := s.call(ctx, "POST", jobsPath, job)
	if err == nil && status == 409 {
		_, body, e := s.call(ctx, "GET", jobsPath+"/"+name, nil)
		if e != nil {
			return e
		}
		var existing struct {
			Status struct {
				Failed int `json:"failed"`
			} `json:"status"`
		}
		if e = json.Unmarshal(body, &existing); e != nil {
			return e
		}
		if existing.Status.Failed > 0 {
			return platform.ErrWorkerFailed
		}
	}
	return err
}

func (s *Scheduler) Stop(ctx context.Context, id string) error {
	name := "run-" + id
	var failures []error
	for _, path := range []string{"/apis/batch/v1/namespaces/" + s.Namespace + "/jobs/" + name, "/api/v1/namespaces/" + s.Namespace + "/secrets/" + name} {
		if _, _, err := s.call(ctx, "DELETE", path, map[string]any{"apiVersion": "v1", "kind": "DeleteOptions", "propagationPolicy": "Foreground"}); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
