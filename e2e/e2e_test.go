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
	"fmt"
	"testing"

	"github.com/andreaskaris/cni-ethtool/pkg/ethtool"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/e2e-framework/klient/k8s"
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
			deploymentFeature := features.New("veth-ethtool").
				Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
					cm := newConfigMap(
						installerNamespace,
						installerConfigMapName,
						map[string]string{
							"deploy.sh":           installerDeployScript,
							"10-kindnet.conflist": generateCNIConfiguration(tc.es),
						},
					)
					if err := cfg.Client().Resources().Create(ctx, cm); err != nil {
						t.Fatal(err)
					}

					daemonSet := newInstallerDaemonset(installerNamespace, installerName, installerImage, cm.Name)
					if err := cfg.Client().Resources().Create(ctx, daemonSet); err != nil {
						t.Fatal(err)
					}
					if err := waitForDaemonSet(cfg, installerNamespace, installerName); err != nil {
						t.Fatal(err)
					}

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
				Assess("test ethtool status in host namespace",
					func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
						// Retrieve the DaemonSet from context.
						ds := ctx.Value(installerName).(*appsv1.DaemonSet)
						selector := fmt.Sprintf("app=%s", ds.Spec.Selector.MatchLabels["app"])
						// List all pods that belong to the DaemonSet.
						listOption := func(lo *metav1.ListOptions) {
							lo.LabelSelector = selector
						}
						pods := &corev1.PodList{}
						err := cfg.Client().Resources(ds.Namespace).List(context.TODO(), pods, listOption)
						if err != nil || pods.Items == nil {
							t.Fatalf("error while getting pods for DaemonSet %+v, selector: %q, err: %q", ds, selector, err)
						}
						// Run verification inside pods.
						script := `for intf in $(chroot /host ip a | awk -F '@| ' '/veth/ {print $2}'); do echo -n "$intf "; chroot /host ethtool -k $intf | grep -E 'tx-checksumming'; echo -n "$intf "; chroot /host ethtool -k $intf | grep -E 'rx-checksumming'; done`
						for _, p := range pods.Items {
							var stdout, stderr bytes.Buffer
							command := []string{"/bin/bash", "-c", script}
							if err := cfg.Client().Resources().ExecInPod(ctx, p.Namespace, p.Name, installerName, command, &stdout, &stderr); err != nil {
								t.Log(stderr.String())
								t.Fatal(err)
							}
							t.Logf("pod %q, stdout: '%s', stderr: '%s'", p.Name, stdout.String(), stderr.String())
						}
						// TODO: automate this.
						return ctx
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
						// Run verification inside pods.
						script := `ethtool -k eth0 | grep -E 'tx-checksumming'; echo -n "$intf "; ethtool -k eth0 | grep -E 'rx-checksumming'`
						for _, p := range pods.Items {
							var stdout, stderr bytes.Buffer
							command := []string{"/bin/bash", "-c", script}
							if err := cfg.Client().Resources().ExecInPod(ctx, p.Namespace, p.Name, dep.Name, command, &stdout, &stderr); err != nil {
								t.Log(stderr.String())
								t.Fatal(err)
							}
							t.Logf("pod %q, stdout: '%s', stderr: '%s'", p.Name, stdout.String(), stderr.String())
						}
						// TODO: automate this.
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

func generateCNIConfiguration(es ethtool.EthtoolConfigs) string {
	return fmt.Sprintf(installerConfigurationTemplate, es.String())
}
