package etcd

import (
	"fmt"
	//"errors"
	//marathon "github.com/gambol99/go-marathon"
	//kapi "golang.org/x/build/kubernetes/api"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	//"net/http"
	"net"
	"github.com/pivotal-cf/brokerapi"
	"time"
	//"strconv"
	"strings"
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
	
	oshandler "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
)

//==============================================================
// 
//==============================================================

const EtcdServcieBrokerName_Standalone = "ETCD_standalone"

func init() {
	oshandler.Register(EtcdServcieBrokerName_Standalone, &Etcd_sampleHandler{})
	
	logger = lager.NewLogger(EtcdServcieBrokerName_Standalone)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

const EtcdBindRole = "binduser"

//const ServiceBrokerNamespace = "default" // use oshandler.OC().Namespace instead

type Etcd_sampleHandler struct{}

func (handler *Etcd_sampleHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
	//初始化到openshift的链接
	
	serviceSpec := brokerapi.ProvisionedServiceSpec{IsAsync: asyncAllowed}
	serviceInfo := oshandler.ServiceInfo{}
	
	//if asyncAllowed == false {
	//	return serviceSpec, serviceInfo, errors.New("Sync mode is not supported")
	//}
	serviceSpec.IsAsync = true
	
	//instanceIdInTempalte   := instanceID // todo: ok?
	instanceIdInTempalte   := strings.ToLower(oshandler.NewThirteenLengthID())
	//serviceBrokerNamespace := ServiceBrokerNamespace
	serviceBrokerNamespace := oshandler.OC().Namespace()
	
	println()
	println("instanceIdInTempalte = ", instanceIdInTempalte)
	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
	println()
	
	// boot etcd
	
	output, err := createEtcdResources_Boot(instanceIdInTempalte, serviceBrokerNamespace)

	if err != nil {
		destroyEtcdResources_Boot(output, serviceBrokerNamespace, true)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.Password = oshandler.GenGUID()
	
	startEtcdOrchestrationJob(&etcdOrchestrationJob{
		cancelled:  false,
		cancelChan: make(chan struct{}),
		
		isProvisioning: true,
		serviceInfo:    &serviceInfo,
		bootResources:  output,
		haResources:    nil,
	})
	
	serviceSpec.DashboardURL = "http://not-available-now"
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Etcd_sampleHandler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
	// try to get state from running job
	job := getEtcdOrchestrationJob (myServiceInfo.Url)
	if job != nil {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "In progress .",
		}, nil
	}
	
	// assume in provisioning
	
	// the job may be finished or interrupted or running in another instance.
	
	// check boot route, if it doesn't exist, return failed
	boot_res, _ := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if boot_res.route.Name == "" {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: "Failed!",
		}, nil
	}
	
	// only check the statuses of 3 ReplicationControllers. The etcd pods may be not running well.
	
	ok := func(rc *kapi.ReplicationController) bool {
		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
			return false
		}
		return true
	}
	
	ha_res, _ := getEtcdResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
	
	//println("num_ok_rcs = ", num_ok_rcs)
	
	if ok (&ha_res.etcdrc1) && ok (&ha_res.etcdrc2) && ok (&ha_res.etcdrc3) {
		return brokerapi.LastOperation{
			State:       brokerapi.Succeeded,
			Description: "Succeeded!",
		}, nil
	} else {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "In progress.",
		}, nil
	}
}

