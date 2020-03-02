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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "github.com/fllaca/job-operator/api/v1"

	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ref "k8s.io/client-go/tools/reference"
)

// JobRunReconciler reconciles a JobRun object
type JobRunReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	APIReader client.Reader
}

// +kubebuilder:rbac:groups=batch.joboperator.io,resources=jobruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.joboperator.io,resources=jobruns/status,verbs=get;update;patch

func (r *JobRunReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("jobrun", req.NamespacedName)

	// your logic here
	var jobRun batchv1.JobRun
	if err := r.Get(ctx, req.NamespacedName, &jobRun); err != nil {
		log.Error(err, "unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}
	}

	jobRun.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		jobRun.Status.Active = append(jobRun.Status.Active, *jobRef)
	}

	jobRun.Status.Completed = nil
	for _, successfulJob := range successfulJobs {
		jobRef, err := ref.GetReference(r.Scheme, successfulJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", successfulJob)
			continue
		}
		jobRun.Status.Completed = append(jobRun.Status.Completed, *jobRef)
	}

	// update status
	if err := r.Status().Update(ctx, &jobRun); err != nil {
		log.Error(err, "unable to update JobRun status")
		return ctrl.Result{}, err
	}

	// Job is already completed or being executed
	if len(jobRun.Status.Completed) > 0 || len(jobRun.Status.Active) > 0 {
		// TODO: uncomment this line:
		log.Info("jobrun is already completed", "name", jobRun.ObjectMeta.Name, "namespace", jobRun.ObjectMeta.Namespace)
		return ctrl.Result{}, nil
	}

	job, err := r.buildJobFromTemplate(ctx, log, &jobRun)
	if err != nil {
		log.Error(err, "unable to build job from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	// ...and create it on the cluster
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for JobRun", "job", job)

	return ctrl.Result{}, nil
}

func (r *JobRunReconciler) buildJobFromTemplate(ctx context.Context, log logr.Logger, jobRun *batchv1.JobRun) (*kbatch.Job, error) {
	// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
	name := fmt.Sprintf("jobrun-%s", jobRun.Name)

	var jobTemplateObject batchv1.JobTemplate

	if err := r.APIReader.Get(ctx, types.NamespacedName{
		Namespace: jobRun.ObjectMeta.Namespace,
		Name:      jobRun.Spec.JobTemplateName,
	}, &jobTemplateObject); err != nil {

	}

	job := &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   jobRun.Namespace,
		},
		Spec: *jobTemplateObject.Spec.JobTemplate.Spec.DeepCopy(),
	}
	for k, v := range jobTemplateObject.Spec.JobTemplate.Annotations {
		job.Annotations[k] = v
	}
	for k, v := range jobTemplateObject.Spec.JobTemplate.Labels {
		job.Labels[k] = v
	}
	if err := ctrl.SetControllerReference(jobRun, job, r.Scheme); err != nil {
		return nil, err
	}

	job = applyOverrides(job, jobRun)

	return job, nil
}

func applyOverrides(job *kbatch.Job, jobRun *batchv1.JobRun) *kbatch.Job {
	// apply overrides
	for _, containerOverride := range jobRun.Spec.Overrides.Containers {
		for i, container := range job.Spec.Template.Spec.Containers {
			if container.Name == containerOverride.Name {
				// TODO: merge environment variables
				job.Spec.Template.Spec.Containers[i].Env = append(container.Env, containerOverride.Env...)
			}
		}
	}

	return job
}

func isJobFinished(job *kbatch.Job) (bool, kbatch.JobConditionType) {
	for _, c := range job.Status.Conditions {
		if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
			return true, c.Type
		}
	}

	return false, ""
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = batchv1.GroupVersion.String()
)

func (r *JobRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&kbatch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		// grab the job object, extract the owner...
		job := rawObj.(*kbatch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a JobRun...
		if owner.APIVersion != apiGVStr || owner.Kind != "JobRun" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.JobRun{}).
		Complete(r)
}
