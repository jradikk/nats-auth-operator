/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AccountLimits defines limits for a NATS account
type AccountLimits struct {
	// Conn is the maximum number of connections (-1 for unlimited)
	// +kubebuilder:default=-1
	Conn int64 `json:"conn,omitempty"`

	// Subs is the maximum number of subscriptions (-1 for unlimited)
	// +kubebuilder:default=-1
	Subs int64 `json:"subs,omitempty"`

	// Payload is the maximum message payload size in bytes (-1 for unlimited)
	// +kubebuilder:default=-1
	Payload int64 `json:"payload,omitempty"`

	// Data is the maximum data size in bytes (-1 for unlimited)
	// +kubebuilder:default=-1
	Data int64 `json:"data,omitempty"`

	// Exports is the maximum number of exports (-1 for unlimited)
	// +kubebuilder:default=-1
	Exports int64 `json:"exports,omitempty"`

	// Imports is the maximum number of imports (-1 for unlimited)
	// +kubebuilder:default=-1
	Imports int64 `json:"imports,omitempty"`

	// WildcardExports whether wildcards are allowed in exports
	// +kubebuilder:default=true
	WildcardExports bool `json:"wildcardExports,omitempty"`

	// JetStream defines JetStream-specific limits
	JetStream *JetStreamLimits `json:"jetstream,omitempty"`
}

// JetStreamLimits defines JetStream resource limits for an account
type JetStreamLimits struct {
	// MemoryStorage is the max number of bytes stored in memory across all streams (-1 for unlimited, 0 to disable)
	MemoryStorage int64 `json:"memoryStorage,omitempty"`

	// DiskStorage is the max number of bytes stored on disk across all streams (-1 for unlimited, 0 to disable)
	DiskStorage int64 `json:"diskStorage,omitempty"`

	// Streams is the maximum number of streams (-1 for unlimited)
	Streams int64 `json:"streams,omitempty"`

	// Consumer is the maximum number of consumers (-1 for unlimited)
	Consumer int64 `json:"consumer,omitempty"`

	// MaxAckPending is the maximum number of outstanding acks per stream (-1 for unlimited)
	MaxAckPending int64 `json:"maxAckPending,omitempty"`

	// MemoryMaxStreamBytes is the max bytes a memory backed stream can have (-1 for unlimited, 0 to disable)
	MemoryMaxStreamBytes int64 `json:"memoryMaxStreamBytes,omitempty"`

	// DiskMaxStreamBytes is the max bytes a disk backed stream can have (-1 for unlimited, 0 to disable)
	DiskMaxStreamBytes int64 `json:"diskMaxStreamBytes,omitempty"`

	// MaxBytesRequired requires max_bytes to be set when creating streams
	MaxBytesRequired bool `json:"maxBytesRequired,omitempty"`
}

// SecretRef references a Kubernetes Secret
type SecretRef struct {
	// Name of the Secret
	Name string `json:"name,omitempty"`

	// Namespace of the Secret
	Namespace string `json:"namespace,omitempty"`
}

// NatsAuthConfigRef references a NatsAuthConfig
type NatsAuthConfigRef struct {
	// Name of the NatsAuthConfig
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the NatsAuthConfig (defaults to same namespace)
	Namespace string `json:"namespace,omitempty"`
}

// NatsAccountSpec defines the desired state of NatsAccount
type NatsAccountSpec struct {
	// AuthConfigRef references the NatsAuthConfig
	// +kubebuilder:validation:Required
	AuthConfigRef NatsAuthConfigRef `json:"authConfigRef"`

	// Description of the account
	Description string `json:"description,omitempty"`

	// Limits defines resource limits for this account
	Limits *AccountLimits `json:"limits,omitempty"`

	// ExistingSeedSecret references an existing account seed (optional)
	ExistingSeedSecret *SecretRef `json:"existingSeedSecret,omitempty"`
}

// NatsAccountStatus defines the observed state of NatsAccount
type NatsAccountStatus struct {
	// AccountID is the public key of the account
	AccountID string `json:"accountId,omitempty"`

	// PublicKey is the public key of the account (same as AccountID)
	PublicKey string `json:"publicKey,omitempty"`

	// JWTSecretRef references the Secret containing the account JWT
	JWTSecretRef SecretRef `json:"jwtSecretRef,omitempty"`

	// Conditions represent the latest available observations of the object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed NatsAccount
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastReconciled is the timestamp of the last reconciliation
	LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Account ID",type=string,JSONPath=`.status.accountId`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NatsAccount is the Schema for the natsaccounts API
type NatsAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NatsAccountSpec   `json:"spec,omitempty"`
	Status NatsAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NatsAccountList contains a list of NatsAccount
type NatsAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NatsAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NatsAccount{}, &NatsAccountList{})
}