func (handler *Etcd_sampleHandler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	go func() {
		job := getEtcdOrchestrationJob (myServiceInfo.Url)
		if job != nil {
			job.cancel()
			
			// wait job to exit
			for {
				time.Sleep(7 * time.Second)
				if nil == getEtcdOrchestrationJob (myServiceInfo.Url) {
					break
				}
			}
		}
		
		// ...
		
		println("to destroy resources")
		
		ha_res, _ := getEtcdResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
		destroyEtcdResources_HA (ha_res, myServiceInfo.Database)
		
		boot_res, _ := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
		destroyEtcdResources_Boot (boot_res, myServiceInfo.Database, true)
	}()
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Etcd_sampleHandler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
	// output.route.Spec.Host
	
	boot_res, err := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if err != nil {
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	//if len(boot_res.service.Spec.Ports) == 0 {
	//	err := errors.New("no ports in boot service")
	//	logger.Error("", err)
	//	return brokerapi.Binding{}, Credentials{}, err
	//}
	
	etcd_addr, host, port := boot_res.endpoint()
	println("etcd addr: ", etcd_addr)
	etcd_addrs := []string{etcd_addr}
	
	etcd_client, err := newAuthrizedEtcdClient (etcd_addrs, "root", myServiceInfo.Password)
	if err != nil {
		logger.Error("create etcd authrized client", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}

	newusername := oshandler.NewElevenLengthID() // oshandler.GenGUID()[:16]
	newpassword := oshandler.GenGUID()
	
	etcd_userapi := etcd.NewAuthUserAPI(etcd_client)
	
	err = etcd_userapi.AddUser(context.Background(), newusername, newpassword)
	if err != nil {
		logger.Error("create new etcd user", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	_, err = etcd_userapi.GrantUser(context.Background(), newusername, []string{EtcdBindRole})
	if err != nil {
		logger.Error("grant new etcd user", err)
		
		err2 := etcd_userapi.RemoveUser(context.Background(), newusername)
		if err2 != nil {
			logger.Error("remove new etcd user", err2)
		}
		
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	// etcd bug: need to change password to make the user applied
	// todo: may this bug is already fixed now.
	_, err = etcd_userapi.ChangePassword(context.Background(), newusername, newpassword)
	if err != nil {
		logger.Error("change new user password", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	mycredentials := oshandler.Credentials{
		Uri:      etcd_addr,
		Hostname: host,
		Port:     port,
		Username: newusername,
		Password: newpassword,
	}

	myBinding := brokerapi.Binding{Credentials: mycredentials}

	return myBinding, mycredentials, nil
}

func (handler *Etcd_sampleHandler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
	boot_res, err := getEtcdResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if err != nil {
		return err
	}
	//if len(boot_res.service.Spec.Ports) == 0 {
	//     err := errors.New("no ports in boot service")
	//	logger.Error("", err)
	//	return err
	//}
	
	etcd_addr, _, _ := boot_res.endpoint()
	println("etcd addr: ", etcd_addr)
	etcd_addrs := []string{etcd_addr}
	
	etcd_client, err := newAuthrizedEtcdClient (etcd_addrs, "root", myServiceInfo.Password)
	if err != nil {
		logger.Error("create etcd authrized client", err)
		return err
	}
	
	etcd_userapi := etcd.NewAuthUserAPI(etcd_client)
	_, err = etcd_userapi.RevokeUser(context.Background(), mycredentials.Username, []string{EtcdBindRole})
	if err != nil {
		logger.Error("revoke role in unbinding", err)
		// return err
	}
	
	err = etcd_userapi.RemoveUser(context.Background(), mycredentials.Username)
	if err != nil {
		logger.Error("remove user", err)
		return err
	}
	
	return nil
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
			job.run()
			
			etcdOrchestrationJobsMutex.Lock()
			delete(etcdOrchestrationJobs, job.serviceInfo.Url)
			etcdOrchestrationJobsMutex.Unlock()
		}()
	}
}

type etcdOrchestrationJob struct {
	//instanceId string // use serviceInfo.
	
	cancelled bool
	cancelChan chan struct{}
	cancelMetex sync.Mutex
	
	isProvisioning bool // false for deprovisionings
	
	serviceInfo   *oshandler.ServiceInfo
	
	bootResources *etcdResources_Boot
	haResources   *etcdResources_HA
}

func (job *etcdOrchestrationJob) cancel() {
	job.cancelMetex.Lock()
	defer job.cancelMetex.Unlock()
	
	if ! job.cancelled {
		job.isProvisioning = false
		job.cancelled = true
		close (job.cancelChan)
	}
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
	statuses, cancel, err := oshandler.OC().KWatch (uri)
	if err != nil {
		logger.Error("start watching boot pod", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	for {
		var status oshandler.WatchStatus
		select {
		case <- job.cancelChan:
			close(cancel)
			return
		case status, _ = <- statuses:
			break
		}
		
		if status.Err != nil {
			close(cancel)
			
			logger.Error("watch boot pod error", status.Err)
			job.isProvisioning = false
			destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
			return
		} else {
			//logger.Debug("watch etcd pod, status.Info: " + string(status.Info))
		}
		
		var wps watchPodStatus
		if err := json.Unmarshal(status.Info, &wps); err != nil {
			close(cancel)
			
			logger.Error("parse boot pod status", err)
			job.isProvisioning = false
			destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
			return
		}
		
		if wps.Object.Status.Phase != kapi.PodPending {
			println("watch pod phase: ", wps.Object.Status.Phase)
			
			if wps.Object.Status.Phase != kapi.PodRunning {
				close(cancel)
				
				logger.Debug("pod phase is neither pending nor running")
				job.isProvisioning = false
				destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
				return
			}
			
			// running now, to create HA resources
			close(cancel)
			break
		}
	}
	
	// todo: check if seed pod is running
	
	// ...
	
	if job.cancelled { return }
	
	time.Sleep(7 * time.Second)
	
	if job.cancelled { return }
	
	etcd_addr, _, _ := job.bootResources.endpoint()
	println("etcd addr: ", etcd_addr)
	etcd_addrs := []string{etcd_addr}
	
	// etcd acl
	
	etcd_client, err := newUnauthrizedEtcdClient (etcd_addrs)
	if err != nil {
		logger.Error("create etcd unauthrized client", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	etcd_userapi := etcd.NewAuthUserAPI(etcd_client)
	err = etcd_userapi.AddUser(context.Background(), "root", serviceInfo.Password)
	if err != nil {
		logger.Error("create etcd root user", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	etcd_client, err = newAuthrizedEtcdClient (etcd_addrs, "root", serviceInfo.Password)
	if err != nil {
		logger.Error("create etcd authrized client", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	etcd_authapi := etcd.NewAuthAPI(etcd_client)
	err = etcd_authapi.Enable(context.Background())
	if err != nil {
		logger.Error("enable etcd auth", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	etcd_roleapi := etcd.NewAuthRoleAPI(etcd_client)
	_, err = etcd_roleapi.RevokeRoleKV(context.Background(), "guest", []string{"/*"}, etcd.ReadWritePermission)
	if err != nil {
		logger.Error("revoke guest role permission", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	err = etcd_roleapi.AddRole(context.Background(), EtcdBindRole)
	if err != nil {
		logger.Error("add etcd binduser role", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	_, err = etcd_roleapi.GrantRoleKV(context.Background(), EtcdBindRole, []string{"/*"}, etcd.ReadWritePermission)
	if err != nil {
		logger.Error("grant ectd binduser role", err)
		job.isProvisioning = false
		destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	if job.cancelled { return }
	
	// create HA resources
	
	err = job.createEtcdResources_HA (serviceInfo.Url, serviceInfo.Database, serviceInfo.Password)
	// todo: if err != nil
	
	// delete boot pod
	//println("to delete boot pod ...")
	//
	//time.Sleep(60 * time.Second)
	//if job.cancelled { return }
	//destroyEtcdResources_Boot (job.bootResources, serviceInfo.Database, false)
}

func newUnauthrizedEtcdClient (etcdEndPoints []string) (etcd.Client, error) {
	cfg := etcd.Config{
		Endpoints: etcdEndPoints,
		Transport: etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 15 * time.Second,
	}
	return etcd.New(cfg)
}

func newAuthrizedEtcdClient (etcdEndPoints []string, etcdUser, etcdPassword string) (etcd.Client, error) {
	cfg := etcd.Config{
		Endpoints: etcdEndPoints,
		Transport: etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 15 * time.Second,
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
		endpoint_postfix := oshandler.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			EtcdTemplateData_Boot = bytes.Replace(
				EtcdTemplateData_Boot, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
	}
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := EtcdTemplateData_Boot
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)	
	
	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.service).
		Decode(&res.route).
		Decode(&res.etcdpod).
		Decode(&res.etcdsrv)
	
	return decoder.Err
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
		
		etcd_image := oshandler.EtcdImage()
		etcd_image = strings.TrimSpace(etcd_image)
		if len(etcd_image) > 0 {
			EtcdTemplateData_HA = bytes.Replace(
				EtcdTemplateData_HA, 
				[]byte("http://etcd-image-place-holder/etcd-openshift-orchestration"), 
				[]byte(etcd_image), 
				-1)
		}
		endpoint_postfix := oshandler.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			EtcdTemplateData_HA = bytes.Replace(
				EtcdTemplateData_HA, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
	}
	
	yamlTemplates := EtcdTemplateData_HA
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("test1234"), []byte(rootPassword), -1)	
	
	//println("========= HA yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.etcdrc1).
		Decode(&res.etcdsrv1).
		Decode(&res.etcdrc2).
		Decode(&res.etcdsrv2).
		Decode(&res.etcdrc3).
		Decode(&res.etcdsrv3)
	
	return decoder.Err
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

func (bootRes *etcdResources_Boot) endpoint() (string, string, string) {
	port := "80" // strconv.Itoa(bootRes.service.Spec.Ports[0].Port)
	host := bootRes.route.Spec.Host
	return "http://" + net.JoinHostPort(host, port), host, port
}
	
func createEtcdResources_Boot (instanceId, serviceBrokerNamespace string) (*etcdResources_Boot, error) {
	var input etcdResources_Boot
	err := loadEtcdResources_Boot(instanceId, &input)
	if err != nil {
		return nil, err
	}
	
	var output etcdResources_Boot
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/services", &input.service, &output.service).
		OPost(prefix + "/routes", &input.route, &output.route).
		KPost(prefix + "/pods", &input.etcdpod, &output.etcdpod).
		KPost(prefix + "/services", &input.etcdsrv, &output.etcdsrv)
	
	if osr.Err != nil {
		logger.Error("createEtcdResources_Boot", osr.Err)
	}
	
	return &output, osr.Err
}
	
func getEtcdResources_Boot (instanceId, serviceBrokerNamespace string) (*etcdResources_Boot, error) {
	var output etcdResources_Boot
	
	var input etcdResources_Boot
	err := loadEtcdResources_Boot(instanceId, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.service.Name, &output.service).
		OGet(prefix + "/routes/" + input.route.Name, &output.route).
		KGet(prefix + "/pods/" + input.etcdpod.Name, &output.etcdpod).
		KGet(prefix + "/services/" + input.etcdsrv.Name, &output.etcdsrv)
	
	if osr.Err != nil {
		logger.Error("getEtcdResources_Boot", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyEtcdResources_Boot (bootRes *etcdResources_Boot, serviceBrokerNamespace string, all bool) {
	// todo: add to retry queue on fail
	
	if all {
		go func() {odel (serviceBrokerNamespace, "routes", bootRes.route.Name)}()
		go func() {kdel (serviceBrokerNamespace, "services", bootRes.service.Name)}()
	}
	
	go func() {kdel (serviceBrokerNamespace, "pods", bootRes.etcdpod.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", bootRes.etcdsrv.Name)}()
}
	
func (job *etcdOrchestrationJob) createEtcdResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) error {
	var input etcdResources_HA
	err := loadEtcdResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return err
	}
	
	var output etcdResources_HA
	
	/*
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/replicationcontrollers", &input.etcdrc1, &output.etcdrc1).
		KPost(prefix + "/services", &input.etcdsrv1, &output.etcdsrv1).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc2, &output.etcdrc2).
		KPost(prefix + "/services", &input.etcdsrv2, &output.etcdsrv2).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc3, &output.etcdrc3).
		KPost(prefix + "/services", &input.etcdsrv3, &output.etcdsrv3)
	
	if osr.Err != nil {
		logger.Error("createEtcdResources_HA", osr.Err)
	}
	
	return osr.Err
	*/
	go func() {
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.etcdrc1, &output.etcdrc1); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.etcdsrv1, &output.etcdsrv1); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.etcdrc2, &output.etcdrc2); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.etcdsrv2, &output.etcdsrv2); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.etcdrc3, &output.etcdrc3); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.etcdsrv3, &output.etcdsrv3); err != nil {
			return
		}
	}()
	
	return nil
}
	
func getEtcdResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) (*etcdResources_HA, error) {
	var output etcdResources_HA
	
	var input etcdResources_HA
	err := loadEtcdResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.etcdsrv1.Name, &output.etcdsrv1).
		KGet(prefix + "/replicationcontrollers/" + input.etcdrc1.Name, &output.etcdrc1).
		KGet(prefix + "/services/" + input.etcdsrv2.Name, &output.etcdsrv2).
		KGet(prefix + "/replicationcontrollers/" + input.etcdrc2.Name, &output.etcdrc2).
		KGet(prefix + "/services/" + input.etcdsrv3.Name, &output.etcdsrv3).
		KGet(prefix + "/replicationcontrollers/" + input.etcdrc3.Name, &output.etcdrc3)
	
	if osr.Err != nil {
		logger.Error("getEtcdResources_HA", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyEtcdResources_HA (haRes *etcdResources_HA, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.etcdsrv1.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.etcdrc1)}()
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.etcdsrv2.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.etcdrc2)}()
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.etcdsrv3.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.etcdrc3)}()
}

//===============================================================
// 
//===============================================================

func (job *etcdOrchestrationJob) kpost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	if job.cancelled {
		return nil
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).KPost(uri, body, into)
	if osr.Err == nil {
		logger.Info("create " + typeName + " succeeded")
	} else {
		i++
		if i < n {
			logger.Error(fmt.Sprintf("%d> create (%s) error", i, typeName), osr.Err)
			goto RETRY
		} else {
			logger.Error(fmt.Sprintf("create (%s) failed", typeName), osr.Err)
			return osr.Err
		}
	}
	
	return nil
}

func (job *etcdOrchestrationJob) opost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	if job.cancelled {
		return nil
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).OPost(uri, body, into)
	if osr.Err == nil {
		logger.Info("create " + typeName + " succeeded")
	} else {
		i++
		if i < n {
			logger.Error(fmt.Sprintf("%d> create (%s) error", i, typeName), osr.Err)
			goto RETRY
		} else {
			logger.Error(fmt.Sprintf("create (%s) failed", typeName), osr.Err)
			return osr.Err
		}
	}
	
	return nil
}
	
func kdel (serviceBrokerNamespace, typeName, resName string) error {
	if resName == "" {
		return nil
	}
	
	println("to delete ", typeName, "/", resName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s/%s", serviceBrokerNamespace, typeName, resName)
	i, n := 0, 5
RETRY:
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).KDelete(uri, nil)
	if osr.Err == nil {
		logger.Info("delete " + uri + " succeeded")
	} else {
		i++
		if i < n {
			logger.Error(fmt.Sprintf("%d> delete (%s) error", i, uri), osr.Err)
			goto RETRY
		} else {
			logger.Error(fmt.Sprintf("delete (%s) failed", uri), osr.Err)
			return osr.Err
		}
	}
	
	return nil
}

func odel (serviceBrokerNamespace, typeName, resName string) error {
	if resName == "" {
		return nil
	}
	
	println("to delete ", typeName, "/", resName)	

	uri := fmt.Sprintf("/namespaces/%s/%s/%s", serviceBrokerNamespace, typeName, resName)
	i, n := 0, 5
RETRY:
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).ODelete(uri, nil)
	if osr.Err == nil {
		logger.Info("delete " + uri + " succeeded")
	} else {
		i++
		if i < n {
			logger.Error(fmt.Sprintf("%d> delete (%s) error", i, uri), osr.Err)
			goto RETRY
		} else {
			logger.Error(fmt.Sprintf("delete (%s) failed", uri), osr.Err)
			return osr.Err
		}
	}
	
	return nil
}

/*
func kdel_rc (serviceBrokerNamespace string, rc *kapi.ReplicationController) {
	kdel (serviceBrokerNamespace, "replicationcontrollers", rc.Name)
}
*/

func kdel_rc (serviceBrokerNamespace string, rc *kapi.ReplicationController) {
	// looks pods will be auto deleted when rc is deleted.
	
	if rc == nil || rc.Name == "" {
		return
	}
	
	println("to delete pods on replicationcontroller", rc.Name)
	
	uri := "/namespaces/" + serviceBrokerNamespace + "/replicationcontrollers/" + rc.Name
	
	// modfiy rc replicas to 0
	
	zero := 0
	rc.Spec.Replicas = &zero
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).KPut(uri, rc, nil)
	if osr.Err != nil {
		logger.Error("modify HA rc", osr.Err)
		return
	}
	
	// start watching rc status
	
	statuses, cancel, err := oshandler.OC().KWatch (uri)
	if err != nil {
		logger.Error("start watching HA rc", err)
		return
	}
	
	go func() {
		for {
			status, _ := <- statuses
			
			if status.Err != nil {
				logger.Error("watch HA etcd rc error", status.Err)
				close(cancel)
				return
			} else {
				//logger.Debug("watch etcd HA rc, status.Info: " + string(status.Info))
			}
			
			var wrcs watchReplicationControllerStatus
			if err := json.Unmarshal(status.Info, &wrcs); err != nil {
				logger.Error("parse boot HA rc status", err)
				close(cancel)
				return
			}
			
			if wrcs.Object.Status.Replicas <= 0 {
				break
			}
		}
		
		// ...
		
		kdel(serviceBrokerNamespace, "replicationcontrollers", rc.Name)
	}()
	
	return
}

type watchReplicationControllerStatus struct {
	// The type of watch update contained in the message
	Type string `json:"type"`
	// RC details
	Object kapi.ReplicationController `json:"object"`
}
