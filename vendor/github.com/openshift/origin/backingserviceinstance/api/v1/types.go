package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

// BackingServiceInstance describe a BackingServiceInstance
type BackingServiceInstance struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	kapi.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the Namespace.
	Spec BackingServiceInstanceSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec" description:"spec defines the behavior of the BackingServiceInstance"`

	// Status describes the current status of a Namespace
	Status BackingServiceInstanceStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status" description:"status describes the current status of a Project; read-only"`
}

// BackingServiceInstanceList describe a list of BackingServiceInstance
type BackingServiceInstanceList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of routes
	Items []BackingServiceInstance `json:"items" protobuf:"bytes,2,rep,name=items" description:"list of BackingServiceInstances"`
}

/*
type BackingServiceInstanceSpec struct {
	Config                 map[string]string `json:"config, omitempty"`
	InstanceID             string            `json:"instance_id, omitempty"`
	DashboardUrl           string            `json:"dashboard_url, omitempty"`
	BackingServiceName     string            `json:"backingservice_name, omitempty"`
	BackingServiceID       string            `json:"backingservice_id, omitempty"`
	BackingServicePlanGuid string            `json:"backingservice_plan_guid, omitempty"`
	Parameters             map[string]string `json:"parameters, omitempty"`
	Binding                bool              `json:"binding, omitempty"`
	BindUuid               string            `json:"bind_uuid, omitempty"`
	BindDeploymentConfig   map[string]string `json:"bind_deploymentconfig, omitempty"`
	Credential             map[string]string `json:"credential, omitempty"`
	Tags                   []string          `json:"tags, omitempty"`
}
*/

// BackingServiceInstanceSpec describes the attributes on a BackingServiceInstance
type BackingServiceInstanceSpec struct {
	// description of an instance.
	InstanceProvisioning `json:"provisioning, omitempty" protobuf:"bytes,1,opt,name=provisioning"`
	// description of an user-provided-service
	UserProvidedService `json:"userprovidedservice, omitempty" protobuf:"bytes,2,opt,name=userprovidedservice"`
	// bindings of an instance
	Binding []InstanceBinding `json:"binding, omitempty" protobuf:"bytes,3,rep,name=binding"`
	// binding number of an instance
	Bound int64 `json:"bound, omitempty" protobuf:"varint,4,opt,name=bound"`
	// id of an instance
	InstanceID string `json:"instance_id, omitempty" protobuf:"bytes,5,opt,name=instance_id"`
	// tags of an instance
	Tags []string `json:"tags, omitempty" protobuf:"bytes,6,rep,name=tags"`
}

/*
type InstanceProvisioning struct {
	DashboardUrl           string            `json:"dashboard_url, omitempty"`
	BackingService         string            `json:"backingservice, omitempty"`
	BackingServiceName     string            `json:"backingservice_name, omitempty"`
	BackingServiceID       string            `json:"backingservice_id, omitempty"`
	BackingServicePlanGuid string            `json:"backingservice_plan_guid, omitempty"`
	BackingServicePlanName string            `json:"backingservice_plan_name, omitempty"`
	Parameters             map[string]string `json:"parameters, omitempty"`
}
*/

// UserProvidedService describe an user-provided-service
type UserProvidedService struct {
	Credentials map[string]string `json:"credentials, omitempty" protobuf:"bytes,1,rep,name=credentials"`
}

// InstanceProvisioning describe an InstanceProvisioning detail
type InstanceProvisioning struct {
	// dashboard url of an instance
	DashboardUrl string `json:"dashboard_url, omitempty" protobuf:"bytes,1,opt,name=dashboard_url"`
	// bs name of an instance
	BackingServiceName string `json:"backingservice_name, omitempty" protobuf:"bytes,2,opt,name=backingservice_name"`
	// bs id of an instance
	BackingServiceSpecID string `json:"backingservice_spec_id, omitempty" protobuf:"bytes,3,opt,name=backingservice_spec_id"`
	// bs plan id of an instance
	BackingServicePlanGuid string `json:"backingservice_plan_guid, omitempty" protobuf:"bytes,4,opt,name=backingservice_plan_guid"`
	// bs plan name of an instance
	BackingServicePlanName string `json:"backingservice_plan_name, omitempty" protobuf:"bytes,5,opt,name=backingservice_plan_name"`
	// parameters of an instance
	Parameters map[string]string `json:"parameters, omitempty" protobuf:"bytes,6,rep,name=parameters"`
}

