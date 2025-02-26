/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Open5GSReference defines the reference to an Open5GS instance
type Open5GSReference struct {
	Name      string `json:"name,omitempty" default:"open5gs"`
	Namespace string `json:"namespace,omitempty" default:"default"`
}

// Open5GSUserSpec defines the desired state of Open5GSUser
type Open5GSUserSpec struct {
	IMSI    string           `json:"imsi,omitempty" default:"999700000000001"`
	Key     string           `json:"key,omitempty" default:"465B5CE8B199B49FAA5F0A2EE238A6BC"`
	OPC     string           `json:"opc,omitempty" default:"E8ED289DEBA952E4283B54E88E6183CA"`
	SD      string           `json:"sd,omitempty" default:"0x111111"`
	SST     string           `json:"sst,omitempty" default:"1"`
	APN     string           `json:"apn,omitempty" default:"internet"`
	Open5GS Open5GSReference `json:"open5gs,omitempty" default:"{\"name\":\"open5gs\",\"namespace\":\"default\"}"`
}

// Open5GSUserStatus defines the observed state of Open5GSUser
type Open5GSUserStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Open5GSUser is the Schema for the open5gsusers API
type Open5GSUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Open5GSUserSpec   `json:"spec,omitempty"`
	Status Open5GSUserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// Open5GSUserList contains a list of Open5GSUser
type Open5GSUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Open5GSUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Open5GSUser{}, &Open5GSUserList{})
}
