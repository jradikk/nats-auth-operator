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

// AuthMode defines the authentication mode for NATS
// +kubebuilder:validation:Enum=token;jwt;mixed
type AuthMode string

const (
	AuthModeToken AuthMode = "token"
	AuthModeJWT   AuthMode = "jwt"
	AuthModeMixed AuthMode = "mixed"
)

// ServerAuthConfigRef defines where to write the server auth configuration
type ServerAuthConfigRef struct {
	// Name of the ConfigMap or Secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the ConfigMap or Secret
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Key within the ConfigMap or Secret
	// +kubebuilder:default="auth.conf"
	Key string `json:"key,omitempty"`

	// Type of the resource (ConfigMap or Secret)
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	// +kubebuilder:default="ConfigMap"
	Type string `json:"type,omitempty"`
}

// OperatorSeedSecretRef references an existing operator seed
type OperatorSeedSecretRef struct {
	// Name of the Secret containing the operator seed
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the Secret
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Key within the Secret
	// +kubebuilder:default="operator.seed"
	Key string `json:"key,omitempty"`
}

// JWTConfig defines JWT-specific configuration
type JWTConfig struct {
	// ResolverDir is the directory path where the resolver is stored
	// +kubebuilder:default="/var/lib/nats-resolver"
	ResolverDir string `json:"resolverDir,omitempty"`

	// OperatorSeedSecret references an existing operator seed (optional)
	OperatorSeedSecret *OperatorSeedSecretRef `json:"operatorSeedSecret,omitempty"`

	// OperatorName is the name of the NATS operator
	// +kubebuilder:default="NATS Operator"
	OperatorName string `json:"operatorName,omitempty"`
}

// NatsAuthConfigSpec defines the desired state of NatsAuthConfig
type NatsAuthConfigSpec struct {
	// NatsURL is the URL for NATS clients to connect
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^nats://.*`
	NatsURL string `json:"natsURL"`

	// Mode defines the authentication mode (token, jwt, or mixed)
	// +kubebuilder:validation:Required
	// +kubebuilder:default="jwt"
	Mode AuthMode `json:"mode"`

	// ServerAuthConfig defines where to write the server auth configuration
	// +kubebuilder:validation:Required
	ServerAuthConfig ServerAuthConfigRef `json:"serverAuthConfig"`

	// JWT configuration (required if mode is jwt or mixed)
	JWT *JWTConfig `json:"jwt,omitempty"`
}

// NatsAuthConfigStatus defines the observed state of NatsAuthConfig
type NatsAuthConfigStatus struct {
	// OperatorPubKey is the public key of the NATS operator (JWT mode)
	OperatorPubKey string `json:"operatorPubKey,omitempty"`

	// ResolverReady indicates if the resolver is ready (JWT mode)
	ResolverReady bool `json:"resolverReady,omitempty"`

	// LastReconciled is the timestamp of the last reconciliation
	LastReconciled *metav1.Time `json:"lastReconciled,omitempty"`

	// Conditions represent the latest available observations of the object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed NatsAuthConfig
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
// +kubebuilder:printcolumn:name="NATS URL",type=string,JSONPath=`.spec.natsURL`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.resolverReady`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// NatsAuthConfig is the Schema for the natsauthconfigs API
type NatsAuthConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NatsAuthConfigSpec   `json:"spec,omitempty"`
	Status NatsAuthConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NatsAuthConfigList contains a list of NatsAuthConfig
type NatsAuthConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NatsAuthConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NatsAuthConfig{}, &NatsAuthConfigList{})
}
