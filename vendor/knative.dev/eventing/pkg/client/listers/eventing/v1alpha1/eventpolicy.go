/*
Copyright 2021 The Knative Authors

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "knative.dev/eventing/pkg/apis/eventing/v1alpha1"
)

// EventPolicyLister helps list EventPolicies.
// All objects returned here must be treated as read-only.
type EventPolicyLister interface {
	// List lists all EventPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.EventPolicy, err error)
	// EventPolicies returns an object that can list and get EventPolicies.
	EventPolicies(namespace string) EventPolicyNamespaceLister
	EventPolicyListerExpansion
}

// eventPolicyLister implements the EventPolicyLister interface.
type eventPolicyLister struct {
	listers.ResourceIndexer[*v1alpha1.EventPolicy]
}

// NewEventPolicyLister returns a new EventPolicyLister.
func NewEventPolicyLister(indexer cache.Indexer) EventPolicyLister {
	return &eventPolicyLister{listers.New[*v1alpha1.EventPolicy](indexer, v1alpha1.Resource("eventpolicy"))}
}

// EventPolicies returns an object that can list and get EventPolicies.
func (s *eventPolicyLister) EventPolicies(namespace string) EventPolicyNamespaceLister {
	return eventPolicyNamespaceLister{listers.NewNamespaced[*v1alpha1.EventPolicy](s.ResourceIndexer, namespace)}
}

// EventPolicyNamespaceLister helps list and get EventPolicies.
// All objects returned here must be treated as read-only.
type EventPolicyNamespaceLister interface {
	// List lists all EventPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.EventPolicy, err error)
	// Get retrieves the EventPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.EventPolicy, error)
	EventPolicyNamespaceListerExpansion
}

// eventPolicyNamespaceLister implements the EventPolicyNamespaceLister
// interface.
type eventPolicyNamespaceLister struct {
	listers.ResourceIndexer[*v1alpha1.EventPolicy]
}
