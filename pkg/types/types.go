package types

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/logging"
)

var yamlDocSeparatorRe = regexp.MustCompile(`(?m)^---\s*$`)

type TektonTypes struct {
	PipelineRuns []*tektonv1.PipelineRun
	Pipelines    []*tektonv1.Pipeline
	TaskRuns     []*tektonv1.TaskRun
	Tasks        []*tektonv1.Task
}

type KubeTypes struct {
	Secrets    []*corev1.Secret
	ConfigMaps []*corev1.ConfigMap
}

type Types struct {
	Kube   KubeTypes
	Tekton TektonTypes
}

func ReadTektonTypes(ctx context.Context, yamls []string) (Types, error) {
	logger := logging.FromContext(ctx)
	types := Types{}
	decoder := k8scheme.Codecs.UniversalDeserializer()

	for _, base64data := range yamls {
		// decode base64data
		bdata, err := base64.StdEncoding.DecodeString(base64data)
		if err != nil {
			return Types{}, err
		}
		fmt.Printf("string(bdata): %v\n", string(bdata))

		for _, doc := range yamlDocSeparatorRe.Split(string(bdata), -1) {
			if strings.TrimSpace(doc) == "" {
				continue
			}

			obj, _, err := decoder.Decode([]byte(doc), nil, nil)
			if err != nil {
				logger.Error(fmt.Sprintf("Skipping document not looking like a kubernetes resources: %s", err.Error()))
				continue
			}
			switch o := obj.(type) {
			case *tektonv1.PipelineRun:
				types.Tekton.PipelineRuns = append(types.Tekton.PipelineRuns, o)
			case *tektonv1.Pipeline:
				types.Tekton.Pipelines = append(types.Tekton.Pipelines, o)
			case *tektonv1.Task:
				types.Tekton.Tasks = append(types.Tekton.Tasks, o)
			case *corev1.Secret:
				types.Kube.Secrets = append(types.Kube.Secrets, o)
			case *corev1.ConfigMap:
				types.Kube.ConfigMaps = append(types.Kube.ConfigMaps, o)
			default:
				logger.Info("Skipping document not looking like a tekton resource we can Resolve.")
			}
		}
	}

	return types, nil
}

//nolint:gochecknoinits
func init() {
	_ = tektonv1.AddToScheme(k8scheme.Scheme)
	_ = corev1.AddToScheme(k8scheme.Scheme)
}
