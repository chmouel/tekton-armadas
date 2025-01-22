/*
Copyright 2020 The Knative Authors

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

package orchestrator

import (
	"context"
	"encoding/base64"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/openshift-pipelines/tekton-armadas/pkg/apis/armada"
	"github.com/openshift-pipelines/tekton-armadas/pkg/clients"
	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonPipelineRunInformerv1 "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/pipelinerun"
	tektonPipelineRunReconcilerv1 "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/pipelinerun"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	clients *clients.Clients
}

// enqueue only the pipelineruns which are in `started` state
// pipelinerun will have a label `pipelinesascode.tekton.dev/state` to describe the state.
func checkStateAndEnqueue(impl *controller.Impl) func(obj interface{}) {
	return func(obj interface{}) {
		pr, err := kmeta.DeletionHandlingAccessor(obj)
		if err == nil {
			val, AnnotationExist := pr.GetAnnotations()[LabelOrchestration]
			if AnnotationExist && val == "true" {
				impl.EnqueueKey(types.NamespacedName{Namespace: pr.GetNamespace(), Name: pr.GetName()})
			}
		}
	}
}

func ctrlOpts() func(impl *controller.Impl) controller.Options {
	return func(_ *controller.Impl) controller.Options {
		return controller.Options{
			FinalizerName: armada.GroupName,
			PromoteFilterFunc: func(obj interface{}) bool {
				val, exist := obj.(*tektonv1.PipelineRun).GetAnnotations()[LabelOrchestration]
				return exist && val == "true"
			},
		}
	}
}

// NewReconciler creates a Reconciler and returns the result of NewImpl.
func NewReconciler(ctx context.Context, _ configmap.Watcher) *controller.Impl {
	pipelineRunInformer := tektonPipelineRunInformerv1.Get(ctx)

	clients, err := clients.NewClients()
	if err != nil {
		logging.FromContext(ctx).Panicf("Couldn't register clients: %w", err)
	}

	r := &Reconciler{
		clients: clients,
	}
	impl := tektonPipelineRunReconcilerv1.NewImpl(ctx, r, ctrlOpts())

	if _, err := pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(checkStateAndEnqueue(impl))); err != nil {
		logging.FromContext(ctx).Panicf("Couldn't register PipelineRun informer event handler: %w", err)
	}

	return impl
}

func serializeObjectYaml(p any) (string, error) {
	// use gopkgs.yaml to serialize
	marshalled, err := yaml.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object: %w", err)
	}
	return base64.StdEncoding.EncodeToString(marshalled), nil
}

func (r *Reconciler) HandlePendingPipelineRun(ctx context.Context, pr *tektonv1.PipelineRun) reconciler.Event {
	logger := logging.FromContext(ctx)
	data, err := serializeObjectYaml(pr)
	if err != nil {
		return err
	}
	logger.Infof("Sending PipelineRun %s to minion: %s", pr.GetName(), data)
	return nil
}

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *tektonv1.PipelineRun) reconciler.Event {
	// This logger has all the context necessary to identify which resource is being reconciled.
	logger := logging.FromContext(ctx)
	if pr.Spec.Status == tektonv1.PipelineRunSpecStatusPending && pr.Status.GetConditions() == nil {
		label, labelExist := pr.GetLabels()[pipelineapi.PipelineLabelKey]
		if !labelExist || label == "" {
			return nil
		}
		logger.Infof("Reconciling PipelineRun %s, status: %s", pr.GetName(), pr.Spec.Status)
		return r.HandlePendingPipelineRun(ctx, pr)
	}
	return nil
}
