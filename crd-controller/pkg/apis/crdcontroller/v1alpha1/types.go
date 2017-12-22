package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//Specification for CustomeDeployment
type CustomDeployment struct{
	metav1.TypeMeta					`json:",inline"`
	metav1.ObjectMeta 				`json:"metadata,omitempty"`

	Spec CustomPodSpec 		`json:"spec"`
	Status CustomDeploymentStatus  `json:"status"`
}

//Specification for DemoPod
type CustomPodSpec struct{
	PodName 	string 	`json:"pod_name"`
	Replicas 	*int32 	`json:"replicas"`
}

//Status for CustomDeployment
type CustomDeploymentStatus struct{
	AvailableReplicas	*int32 `json:"available_replicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//List of CustomDeployment
type CustomDeploymentList struct{
	metav1.TypeMeta	`json:",inline"`
	metav1.ListMeta	`json:"metadata"`

	Items []CustomDeployment `json:"items"`
}