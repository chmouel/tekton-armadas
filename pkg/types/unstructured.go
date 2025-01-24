package types

import (
	"encoding/base64"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// objToMap converts an object to a map representation
func objToMap(obj interface{}) map[string]interface{} {
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	unstructuredMap["Kind"] = unstructuredMap["kind"]
	delete(unstructuredMap, "kind")
	fmt.Printf("unstructuredMap: %v\n", unstructuredMap)
	if err != nil {
		log.Fatalf("Error converting object to map: %v", err)
	}
	return unstructuredMap
}

// toUnstructured converts a structured object to an unstructured object
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(objToMap(obj), u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// removeFieldForExport remove fields that should not be applied
func removeFieldForExport(obj *unstructured.Unstructured) error {
	content := obj.UnstructuredContent()

	// remove the status from pipelinerun and taskrun
	unstructured.RemoveNestedField(content, "status")

	// remove some metadata information of previous resource
	metadataFields := []string{
		"managedFields",
		"resourceVersion",
		"uid",
		"finalizers",
		"generation",
		"namespace",
		"creationTimestamp",
		"ownerReferences",
	}
	for _, field := range metadataFields {
		unstructured.RemoveNestedField(content, "metadata", field)
	}
	unstructured.RemoveNestedField(content, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")

	// check if generateName exists and remove name if it does
	if _, exist, err := unstructured.NestedString(content, "metadata", "generateName"); err != nil {
		return err
	} else if exist {
		unstructured.RemoveNestedField(content, "metadata", "name")
	}

	// remove the status from spec which are related to status
	specFields := []string{"status", "statusMessage"}
	for _, field := range specFields {
		unstructured.RemoveNestedField(content, "spec", field)
	}

	return nil
}

// SerializeObjectYaml serializes an object to a base64 encoded yaml string
func SerializeObjectYaml(p any) (string, error) {
	// use gopkgs.yaml to serialize
	uns, err := toUnstructured(p)
	if err != nil {
		return "", fmt.Errorf("failed to convert object to unstructured: %w", err)
	}
	if err := removeFieldForExport(uns); err != nil {
		return "", fmt.Errorf("failed to remove fields for export: %w", err)
	}

	marshalled, err := yaml.Marshal(uns)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object: %w", err)
	}

	return base64.StdEncoding.EncodeToString(marshalled), nil
}
