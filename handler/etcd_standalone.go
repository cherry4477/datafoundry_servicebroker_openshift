package handler

import (
	"fmt"
	"errors"
	//marathon "github.com/gambol99/go-marathon"
	//kapi "golang.org/x/build/kubernetes/api"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	//"net/http"
	"github.com/pivotal-cf/brokerapi"
	"time"
	//"strings"
	"bytes"
	"encoding/json"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	"sync"
	
	"github.com/pivotal-golang/lager"
	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
)

//==============================================================
// 
//==============================================================

func (oc *OpenshiftClient) t() {
	new(Etcd_sampleHandler).DoProvision(
		"test1", 
		brokerapi.ProvisionDetails{
			OrganizationGUID: "ttt",
		},
		true)
}

//==============================================================
// 
//==============================================================

const EtcdServcieBrokerName = "etcd_sample_Sample"

func init() {
	register(EtcdServcieBrokerName, &Etcd_sampleHandler{})
	
	
	logger = lager.NewLogger(EtcdServcieBrokerName)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

const EtcdBindRole = "binduser"

const ServiceBrokerNamespace = "default"

type Etcd_sampleHandler struct{}

func (handler *Etcd_sampleHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, ServiceInfo, error) {
	//初始化到openshift的链接
	
	serviceSpec := brokerapi.ProvisionedServiceSpec{IsAsync: asyncAllowed}
	serviceInfo := ServiceInfo{}
	
	if asyncAllowed == false {
		return serviceSpec, serviceInfo, errors.New("Sync mode is not supported")
	}
	
	instanceIdInTempalte   := instanceID // todo: ok?
	serviceBrokerNamespace := ServiceBrokerNamespace // 
	
	// boot etcd
	
	output, err := createEtcdResources_Boot(instanceIdInTempalte, serviceBrokerNamespace)

	if err != nil {
		println("etcd prosision error: ", err.Error())
		
		destroyEtcdResources_Boot(output, serviceBrokerNamespace)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.Admin_password = getguid()
	serviceInfo.User = getguid()
	serviceInfo.Password = getguid()
	
	startEtcdOrchestrationJob(&etcdOrchestrationJob{
		//instanceId:     instanceIdInTempalte,
		isProvisioning: true,
		serviceInfo:    &serviceInfo,
		bootResources:  output,
		haResources:    nil,
	})
	
	serviceSpec.DashboardURL = "http://not-available-now"
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Etcd_sampleHandler) DoLastOperation(myServiceInfo *ServiceInfo) (brokerapi.LastOperation, error) {
	// try to get state from running job
	job := getEtcdOrchestrationJob (myServiceInfo.Url)
	if job != nil {
		if job.isProvisioning {
			return brokerapi.LastOperation{
				State:       brokerapi.InProgress,
				Description: "In progress.",
			}, nil
		} else {
			return brokerapi.LastOperation{
				State:       brokerapi.Failed,
				Description: "Failed!",
			}, nil
		}
	}
	
	// the job may be finished or interrupted or running in another instance.
	
	// check boot route, if it doesn't exist, return failed
	// no need to check err here, for error mya be caused by boot-pod has already been removed.
	boot_res, _ := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if boot_res.route.Name == "" {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: "Failed!",
		}, nil
	}
	
	// only check the statuses of 3 ReplicationControllers. The etcd pods may be not running well.
	
	ok := func(rc *kapi.ReplicationController) int {
		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || *rc.Spec.Replicas <= rc.Status.Replicas {
			return 0
		}
		return 1
	}
	
	ha_res, _ := getEtcdResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Admin_password)
	
	num_ok_rcs := 0
	num_ok_rcs += ok (&ha_res.etcdrc1)
	num_ok_rcs += ok (&ha_res.etcdrc2)
	num_ok_rcs += ok (&ha_res.etcdrc3)
	
	if num_ok_rcs < 3 {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "In progress.",
		}, nil
	} else {
		return brokerapi.LastOperation{
			State:       brokerapi.Succeeded,
			Description: "Succeeded!",
		}, nil
	}
}