// InstanceBinding describe an instance binding.
type InstanceBinding struct {
	// bound time of an instance binding
	BoundTime *unversioned.Time `json:"bound_time,omitempty" protobuf:"bytes,1,opt,name=bound_time"`
	// bind uid of an instance binding
	BindUuid string `json:"bind_uuid, omitempty" protobuf:"bytes,2,opt,name=bind_uuid"`
	// deploymentconfig of an binding.
	BindDeploymentConfig string `json:"bind_deploymentconfig, omitempty" protobuf:"bytes,3,opt,name=bind_deploymentconfig"`
	// credentials of an instance binding
	Credentials map[string]string `json:"credentials, omitempty" protobuf:"bytes,4,rep,name=credentials"`
}

// BackingServiceInstanceStatus describe the status of a BackingServiceInstance
type BackingServiceInstanceStatus struct {
	// phase is the current lifecycle phase of the instance
	Phase BackingServiceInstancePhase `json:"phase, omitempty" protobuf:"bytes,1,opt,name=phase"`
	// action is the action of the instance
	Action BackingServiceInstanceAction `json:"action, omitempty" protobuf:"bytes,2,opt,name=action"`
	//last operation  of a instance provisioning
	LastOperation *LastOperation `json:"last_operation, omitempty" protobuf:"bytes,3,opt,name=last_operation"`
}

// LastOperation describe last operation of an instance provisioning
type LastOperation struct {
	// state of last operation
	State string `json:"state" protobuf:"bytes,1,opt,name=state"`
	// description of last operation
	Description string `json:"description" protobuf:"bytes,2,opt,name=description"`
	// async_poll_interval_seconds of a last operation
	AsyncPollIntervalSeconds int64 `json:"async_poll_interval_seconds, omitempty" protobuf:"varint,3,opt,name=async_poll_interval_seconds"`
}

type BackingServiceInstancePhase string
type BackingServiceInstanceAction string

const (
	BackingServiceInstancePhaseProvisioning BackingServiceInstancePhase = "Provisioning"
	BackingServiceInstancePhaseUnbound      BackingServiceInstancePhase = "Unbound"
	BackingServiceInstancePhaseBound        BackingServiceInstancePhase = "Bound"
	BackingServiceInstancePhaseDeleted      BackingServiceInstancePhase = "Deleted"

	BackingServiceInstanceActionToBind   BackingServiceInstanceAction = "_ToBind"
	BackingServiceInstanceActionToUnbind BackingServiceInstanceAction = "_ToUnbind"
	BackingServiceInstanceActionToDelete BackingServiceInstanceAction = "_ToDelete"

	BindDeploymentConfigBinding   string = "binding"
	BindDeploymentConfigUnbinding string = "unbinding"
	BindDeploymentConfigBound     string = "bound"

	UPS string = "USER-PROVIDED-SERVICE"
)

//=====================================================
//
//=====================================================

const BindKind_DeploymentConfig = "DeploymentConfig"

//type BindingRequest struct {
//	unversioned.TypeMeta
//	kapi.ObjectMeta
//
//	// the dc
//	DeploymentConfigName string `json:"deployment_name, omitempty"`
//}

// BindingRequestOptions describe a BindingRequestOptions.
type BindingRequestOptions struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	kapi.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// bind kind is bindking of an instance binding
	BindKind string `json:"bindKind, omitempty" protobuf:"bytes,2,opt,name=bindKind"`
	// bindResourceVersion is bindResourceVersion of an instance binding.
	BindResourceVersion string `json:"bindResourceVersion, omitempty" protobuf:"bytes,3,opt,name=bindResourceVersion"`
	// resourceName of an instance binding
	ResourceName string `json:"resourceName, omitempty" protobuf:"bytes,4,opt,name=resourceName"`
}
