/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Open5GSSpec defines the desired state of Open5GS
type Open5GSSpec struct {
	AMF            Open5GSFunction      `json:"amf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":true,\"serviceMonitor\":false}"`
	AUSF           Open5GSFunction      `json:"ausf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	BSF            Open5GSFunction      `json:"bsf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	MongoDB        Open5GSFunction      `json:"mongoDB,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	NRF            Open5GSFunction      `json:"nrf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	NSSF           Open5GSFunction      `json:"nssf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	PCF            Open5GSFunction      `json:"pcf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":true,\"serviceMonitor\":false}"`
	SCP            Open5GSFunction      `json:"scp,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	SMF            Open5GSFunction      `json:"smf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":true,\"serviceMonitor\":false}"`
	UDM            Open5GSFunction      `json:"udm,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	UDR            Open5GSFunction      `json:"udr,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	UPF            Open5GSFunction      `json:"upf,omitempty" default:"{\"enabled\":true,\"serviceAccount\":false,\"metrics\":true,\"serviceMonitor\":false}"`
	WebUI          Open5GSFunction      `json:"webui,omitempty" default:"{\"enabled\":false,\"serviceAccount\":false,\"metrics\":false,\"serviceMonitor\":false}"`
	WebUIImage     string               `json:"webuiImage,omitempty" default:"docker.io/gradiant/open5gs-webui:2.7.3"`
	Open5GSImage   string               `json:"open5gsImage,omitempty" default:"docker.io/gradiant/open5gs:2.7.3"`
	MongoDBVersion string               `json:"mongoDBVersion,omitempty" default:"bitnami/mongodb:8.0.6-debian-12-r0"`
	Configuration  Open5GSConfiguration `json:"configuration,omitempty" default:"{\"mcc\":\"999\",\"mnc\":\"70\",\"region\":\"2\",\"set\":\"1\",\"tac\":\"0001\",\"slices\":[]}"`
}

type Open5GSConfiguration struct {
	MCC    string         `json:"mcc,omitempty" default:"999"`
	MNC    string         `json:"mnc,omitempty" default:"70"`
	Region string         `json:"region,omitempty" default:"2"`
	Set    string         `json:"set,omitempty" default:"1"`
	TAC    string         `json:"tac,omitempty" default:"0001"`
	Slices []Open5GSSlice `json:"slices,omitempty"`
}

type Open5GSSlice struct {
	SST string `json:"sst,omitempty"`
	SD  string `json:"sd,omitempty"`
}

type Open5GSFunction struct {
	Enabled        *bool            `json:"enabled,omitempty" default:"true"`
	ServiceAccount *bool            `json:"serviceAccount,omitempty" default:"false"`
	Metrics        *bool            `json:"metrics,omitempty" default:"true"`
	ServiceMonitor *bool            `json:"serviceMonitor,omitempty" default:"false"`
	Service        []Open5GSService `json:"service,omitempty" default:"{\"name\":\"\",\"port\":0,\"serviceType\":\"\"}"`
}

type Open5GSService struct {
	Name        string `json:"name,omitempty"`
	ServiceType string `json:"serviceType,omitempty"`
}

// Open5GSStatus defines the observed state of Open5GS
type Open5GSStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Ready bool `json:"ready"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Open5GS is the Schema for the open5gs API
type Open5GS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Open5GSSpec   `json:"spec,omitempty"`
	Status Open5GSStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// Open5GSList contains a list of Open5GS
type Open5GSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Open5GS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Open5GS{}, &Open5GSList{})
}