func (handler *Etcd_sampleHandler) DoDeprovision(myServiceInfo *ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	// todo: handle errors
	
	ha_res, _ := getEtcdResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Admin_password)
	destroyEtcdResources_HA (ha_res, myServiceInfo.Database)
	
	boot_res, _ := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	destroyEtcdResources_Boot (boot_res, myServiceInfo.Database)
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Etcd_sampleHandler) DoBind(myServiceInfo *ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, Credentials, error) {
	// output.route.Spec.Host
	
	
	
	return brokerapi.Binding{}, Credentials{}, errors.New("not implemented")
}

func (handler *Etcd_sampleHandler) DoUnbind(myServiceInfo *ServiceInfo, mycredentials *Credentials) error {
	//

	return errors.New("not implemented")

}

//==============================================================
// 
//==============================================================

var etcdOrchestrationJobs = map[string]*etcdOrchestrationJob{}
var etcdOrchestrationJobsMutex sync.Mutex

func getEtcdOrchestrationJob (instanceId string) *etcdOrchestrationJob {
	etcdOrchestrationJobsMutex.Lock()
	defer etcdOrchestrationJobsMutex.Unlock()
	
	return etcdOrchestrationJobs[instanceId]
}

func startEtcdOrchestrationJob (job *etcdOrchestrationJob) {
	etcdOrchestrationJobsMutex.Lock()
	defer etcdOrchestrationJobsMutex.Unlock()
	
	if etcdOrchestrationJobs[job.serviceInfo.Url] == nil {
		etcdOrchestrationJobs[job.serviceInfo.Url] = job
		go func() {
			//defer removeEtcdOrchestrationJob (job.serviceInfo.Url) // slowers
			job.run()
			removeEtcdOrchestrationJob (job.serviceInfo.Url) // faster
		}()
	}
}

func removeEtcdOrchestrationJob (instanceId string) {
	etcdOrchestrationJobsMutex.Lock()
	defer etcdOrchestrationJobsMutex.Unlock()
	
	delete(etcdOrchestrationJobs, instanceId)
}

type etcdOrchestrationJob struct {
	//instanceId string // use serviceInfo.
	
	isProvisioning bool // false for deprovisionings
	
	serviceInfo   *ServiceInfo
	bootResources *etcdResources_Boot
	haResources   *etcdResources_HA
}

type watchPodStatus struct {
	// The type of watch update contained in the message
	Type string `json:"type"`
	// Pod details
	Object kapi.Pod `json:"object"`
}


//provisioning steps:
//1. create first etcd pod/service/route (aka. a single node etcd cluster)
//   watch pod status to running
//2. modify etcd acl (add root, remove gust, add bind role)
//3. create HA resources
//4. watch HA pods, when all are running, delete boot pod
//   (or add 2nd and 3rd etcd pod, so no need to delete boot pod)

