package main

import (
	"context"

	"github.com/openshift-pipelines/tekton-armadas/pkg/clients"
	minionController "github.com/openshift-pipelines/tekton-armadas/pkg/controller/minion"
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
)

func main() {
	ctx := signals.NewContext()
	ctx = adapter.WithInjectorEnabled(ctx)
	newClients, err := clients.NewClients()
	if err != nil {
		// TODO: handle error
		panic(err)
	}

	ctx = context.WithValue(ctx, client.Key{}, newClients.Kube)
	ctx = adapter.WithNamespace(ctx, system.Namespace())
	ctx = adapter.WithConfigWatcherEnabled(ctx)
	adapter.MainWithContext(ctx, "minion-controller", minionController.NewEnvConfig, minionController.NewController(newClients))
}