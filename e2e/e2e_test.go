/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"testing"
	"time"

	"github.com/andreaskaris/cni-ethtool/pkg/ethtool"
	"github.com/andreaskaris/cni-ethtool/pkg/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	waite2e "sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	installerNamespace      = "default"
	installerName           = "cni-ethtool-installer"
	installerConfigMapName  = "cni-ethtool-installer-cm"
	installerImage          = "quay.io/akaris/cni-ethtool:latest"
	testDeploymentName      = "test-deployment"
	testDeploymentImageName = "quay.io/akaris/fedora:ethtool"
	privilegedPodImageName  = "quay.io/akaris/fedora:ethtool"
	installerDeployScript   = `cp /usr/local/bin/cni-ethtool /host/opt/cni/bin/cni-ethtool
	cp /etc/cni-ethtool/10-kindnet.conflist /host/etc/cni/net.d/10-kindnet.conflist
	sleep infinity`
	installerConfigurationTemplate = `{
		"cniVersion": "0.3.1",
		"name": "kindnet",
		"plugins": [
		{
		  "type": "ptp",
		  "ipMasq": false,
		  "ipam": {
			"type": "host-local",
			"dataDir": "/run/cni-ipam-state",
			"routes": [
			  { "dst": "0.0.0.0/0" }
			],
			"ranges": [
			  [ { "subnet": "10.244.0.0/24" } ]
			]
		  }
		  ,
		  "mtu": 1500
		},
		{
		  "type": "portmap",
		  "capabilities": {
			"portMappings": true
		  }
		},
		{
		  "type": "cni-ethtool",
		  "debug": true,
		  "ethtool": %s
		}
		]
	  }`
)

func TestRun(t *testing.T) {
	tcs := map[string]struct {
		es ethtool.EthtoolConfigs
	}{
		"Test EthtoolConfig 1": {
			map[string]ethtool.EthtoolConfig{
				"eth0": {
					"self": {"tx-checksumming": false, "rx-checksumming": false},
					"peer": {"tx-checksumming": false, "rx-checksumming": false},
				},
			},
		},
	}
	for desc, tc := range tcs {
		t.Run(desc, func(t *testing.T) {
			deploymentFeature := features.New("cni-ethtool normal handling").
				Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					daemonSet, cm := deployCNITool(ctx, t, cfg, installerDeployScript, generateCNIConfiguration(tc.es))
					enableEthtool(t, ctx, cfg, daemonSet)
					ctx = context.WithValue(ctx, installerName, daemonSet)
					return context.WithValue(ctx, installerConfigMapName, cm)
				}).
				Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					deployment := newDeployment(cfg.Namespace(), testDeploymentName, testDeploymentImageName, 1)
					if err := cfg.Client().Resources().Create(ctx, deployment); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).
						DeploymentAvailable(deployment.Name, deployment.Namespace), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}
					t.Logf("deployment found: %s/%s", deployment.Namespace, deployment.Name)

					return context.WithValue(ctx, testDeploymentName, deployment)
				}).
				Assess("test ethtool status inside test pods",
					func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
						// Retrieve the Deployment from context.
						dep := ctx.Value(testDeploymentName).(*appsv1.Deployment)
						selector := fmt.Sprintf("app=%s", dep.Spec.Selector.MatchLabels["app"])
						// List all pods that belong to the Deployment.
						listOption := func(lo *metav1.ListOptions) {
							lo.LabelSelector = selector
						}
						pods := &corev1.PodList{}
						err := cfg.Client().Resources(dep.Namespace).List(context.TODO(), pods, listOption)
						if err != nil || pods.Items == nil {
							t.Fatalf("error while getting pods for DaemonSet %+v, selector: %q, err: %q", dep, selector, err)
						}

						for _, pod := range pods.Items {
							// Run verification inside pods.
							verifyEthtoolSettingsInsidePod(t, ctx, cfg, pod, dep.Name, tc.es)
							// Run verification in host namespace.
							ifIndexAndES := getIFIndexesFromPod(t, ctx, cfg, pod, dep.Name, tc.es)
							verifyEthtoolSettingsOutsidePod(t, ctx, cfg, pod, dep.Name, ifIndexAndES)
						}
						return ctx
					}).
				Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					dep := ctx.Value(testDeploymentName).(*appsv1.Deployment)
					if err := cfg.Client().Resources().Delete(ctx, dep); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(dep), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}
					ds := ctx.Value(installerName).(*appsv1.DaemonSet)
					if err := cfg.Client().Resources().Delete(ctx, ds); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(ds), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}
					cm := ctx.Value(installerConfigMapName).(*corev1.ConfigMap)
					if err := cfg.Client().Resources().Delete(ctx, cm); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(cm), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}
					return ctx
				}).Feature()
			testenv.Test(t, deploymentFeature)
		})
	}
}