func (job *etcdOrchestrationJob) run() {
	serviceInfo := job.serviceInfo
	pod := job.bootResources.etcdpod
	uri := "/namespaces/" + serviceInfo.Database + "/pods/" + pod.Name
	statuses, _, err := theOC.KWatch (uri)
	if err != nil {
		logger.Error("start watching boot pod", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	var wps watchPodStatus
	for {
		// todo: add cancel mechanism
		
		status := <- statuses
		
		if status.Err != nil {
			logger.Error("watch boot pod", err)
			destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
			return
		}
		
		if err := json.Unmarshal(status.Info, &wps); err != nil {
			logger.Error("parse boot pod status", err)
			destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
			return
		}
		
		if wps.Object.Status.Phase != kapi.PodPending {
			if wps.Object.Status.Phase != kapi.PodRunning {
				logger.Debug("pod phase is neither pending nor running")
				destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
				return
			}
			
			break
		}
	}
	
	// etcd acl
	
	etcd_client, err := newUnauthrizedEtcdClient ([]string{})
	if err != nil {
		logger.Error("create etcd unauthrized client", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	etcd_userapi := etcd.NewAuthUserAPI(etcd_client)
	err = etcd_userapi.AddUser(context.Background(), "root", serviceInfo.Admin_password)
	if err != nil {
		logger.Error("create etcd root user", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	etcd_authapi := etcd.NewAuthAPI(etcd_client)
	err = etcd_authapi.Enable(context.Background())
	if err != nil {
		logger.Error("enable etcd auth", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	etcd_roleapi := etcd.NewAuthRoleAPI(etcd_client)
	_, err = etcd_roleapi.RevokeRoleKV(context.Background(), "guest", []string{"/*"}, etcd.ReadWritePermission)
	if err != nil {
		logger.Error("revoke guest role permission", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	err = etcd_roleapi.AddRole(context.Background(), EtcdBindRole)
	if err != nil {
		logger.Error("add etcd binduser role", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	_, err = etcd_roleapi.GrantRoleKV(context.Background(), EtcdBindRole, []string{"/*"}, etcd.ReadWritePermission)
	if err != nil {
		logger.Error("grant ectd binduser role", err)
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database)
		return
	}
	
	// create HA resources
	
	ha_res, err := createEtcdResources_HA (serviceInfo.Url, serviceInfo.Database, serviceInfo.Admin_password)
	_ = ha_res
	//ha_res, err = getEtcdResources_HA (serviceInfo.Url, serviceInfo.Database, serviceInfo.Admin_password)
	
	// ...
}

func newUnauthrizedEtcdClient (etcdEndPoints []string) (etcd.Client, error) {
	cfg := etcd.Config{
		Endpoints: etcdEndPoints,
		Transport: etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	return etcd.New(cfg)
}

func newAuthrizedEtcdClient (etcdEndPoints []string, etcdUser, etcdPassword string) (etcd.Client, error) {
	cfg := etcd.Config{
		Endpoints: etcdEndPoints,
		Transport: etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
		Username:                etcdUser,
		Password:                etcdPassword,
	}
	return etcd.New(cfg)
}

//===============================================================
// 
//===============================================================

var EtcdTemplateData_Boot []byte = nil

func loadEtcdResources_Boot(instanceID string, res *etcdResources_Boot) error {
	if EtcdTemplateData_Boot == nil {
		f, err := os.Open("etcd-outer-boot.yaml")
		if err != nil {
			return err
		}
		EtcdTemplateData_Boot, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
	}
	
	yamlTemplates := bytes.Replace(EtcdTemplateData_Boot, []byte("instanceid"), []byte(instanceID), -1)
	
	
	println("========= Boot yamlTemplates ===========")
	println(string(yamlTemplates))
	println()

	
	decoder := NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.service).
		Decode(&res.route).
		Decode(&res.etcdpod).
		Decode(&res.etcdsrv)
	
	return decoder.err
}

var EtcdTemplateData_HA []byte = nil

func loadEtcdResources_HA(instanceID, rootPassword string, res *etcdResources_HA) error {
	if EtcdTemplateData_HA == nil {
		f, err := os.Open("etcd-outer-ha.yaml")
		if err != nil {
			return err
		}
		EtcdTemplateData_HA, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
	}
	
	yamlTemplates := bytes.Replace(EtcdTemplateData_HA, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("test1234"), []byte(rootPassword), -1)
	
	
	println("========= HA yamlTemplates ===========")
	println(string(yamlTemplates))
	println()

	
	decoder := NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.etcdrc1).
		Decode(&res.etcdsrv1).
		Decode(&res.etcdrc2).
		Decode(&res.etcdsrv2).
		Decode(&res.etcdrc3).
		Decode(&res.etcdsrv3)
	
	return decoder.err
}

type etcdResources_Boot struct {
	service kapi.Service
	route   routeapi.Route
	etcdpod kapi.Pod
	etcdsrv kapi.Service
}

type etcdResources_HA struct {
	etcdrc1  kapi.ReplicationController
	etcdsrv1 kapi.Service
	etcdrc2  kapi.ReplicationController
	etcdsrv2 kapi.Service
	etcdrc3  kapi.ReplicationController
	etcdsrv3 kapi.Service
}
	
func createEtcdResources_Boot (instanceId, serviceBrokerNamespace string) (*etcdResources_Boot, error) {
	var input etcdResources_Boot
	err := loadEtcdResources_Boot(instanceId, &input)
	if err != nil {
		return nil, err
	}
	
	var output etcdResources_Boot
	
	osr := NewOpenshiftREST(theOC)
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/services", &input.service, &output.service).
		OPost(prefix + "/routes", &input.route, &output.route).
		KPost(prefix + "/pods", &input.etcdpod, &output.etcdpod).
		KPost(prefix + "/services", &input.etcdsrv, &output.etcdsrv)
	
	return &output, osr.err
}
	
func getEtcdResources_Boot (instanceId, serviceBrokerNamespace string) (*etcdResources_Boot, error) {
	var output etcdResources_Boot
	
	var input etcdResources_Boot
	err := loadEtcdResources_Boot(instanceId, &input)
	if err != nil {
		return &output, err
	}
	
	osr := NewOpenshiftREST(theOC)
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.service.Name, &output.service).
		OGet(prefix + "/routes/" + input.route.Name, &output.route).
		KGet(prefix + "/pods/" + input.etcdpod.Name, &output.etcdpod).
		KGet(prefix + "/services/" + input.etcdsrv.Name, &output.etcdsrv)
	
	return &output, osr.err
}

func destroyEtcdResources_Boot (bootRes *etcdResources_Boot, serviceBrokerNamespace string) {
	// todo: handle errors in processing
	
	kdel := func(typeName, resName string) {
		if resName != "" {
			uri := fmt.Sprintf("/namespaces/%s/%s/%s", serviceBrokerNamespace, typeName, resName)
			NewOpenshiftREST(theOC).KDelete(uri, nil)
		}
	}
	odel := func(typeName, resName string) {
		if resName != "" {
			uri := fmt.Sprintf("/namespaces/%s/%s/%s", serviceBrokerNamespace, typeName, resName)
			NewOpenshiftREST(theOC).ODelete(uri, nil)
		}
	}
	
	kdel ("services", bootRes.etcdsrv.Name)
	kdel ("pods", bootRes.etcdpod.Name)
	odel ("routes", bootRes.route.Name)
	kdel ("services", bootRes.service.Name)
}
	
func createEtcdResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) (*etcdResources_HA, error) {
	var output etcdResources_HA
	
	var input etcdResources_HA
	err := loadEtcdResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := NewOpenshiftREST(theOC)
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/replicationcontrollers", &input.etcdrc1, &output.etcdrc1).
		KPost(prefix + "/services", &input.etcdsrv1, &output.etcdsrv1).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc2, &output.etcdrc2).
		KPost(prefix + "/services", &input.etcdsrv2, &output.etcdsrv2).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc3, &output.etcdrc3).
		KPost(prefix + "/services", &input.etcdsrv3, &output.etcdsrv3)
	
	return &output, osr.err
}
	
func getEtcdResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) (*etcdResources_HA, error) {
	var output etcdResources_HA
	
	var input etcdResources_HA
	err := loadEtcdResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := NewOpenshiftREST(theOC)
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/replicationcontrollers" + input.etcdrc1.Name, &output.etcdrc1).
		KGet(prefix + "/services" + input.etcdsrv1.Name, &output.etcdsrv1.Name).
		KGet(prefix + "/replicationcontrollers" + input.etcdrc2.Name, &output.etcdrc2).
		KGet(prefix + "/services" + input.etcdsrv2.Name, &output.etcdsrv2).
		KGet(prefix + "/replicationcontrollers" + input.etcdrc3.Name, &output.etcdrc3).
		KGet(prefix + "/services" + input.etcdsrv3.Name, &output.etcdsrv3)
	
	return &output, osr.err
}

func destroyEtcdResources_HA (haRes *etcdResources_HA, serviceBrokerNamespace string) {
	// todo: handle errors in processing
	
	kdel := func(typeName, resName string) {
		if resName != "" {
			uri := fmt.Sprintf("/namespaces/%s/%s/%s", serviceBrokerNamespace, typeName, resName)
			NewOpenshiftREST(theOC).KDelete(uri, nil)
		}
	}
	
	kdel ("services", haRes.etcdsrv1.Name)
	kdel ("replicationcontrollers", haRes.etcdrc1.Name)
	
	kdel ("services", haRes.etcdsrv2.Name)
	kdel ("replicationcontrollers", haRes.etcdrc2.Name)
	
	kdel ("services", haRes.etcdsrv3.Name)
	kdel ("replicationcontrollers", haRes.etcdrc3.Name)
}

//===============================================================
// 
//===============================================================


