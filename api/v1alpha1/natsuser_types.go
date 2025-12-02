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

// UserAuthType defines the authentication type for a user
// +kubebuilder:validation:Enum=token;jwt;inherit
type UserAuthType string

const (
	UserAuthTypeToken   UserAuthType = "token"
	UserAuthTypeJWT     UserAuthType = "jwt"
	UserAuthTypeInherit UserAuthType = "inherit"
)

// NatsAccountRef references a NatsAccount
type NatsAccountRef struct {
	// Name of the NatsAccount
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the NatsAccount (defaults to same namespace)
	Namespace string `json:"namespace,omitempty"`
}

// PasswordSource defines how to obtain the password for token auth
type PasswordSource struct {
	// Generate indicates whether to generate a random password
	Generate bool `json:"generate,omitempty"`

	// SecretRef references an existing Secret containing the password
	SecretRef *SecretRef `json:"secretRef,omitempty"`
}

// Permissions defines publish/subscribe permissions
type Permissions struct {
	// PublishAllow is a list of subjects the user can publish to
	PublishAllow []string `json:"publishAllow,omitempty"`

	// PublishDeny is a list of subjects the user cannot publish to
	PublishDeny []string `json:"publishDeny,omitempty"`

	// SubscribeAllow is a list of subjects the user can subscribe to
	SubscribeAllow []string `json:"subscribeAllow,omitempty"`

	// SubscribeDeny is a list of subjects the user cannot subscribe to
	SubscribeDeny []string `json:"subscribeDeny,omitempty"`
}

// NatsUserSpec defines the desired state of NatsUser
type NatsUserSpec struct {
	// AuthConfigRef references the NatsAuthConfig
	// +kubebuilder:validation:Required
	AuthConfigRef NatsAuthConfigRef `json:"authConfigRef"`

	// AuthType defines the authentication type (token, jwt, or inherit from NatsAuthConfig)
	// +kubebuilder:default="inherit"
	AuthType UserAuthType `json:"authType,omitempty"`

	// AccountRef references the NatsAccount (required for JWT mode)
	AccountRef *NatsAccountRef `json:"accountRef,omitempty"`

	// Username for token-based auth
	Username string `json:"username,omitempty"`

	// PasswordFrom defines how to obtain the password (for token auth)
	PasswordFrom *PasswordSource `json:"passwordFrom,omitempty"`

	// Permissions defines publish/subscribe permissions
	Permissions *Permissions `json:"permissions,omitempty"`

	// ExistingSeedSecret references an existing user seed (optional, JWT mode)
	ExistingSeedSecret *SecretRef `json:"existingSeedSecret,omitempty"`
}

// UserState represents the state of the user
// +kubebuilder:validation:Enum=Ready;Error;Pending
type UserState string

const (
	UserStateReady   UserState = "Ready"
	UserStateError   UserState = "Error"
	UserStatePending UserState = "Pending"
)

// NatsUserStatus defines the observed state of NatsUser
type NatsUserStatus struct {
	// State represents the current state of the user
	// +kubebuilder:default="Pending"
	State UserState `json:"state,omitempty"`

	// Reason provides more detail about the current state
	Reason string `json:"reason,omitempty"`

	// SecretRef references the Secret containing user credentials
	SecretRef SecretRef `json:"secretRef,omitempty"`

	// PublicKey is the public key of the user (JWT mode)
	PublicKey string `json:"publicKey,omitempty"`

	// Conditions represent the latest available observations of the object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed NatsUser
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastReconciled is the timestamp of the last reconciliation
	LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Auth Type",type=string,JSONPath=`.spec.authType`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Account",type=string,JSONPath=`.spec.accountRef.name`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NatsUser is the Schema for the natsusers API
type NatsUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NatsUserSpec   `json:"spec,omitempty"`
	Status NatsUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NatsUserList contains a list of NatsUser
type NatsUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NatsUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NatsUser{}, &NatsUserList{})
}