func TestNoEthtool(t *testing.T) {
	tcs := map[string]struct {
		es ethtool.EthtoolConfigs
	}{
		"Test EthtoolConfig 1": {
			map[string]ethtool.EthtoolConfig{
				"eth0": {
					"self": {"tx-checksumming": false, "rx-checksumming": false},
					"peer": {"tx-checksumming": false, "rx-checksumming": false},
				},
			},
		},
	}
	for desc, tc := range tcs {
		t.Run(desc, func(t *testing.T) {
			deploymentFeature := features.New("cni-ethtool handling of missing binary").
				Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					daemonSet, cm := deployCNITool(ctx, t, cfg, installerDeployScript, generateCNIConfiguration(tc.es))
					disableEthtool(t, ctx, cfg, daemonSet)
					ctx = context.WithValue(ctx, installerName, daemonSet)
					return context.WithValue(ctx, installerConfigMapName, cm)
				}).
				Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					deployment := newDeployment(cfg.Namespace(), testDeploymentName, testDeploymentImageName, 1)
					if err := cfg.Client().Resources().Create(ctx, deployment); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).
						DeploymentAvailable(deployment.Name, deployment.Namespace),
						waite2e.WithImmediate(),
						wait.WithTimeout(time.Minute*1)); err == nil {
						t.Fatal("expected to get an error with broken ethtool, got none instead")
					}
					t.Logf("deployment found: %s/%s", deployment.Namespace, deployment.Name)

					return context.WithValue(ctx, testDeploymentName, deployment)
				}).
				Assess("check node logs",
					func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
						// Retrieve the Deployment from context.
						dep := ctx.Value(testDeploymentName).(*appsv1.Deployment)
						ds := ctx.Value(installerName).(*appsv1.DaemonSet)
						checkJournal(t, ctx, cfg, ds, fmt.Sprintf(".*could not find executable.*ethtool.*%s/%s.*",
							dep.Namespace, dep.Name))
						return ctx
					}).
				Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					dep := ctx.Value(testDeploymentName).(*appsv1.Deployment)
					if err := cfg.Client().Resources().Delete(ctx, dep); err != nil {
						t.Fatal(err)
					}
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(dep), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}

					ds := ctx.Value(installerName).(*appsv1.DaemonSet)
					if err := cfg.Client().Resources().Delete(ctx, ds); err != nil {
						t.Fatal(err)
					}
					enableEthtool(t, ctx, cfg, ds)
					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(ds), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}

					cm := ctx.Value(installerConfigMapName).(*corev1.ConfigMap)
					if err := cfg.Client().Resources().Delete(ctx, cm); err != nil {
						t.Fatal(err)
					}

					if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(cm), waite2e.WithImmediate()); err != nil {
						t.Fatal(err)
					}

					return ctx
				}).Feature()
			testenv.Test(t, deploymentFeature)
		})
	}
}

func deployCNITool(ctx context.Context, t *testing.T, cfg *envconf.Config, deploySH, kindnetConfList string) (*appsv1.DaemonSet, *corev1.ConfigMap) {
	// Delete preexisting CM and create it.
	cm := newConfigMap(
		installerNamespace,
		installerConfigMapName,
		map[string]string{
			"deploy.sh":           deploySH,
			"10-kindnet.conflist": kindnetConfList,
		},
	)
	if err := cfg.Client().Resources().Delete(ctx, cm); err != nil && !errors.IsNotFound(err) {
		t.Fatal(err)
	}
	if err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(cm), wait.WithTimeout(time.Minute*1)); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Client().Resources().Create(ctx, cm); err != nil {
		t.Fatal(err)
	}

	// Delete preexisting DS and create it.
	daemonSet := newInstallerDaemonset(installerNamespace, installerName, installerImage, cm.Name)
	if err := cfg.Client().Resources().Delete(ctx, daemonSet); err != nil && !errors.IsNotFound(err) {
		t.Fatal(err)
	}
	if err := wait.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(daemonSet), wait.WithTimeout(time.Minute*1)); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Client().Resources().Create(ctx, daemonSet); err != nil {
		t.Fatal(err)
	}
	if err := waitForDaemonSet(cfg, installerNamespace, installerName); err != nil {
		t.Fatal(err)
	}
	return daemonSet, cm
}

func newDeployment(namespace, name, imageName string, replicaCount int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"app": "test-app"}},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test-app"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           imageName,
							ImagePullPolicy: corev1.PullNever,
							Command: []string{
								"sleep",
								"infinity",
							},
						},
					},
				},
			},
		},
	}
}

