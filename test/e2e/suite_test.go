/*

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
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	batchv1 "github.com/fllaca/job-operator/api/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports

	"k8s.io/client-go/tools/clientcmd"

	k8sbatchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{envtest.NewlineReporter{}})
}

func TestCreateJobTemplate(t *testing.T) {

	jobTemplate := &batchv1.JobTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: batchv1.JobTemplateSpec{
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: k8sbatchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "alpine:3.10",
									Args: []string{
										"/bin/sh",
										"-c",
										"date; echo Hello $MESSAGE",
									},
									Env: []corev1.EnvVar{
										{
											Name:  "MESSAGE",
											Value: "World",
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
	logf.Log.Logger.Info("to be created", "jobTemplate", jobTemplate)
	ctx := context.Background()

	err := k8sClient.Create(ctx, jobTemplate)
	if err != nil {
		Fail(err.Error())
	}

	jobRun := &batchv1.JobRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: batchv1.JobRunSpec{
			JobTemplateName: "test",
			Overrides: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		},
	}

	err = k8sClient.Create(ctx, jobRun)
	if err != nil {
		t.Log(err)
		Fail(err.Error())
	}

	time.Sleep(5 * time.Second)

	job := &k8sbatchv1.Job{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name: "jobrun-test",
	}, job)
	if err != nil {
		t.Log(err)
		Fail(err.Error())
	}

	//Expect(err).ToNot(HaveOccurred())

	//Expect(err).ToNot(HaveOccurred())
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")

	var err error
	var kubeconfig string

	// TODO:  <01-03-20, fllaca // replace with environment variable lookup
	kubeconfig = "/Users/fllaca/.kube/config"
	cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = batchv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	ctx := context.Background()
	// remove existing jobTemplates
	jobTemplateList := &batchv1.JobTemplateList{}
	err = k8sClient.List(ctx, jobTemplateList)
	if err != nil {
		Fail(err.Error())
	}
	for _, jobTemplate := range jobTemplateList.Items {
		err = k8sClient.Delete(ctx, &jobTemplate)
		if err != nil {
			Fail(err.Error())
		}
	}
	// remove existing JobRuns
	jobRunList := &batchv1.JobRunList{}
	err = k8sClient.List(ctx, jobRunList)
	if err != nil {
		Fail(err.Error())
	}
	for _, jobRun := range jobRunList.Items {
		err = k8sClient.Delete(ctx, &jobRun)
		if err != nil {
			Fail(err.Error())
		}
	}
	time.Sleep(10 * time.Second)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	//Expect(err).ToNot(HaveOccurred())
})