func newInstallerDaemonset(namespace, name, image, configMapName string) *appsv1.DaemonSet {
	labels := map[string]string{"app": name}
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           image,
							ImagePullPolicy: corev1.PullNever,
							Command: []string{
								"/bin/bash",
								"/etc/cni-ethtool/deploy.sh",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "host", MountPath: "/host"},
								{Name: "config", MountPath: "/etc/cni-ethtool"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "host",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{Path: "/"},
							},
						},
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
									DefaultMode:          pointer.Int32(0420),
								},
							},
						},
					},
				},
			},
		},
	}
}

func waitForDaemonSet(cfg *envconf.Config, namespace, name string) error {
	ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	return waite2e.For(conditions.New(cfg.Client().Resources()).ResourceMatch(
		&ds,
		func(object k8s.Object) bool {
			ds := object.(*appsv1.DaemonSet)
			return isDaemonSetReady(ds)
		}))
}

func isDaemonSetReady(ds *appsv1.DaemonSet) bool {
	n := ds.Status.DesiredNumberScheduled
	return n == ds.Status.CurrentNumberScheduled && n == ds.Status.NumberReady && n == ds.Status.NumberAvailable
}

func newConfigMap(namespace, name string, data map[string]string) *corev1.ConfigMap {
	labels := map[string]string{"app": name}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Data:       data,
	}
}

func newPrivilegedPod(namespace, name, node, image string) *corev1.Pod {
	labels := map[string]string{"app": name}
	mountPropagation := corev1.MountPropagationHostToContainer
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			NodeName:    node,
			Containers: []corev1.Container{
				{
					Name:            name,
					Image:           image,
					ImagePullPolicy: corev1.PullNever,
					Command: []string{
						"sleep",
						"infinity",
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "host", MountPath: "/host"},
						{Name: "netns", MountPath: "/run/netns", MountPropagation: &mountPropagation},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "host",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{Path: "/"},
					},
				},
				{
					Name: "netns",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{Path: "/run/netns"},
					},
				},
			},
		},
	}
}

func generateCNIConfiguration(es ethtool.EthtoolConfigs) string {
	return fmt.Sprintf(installerConfigurationTemplate, es.String())
}

func parseEthtoolOutput(out, field string) (bool, error) {
	re, err := regexp.Compile(fmt.Sprintf("(%s): (on|off)", field))
	if err != nil {
		return false, err
	}
	subMatches := re.FindStringSubmatch(out)
	if len(subMatches) == 0 {
		return false, fmt.Errorf("could not find field %q", field)
	}
	if len(subMatches) != 3 || (subMatches[2] != "on" && subMatches[2] != "off") {
		return false, fmt.Errorf("unexpected output for field %q, got %v", field, subMatches)
	}
	if subMatches[2] == "on" {
		return true, nil
	}
	return false, nil
}

func verifyEthtoolSettingsInsidePod(t *testing.T, ctx context.Context, cfg *envconf.Config, pod corev1.Pod, containerName string, es ethtool.EthtoolConfigs) {
	for intf, e := range es {
		cmd := fmt.Sprintf(`ethtool -k %s`, intf)
		var stdout, stderr bytes.Buffer
		command := []string{"/bin/bash", "-c", cmd}
		if err := cfg.Client().Resources().ExecInPod(ctx, pod.Namespace, pod.Name, containerName, command, &stdout, &stderr); err != nil {
			t.Log(stderr.String())
			t.Fatal(err)
		}
		for parameter, state := range e.GetSelf() {
			if outState, err := parseEthtoolOutput(stdout.String(), parameter); err != nil || state != outState {
				t.Fatalf("received invalid state for pod %s/%s, interface %q and parameter %q, "+
					"expected state: %t, got state: %t, got err: %q",
					pod.Namespace, pod.Name, intf, parameter, state, outState, err)
			}
		}
	}
}

func getIFIndexesFromPod(t *testing.T, ctx context.Context, cfg *envconf.Config, pod corev1.Pod, containerName string, es ethtool.EthtoolConfigs) map[int]ethtool.EthtoolConfig {
	ifIndexes := map[int]ethtool.EthtoolConfig{}
	iplinks := []IPLink{}
	for intf, e := range es {
		cmd := fmt.Sprintf(`ip --json link ls dev %s`, intf)
		var stdout, stderr bytes.Buffer
		command := []string{"/bin/bash", "-c", cmd}
		if err := cfg.Client().Resources().ExecInPod(ctx, pod.Namespace, pod.Name, containerName, command, &stdout, &stderr); err != nil {
			t.Log(stderr.String())
			t.Fatal(err)
		}
		err := json.Unmarshal(stdout.Bytes(), &iplinks)
		if err != nil {
			t.Fatal(err)
		}
		if len(iplinks) != 1 {
			t.Fatalf("unexpected length of iplinks for interface %s, got %v", intf, iplinks)
		}
		ifIndexes[iplinks[0].IFIndex] = e
	}
	return ifIndexes
}

func verifyEthtoolSettingsOutsidePod(t *testing.T, ctx context.Context, cfg *envconf.Config, pod corev1.Pod, containerName string, ifIndexAndES map[int]ethtool.EthtoolConfig) {
	// Spawn a privileged pod on the same node as the pod and wait until it's ready.
	privilegedPod := newPrivilegedPod(cfg.Namespace(), pod.Spec.NodeName, pod.Spec.NodeName, privilegedPodImageName)
	if err := cfg.Client().Resources().Create(ctx, privilegedPod); err != nil {
		t.Fatalf("could not create privileged pod %s/%s on node %s, err: %q",
			privilegedPod.Namespace, privilegedPod.Name, privilegedPod.Spec.NodeName, err)
	}
	if err := waite2e.For(conditions.New(cfg.Client().Resources()).
		PodReady(privilegedPod), waite2e.WithImmediate()); err != nil {
		t.Fatal(err)
	}

	// First, get the pod ID.
	cmd := fmt.Sprintf(`chroot /host crictl pods -q --namespace %s --name %s`, pod.Namespace, pod.Name)
	command := []string{"/bin/bash", "-c", cmd}
	var stdout, stderr bytes.Buffer
	if err := cfg.Client().Resources().ExecInPod(ctx, privilegedPod.Namespace, privilegedPod.Name,
		privilegedPod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
		t.Log(stderr.String())
		t.Fatal(err)
	}
	podID := stdout.String()
	t.Logf("pod id is %s", podID)

	// Now, inspect the pod and extract the pod net namespace.
	cmd = fmt.Sprintf(`chroot /host crictl inspectp -o json %s`, podID)
	command = []string{"/bin/bash", "-c", cmd}
	stdout = bytes.Buffer{}
	stderr = bytes.Buffer{}
	if err := cfg.Client().Resources().ExecInPod(ctx, privilegedPod.Namespace, privilegedPod.Name,
		privilegedPod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
		t.Log(stderr.String())
		t.Fatal(err)
	}
	var podInspect PodInspect
	if err := json.Unmarshal(stdout.Bytes(), &podInspect); err != nil {
		t.Fatal(err)
	}
	netns := podInspect.GetNetns()
	t.Logf("namespace id is %s", netns)

	// Now, list all interfaces. and find the one that has the netns and if index.
	cmd = `chroot /host ip -o link ls`
	command = []string{"/bin/bash", "-c", cmd}
	stdout = bytes.Buffer{}
	stderr = bytes.Buffer{}
	if err := cfg.Client().Resources().ExecInPod(ctx, privilegedPod.Namespace, privilegedPod.Name,
		privilegedPod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
		t.Log(stderr.String())
		t.Fatal(err)
	}
	t.Logf(stdout.String())

	// Now, find the interface with the netns and if index that we are looking for.
	for i, es := range ifIndexAndES {
		expr := fmt.Sprintf("[0-9]+: ([a-zA-Z0-9]+)@if%d:.*(%s)", i, netns)
		t.Logf(expr)
		re, err := regexp.Compile(expr)
		if err != nil {
			t.Fatal(err)
		}
		subMatches := re.FindStringSubmatch(stdout.String())
		if len(subMatches) != 3 {
			t.Fatalf("could not find matching interface for namespace %q, got: %v", netns, subMatches)
		}
		intf := subMatches[1]
		t.Logf("interface name %s", intf)

		// Now, list ethtool settings for that interface.
		cmd = fmt.Sprintf(`ethtool -k %s`, intf)
		command = []string{"/bin/bash", "-c", cmd}
		stdout = bytes.Buffer{}
		stderr = bytes.Buffer{}
		if err := cfg.Client().Resources().ExecInPod(ctx, privilegedPod.Namespace, privilegedPod.Name,
			privilegedPod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
			t.Log(stderr.String())
			t.Fatal(err)
		}
		for parameter, state := range es.GetPeer() {
			if outState, err := parseEthtoolOutput(stdout.String(), parameter); err != nil || state != outState {
				t.Fatalf("received invalid state for pod %s/%s, interface %q and parameter %q, "+
					"expected state: %t, got state: %t, got err: %q",
					pod.Namespace, pod.Name, intf, parameter, state, outState, err)
			}
		}
	}

	// Delete the privileged pod.
	if err := cfg.Client().Resources().Delete(ctx, privilegedPod); err != nil {
		t.Fatal(err)
	}
	if err := waite2e.For(conditions.New(cfg.Client().Resources()).ResourceDeleted(privilegedPod), waite2e.WithImmediate()); err != nil {
		t.Fatal(err)
	}
}

func enableEthtool(t *testing.T, ctx context.Context, cfg *envconf.Config, daemonset *appsv1.DaemonSet) {
	t.Log("enabling ethtool")
	modifyEthtool(t, ctx, cfg, daemonset, true)
}

func disableEthtool(t *testing.T, ctx context.Context, cfg *envconf.Config, daemonset *appsv1.DaemonSet) {
	t.Log("disabling ethtool")
	modifyEthtool(t, ctx, cfg, daemonset, false)
}

func modifyEthtool(t *testing.T, ctx context.Context, cfg *envconf.Config, daemonset *appsv1.DaemonSet, enable bool) {
	// List all pods that belong to the DaemonSet.
	listOption := func(lo *metav1.ListOptions) {
		lo.LabelSelector = fmt.Sprintf("app=%s", daemonset.Spec.Selector.MatchLabels["app"])
		t.Logf("listing all pods with LabelSelector %q", lo.LabelSelector)
	}
	pods := &corev1.PodList{}
	if err := cfg.Client().Resources(daemonset.Namespace).List(ctx, pods, listOption); err != nil {
		t.Fatal(err)
	}

	// Define command to run.
	cmd := `if [ -f /host/sbin/ethtool ]; then mv /host/sbin/ethtool /host/sbin/ethtool.back; fi`
	if enable {
		cmd = `if [ -f /host/sbin/ethtool.back ]; then mv /host/sbin/ethtool.back /host/sbin/ethtool; fi`
	}
	command := []string{"/bin/bash", "-c", cmd}
	var stdout, stderr bytes.Buffer

	// Run command in all pods of DaemonSet.
	for _, pod := range pods.Items {
		stdout = bytes.Buffer{}
		stderr = bytes.Buffer{}
		if err := cfg.Client().Resources().ExecInPod(ctx, pod.Namespace, pod.Name,
			pod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
			t.Log(stderr.String())
			t.Fatal(err)
		}
	}
}

func checkJournal(t *testing.T, ctx context.Context, cfg *envconf.Config, daemonset *appsv1.DaemonSet, expression string) {
	re, err := regexp.Compile(expression)
	if err != nil {
		t.Fatal(err)
	}

	// List all pods that belong to the DaemonSet.
	listOption := func(lo *metav1.ListOptions) {
		lo.LabelSelector = fmt.Sprintf("app=%s", daemonset.Spec.Selector.MatchLabels["app"])
		t.Logf("listing all pods with LabelSelector %q", lo.LabelSelector)
	}
	pods := &corev1.PodList{}
	if err := cfg.Client().Resources(daemonset.Namespace).List(ctx, pods, listOption); err != nil {
		t.Fatal(err)
	}

	// Define command to run.
	cmd := `chroot /host journalctl --since="5 minutes ago"`
	command := []string{"/bin/bash", "-c", cmd}
	var stdout, stderr bytes.Buffer

	// Run command in all pods of DaemonSet.
	matched := false
	for _, pod := range pods.Items {
		stdout = bytes.Buffer{}
		stderr = bytes.Buffer{}
		if err := cfg.Client().Resources().ExecInPod(ctx, pod.Namespace, pod.Name,
			pod.Spec.Containers[0].Name, command, &stdout, &stderr); err != nil {
			t.Log(stderr.String())
			t.Fatal(err)
		}
		//t.Logf("journal for node %s, looking for expression %s:\n%s", pod.Spec.NodeName, expression, stdout.String())
		if re.Match(stdout.Bytes()) {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("could not find regular expression %q in journals", expression)
	}
}

type IPLink struct {
	IFIndex int
}

type Namespace struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type PodInspect struct {
	Info struct {
		RuntimeSpec struct {
			Linux struct {
				Namespaces []Namespace `json:"namespaces"`
			} `json:"linux"`
		} `json:"runtimeSpec"`
	} `json:"info"`
}

// GetNetns returns the netns (only the name, without the full path) of this pod.
func (p PodInspect) GetNetns() string {
	for _, ns := range p.Info.RuntimeSpec.Linux.Namespaces {
		if ns.Type == helpers.TypeNetwork {
			return path.Base(ns.Path)
		}
	}
	return ""
}
