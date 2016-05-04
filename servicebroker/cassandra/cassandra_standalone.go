package cassandra

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
	//cassandra "github.com/gocql/gocql"
	"golang.org/x/net/context"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
	
	oshandler "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
)

//==============================================================
// 
//==============================================================

const CassandraServcieBrokerName_Standalone = "Cassandra_standalone"

func init() {
	oshandler.Register(CassandraServcieBrokerName_Standalone, &Cassandra_sampleHandler{})
	
	logger = lager.NewLogger(CassandraServcieBrokerName_Standalone)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

type Cassandra_sampleHandler struct{}

func (handler *Cassandra_sampleHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
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
	
	// boot cassandra
	
	output, err := createCassandraResources_Boot(instanceIdInTempalte, serviceBrokerNamespace)

	if err != nil {
		destroyCassandraResources_Boot(output, serviceBrokerNamespace, true)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.Password = oshandler.GenGUID()
	
	// todo: improve watch. Pod may be already running before watching!
	startCassandraOrchestrationJob(&cassandraOrchestrationJob{
		cancelled:  false,
		cancelChan: make(chan struct{}),
		
		isProvisioning: true,
		serviceInfo:    &serviceInfo,
		bootResources:  output,
		//haResources:    nil,
	})
	
	serviceSpec.DashboardURL = "http://not-available-now"
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Cassandra_sampleHandler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
	// try to get state from running job
	job := getCassandraOrchestrationJob (myServiceInfo.Url)
	if job != nil {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "In progress .",
		}, nil
	}
	
	// assume in provisioning
	
	// the job may be finished or interrupted or running in another instance.
	
	// check boot route, if it doesn't exist, return failed
	boot_res, _ := getCassandraResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if boot_res.route.Name == "" {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: "Failed!",
		}, nil
	}
	
	// only check the statuses of 3 ReplicationControllers. The cassandra pods may be not running well.
	
	ok := func(rc *kapi.ReplicationController) bool {
		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
			return false
		}
		return true
	}
	
	ha_res, _ := getCassandraResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
	
	//println("num_ok_rcs = ", num_ok_rcs)
	
	if ok (&ha_res.cassandrarc1) && ok (&ha_res.cassandrarc2) && ok (&ha_res.cassandrarc3) {
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

func (handler *Cassandra_sampleHandler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	go func() {
		job := getCassandraOrchestrationJob (myServiceInfo.Url)
		if job != nil {
			job.cancel()
			
			// wait job to exit
			for {
				time.Sleep(7 * time.Second)
				if nil == getCassandraOrchestrationJob (myServiceInfo.Url) {
					break
				}
			}
		}
		
		// ...
		
		println("to destroy resources")
		
		ha_res, _ := getCassandraResources_HA (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
		destroyCassandraResources_HA (ha_res, myServiceInfo.Database)
		
		boot_res, _ := getCassandraResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
		destroyCassandraResources_Boot (boot_res, myServiceInfo.Database, true)
	}()
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Cassandra_sampleHandler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
	// output.route.Spec.Host
	
	boot_res, err := getCassandraResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if err != nil {
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	//if len(boot_res.service.Spec.Ports) == 0 {
	//	err := errors.New("no ports in boot service")
	//	logger.Error("", err)
	//	return brokerapi.Binding{}, Credentials{}, err
	//}
	
	cassandra_addr, host, port := boot_res.endpoint()
	println("cassandra addr: ", cassandra_addr)
	cassandra_addrs := []string{cassandra_addr}

	newusername := oshandler.NewElevenLengthID() // oshandler.GenGUID()[:16]
	newpassword := oshandler.GenGUID()
	
	/*
	cassandra_client, err := newAuthrizedCassandraClient (cassandra_addrs, "root", myServiceInfo.Password)
	if err != nil {
		logger.Error("create cassandra authrized client", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	cassandra_userapi := cassandra.NewAuthUserAPI(cassandra_client)
	
	err = cassandra_userapi.AddUser(context.Background(), newusername, newpassword)
	if err != nil {
		logger.Error("create new cassandra user", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	_, err = cassandra_userapi.GrantUser(context.Background(), newusername, []string{CassandraBindRole})
	if err != nil {
		logger.Error("grant new cassandra user", err)
		
		err2 := cassandra_userapi.RemoveUser(context.Background(), newusername)
		if err2 != nil {
			logger.Error("remove new cassandra user", err2)
		}
		
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	// cassandra bug: need to change password to make the user applied
	// todo: may this bug is already fixed now.
	_, err = cassandra_userapi.ChangePassword(context.Background(), newusername, newpassword)
	if err != nil {
		logger.Error("change new user password", err)
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	*/
	
	mycredentials := oshandler.Credentials{
		Uri:      cassandra_addr,
		Hostname: host,
		Port:     port,
		Username: newusername,
		Password: newpassword,
	}

	myBinding := brokerapi.Binding{Credentials: mycredentials}

	return myBinding, mycredentials, nil
}

func (handler *Cassandra_sampleHandler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
	boot_res, err := getCassandraResources_Boot (myServiceInfo.Url, myServiceInfo.Database)
	if err != nil {
		return err
	}
	//if len(boot_res.service.Spec.Ports) == 0 {
	//     err := errors.New("no ports in boot service")
	//	logger.Error("", err)
	//	return err
	//}
	
	cassandra_addr, _, _ := boot_res.endpoint()
	println("cassandra addr: ", cassandra_addr)
	cassandra_addrs := []string{cassandra_addr}
	
	/*
	cassandra_client, err := newAuthrizedCassandraClient (cassandra_addrs, "root", myServiceInfo.Password)
	if err != nil {
		logger.Error("create cassandra authrized client", err)
		return err
	}
	
	cassandra_userapi := cassandra.NewAuthUserAPI(cassandra_client)
	_, err = cassandra_userapi.RevokeUser(context.Background(), mycredentials.Username, []string{CassandraBindRole})
	if err != nil {
		logger.Error("revoke role in unbinding", err)
		// return err
	}
	
	err = cassandra_userapi.RemoveUser(context.Background(), mycredentials.Username)
	if err != nil {
		logger.Error("remove user", err)
		return err
	}
	*/
	
	return nil
}

//==============================================================
// 
//==============================================================

var cassandraOrchestrationJobs = map[string]*cassandraOrchestrationJob{}
var cassandraOrchestrationJobsMutex sync.Mutex

func getCassandraOrchestrationJob (instanceId string) *cassandraOrchestrationJob {
	cassandraOrchestrationJobsMutex.Lock()
	defer cassandraOrchestrationJobsMutex.Unlock()
	
	return cassandraOrchestrationJobs[instanceId]
}

func startCassandraOrchestrationJob (job *cassandraOrchestrationJob) {
	cassandraOrchestrationJobsMutex.Lock()
	defer cassandraOrchestrationJobsMutex.Unlock()
	
	if cassandraOrchestrationJobs[job.serviceInfo.Url] == nil {
		cassandraOrchestrationJobs[job.serviceInfo.Url] = job
		go func() {
			job.run()
			
			cassandraOrchestrationJobsMutex.Lock()
			delete(cassandraOrchestrationJobs, job.serviceInfo.Url)
			cassandraOrchestrationJobsMutex.Unlock()
		}()
	}
}

type cassandraOrchestrationJob struct {
	//instanceId string // use serviceInfo.
	
	cancelled bool
	cancelChan chan struct{}
	cancelMetex sync.Mutex
	
	isProvisioning bool // false for deprovisionings
	
	serviceInfo   *oshandler.ServiceInfo
	
	bootResources *cassandraResources_Boot
	//haResources   *cassandraResources_HA
}

func (job *cassandraOrchestrationJob) cancel() {
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
//1. create first cassandra pod/service/route (aka. a single node cassandra cluster)
//   watch pod status to running
//2. modify cassandra acl (add root, remove gust, add bind role)
//3. create HA resources
//4. watch HA pods, when all are running, delete boot pod
//   (or add 2nd and 3rd cassandra pod, so no need to delete boot pod)

func (job *cassandraOrchestrationJob) run() {
	serviceInfo := job.serviceInfo
	pod := job.bootResources.cassandrapod
	uri := "/namespaces/" + serviceInfo.Database + "/pods/" + pod.Name
	statuses, cancel, err := oshandler.OC().KWatch (uri)
	if err != nil {
		logger.Error("start watching boot pod", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
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
			destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
			return
		} else {
			//logger.Debug("watch cassandra pod, status.Info: " + string(status.Info))
		}
		
		var wps watchPodStatus
		if err := json.Unmarshal(status.Info, &wps); err != nil {
			close(cancel)
			
			logger.Error("parse boot pod status", err)
			job.isProvisioning = false
			destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
			return
		}
		
		if wps.Object.Status.Phase != kapi.PodPending {
			println("watch pod phase: ", wps.Object.Status.Phase)
			
			if wps.Object.Status.Phase != kapi.PodRunning {
				close(cancel)
				
				logger.Debug("pod phase is neither pending nor running")
				job.isProvisioning = false
				destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
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
	
	cassandra_addr, _, _ := job.bootResources.endpoint()
	println("cassandra addr: ", cassandra_addr)
	cassandra_addrs := []string{cassandra_addr}
	
	// cassandra acl
	
	/*
	cassandra_client, err := newUnauthrizedCassandraClient (cassandra_addrs)
	if err != nil {
		logger.Error("create cassandra unauthrized client", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	cassandra_userapi := cassandra.NewAuthUserAPI(cassandra_client)
	err = cassandra_userapi.AddUser(context.Background(), "root", serviceInfo.Password)
	if err != nil {
		logger.Error("create cassandra root user", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	cassandra_client, err = newAuthrizedCassandraClient (cassandra_addrs, "root", serviceInfo.Password)
	if err != nil {
		logger.Error("create cassandra authrized client", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	cassandra_authapi := cassandra.NewAuthAPI(cassandra_client)
	err = cassandra_authapi.Enable(context.Background())
	if err != nil {
		logger.Error("enable cassandra auth", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	cassandra_roleapi := cassandra.NewAuthRoleAPI(cassandra_client)
	_, err = cassandra_roleapi.RevokeRoleKV(context.Background(), "guest", []string{"/*"}, cassandra.ReadWritePermission)
	if err != nil {
		logger.Error("revoke guest role permission", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	err = cassandra_roleapi.AddRole(context.Background(), CassandraBindRole)
	if err != nil {
		logger.Error("add cassandra binduser role", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	
	_, err = cassandra_roleapi.GrantRoleKV(context.Background(), CassandraBindRole, []string{"/*"}, cassandra.ReadWritePermission)
	if err != nil {
		logger.Error("grant ectd binduser role", err)
		job.isProvisioning = false
		destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, true)
		return
	}
	*/
	
	if job.cancelled { return }
	
	// create HA resources
	
	err = job.createCassandraResources_HA (serviceInfo.Url, serviceInfo.Database, serviceInfo.Password)
	// todo: if err != nil
	
	// delete boot pod
	//println("to delete boot pod ...")
	//
	//time.Sleep(60 * time.Second)
	//if job.cancelled { return }
	//destroyCassandraResources_Boot (job.bootResources, serviceInfo.Database, false)
}

/*
func newUnauthrizedCassandraClient (cassandraEndPoints []string) (cassandra.Client, error) {
	cfg := cassandra.Config{
		Endpoints: cassandraEndPoints,
		Transport: cassandra.DefaultTransport,
		HeaderTimeoutPerRequest: 15 * time.Second,
	}
	return cassandra.New(cfg)
}

func newAuthrizedCassandraClient (cassandraEndPoints []string, cassandraUser, cassandraPassword string) (cassandra.Client, error) {
	cfg := cassandra.Config{
		Endpoints: cassandraEndPoints,
		Transport: cassandra.DefaultTransport,
		HeaderTimeoutPerRequest: 15 * time.Second,
		Username:                cassandraUser,
		Password:                cassandraPassword,
	}
	return cassandra.New(cfg)
}
*/

//===============================================================
// 
//===============================================================

var CassandraTemplateData_Boot []byte = nil

func loadCassandraResources_Boot(instanceID string, res *cassandraResources_Boot) error {
	if CassandraTemplateData_Boot == nil {
		f, err := os.Open("cassandra-outer-boot.yaml")
		if err != nil {
			return err
		}
		CassandraTemplateData_Boot, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		
		cassandra_image := oshandler.CassandraImage()
		cassandra_image = strings.TrimSpace(cassandra_image)
		if len(cassandra_image) > 0 {
			CassandraTemplateData_HA = bytes.Replace(
				CassandraTemplateData_HA, 
				[]byte("http://cassandra-image-place-holder/cassandra-openshift-orchestration"), 
				[]byte(cassandra_image), 
				-1)
		}
		endpoint_postfix := oshandler.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			CassandraTemplateData_Boot = bytes.Replace(
				CassandraTemplateData_Boot, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
	}
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := CassandraTemplateData_Boot
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)	
	
	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.service).
		Decode(&res.route).
		Decode(&res.cassandrapod).
		Decode(&res.cassandrasrv)
	
	return decoder.Err
}

var CassandraTemplateData_HA []byte = nil

func loadCassandraResources_HA(instanceID, rootPassword string, res *cassandraResources_HA) error {
	if CassandraTemplateData_HA == nil {
		f, err := os.Open("cassandra-outer-ha.yaml")
		if err != nil {
			return err
		}
		CassandraTemplateData_HA, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		
		cassandra_image := oshandler.CassandraImage()
		cassandra_image = strings.TrimSpace(cassandra_image)
		if len(cassandra_image) > 0 {
			CassandraTemplateData_HA = bytes.Replace(
				CassandraTemplateData_HA, 
				[]byte("http://cassandra-image-place-holder/cassandra-openshift-orchestration"), 
				[]byte(cassandra_image), 
				-1)
		}
	}
	
	yamlTemplates := CassandraTemplateData_HA
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	
	//println("========= HA yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.cassandrarc1).
		Decode(&res.cassandrasrv1).
		Decode(&res.cassandrarc2).
		Decode(&res.cassandrasrv2).
		Decode(&res.cassandrarc3).
		Decode(&res.cassandrasrv3)
	
	return decoder.Err
}

type cassandraResources_Boot struct {
	service kapi.Service
	route   routeapi.Route
	cassandrapod kapi.Pod
	cassandrasrv kapi.Service
}

type cassandraResources_HA struct {
	cassandrarc1  kapi.ReplicationController
	cassandrasrv1 kapi.Service
	cassandrarc2  kapi.ReplicationController
	cassandrasrv2 kapi.Service
	cassandrarc3  kapi.ReplicationController
	cassandrasrv3 kapi.Service
}

func (bootRes *cassandraResources_Boot) endpoint() (string, string, string) {
	port := "80" // strconv.Itoa(bootRes.service.Spec.Ports[0].Port)
	host := bootRes.route.Spec.Host
	return "http://" + net.JoinHostPort(host, port), host, port
}
	
func createCassandraResources_Boot (instanceId, serviceBrokerNamespace string) (*cassandraResources_Boot, error) {
	var input cassandraResources_Boot
	err := loadCassandraResources_Boot(instanceId, &input)
	if err != nil {
		return nil, err
	}
	
	var output cassandraResources_Boot
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/services", &input.service, &output.service).
		OPost(prefix + "/routes", &input.route, &output.route).
		KPost(prefix + "/pods", &input.cassandrapod, &output.cassandrapod).
		KPost(prefix + "/services", &input.cassandrasrv, &output.cassandrasrv)
	
	if osr.Err != nil {
		logger.Error("createCassandraResources_Boot", osr.Err)
	}
	
	return &output, osr.Err
}
	
func getCassandraResources_Boot (instanceId, serviceBrokerNamespace string) (*cassandraResources_Boot, error) {
	var output cassandraResources_Boot
	
	var input cassandraResources_Boot
	err := loadCassandraResources_Boot(instanceId, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.service.Name, &output.service).
		OGet(prefix + "/routes/" + input.route.Name, &output.route).
		KGet(prefix + "/pods/" + input.cassandrapod.Name, &output.cassandrapod).
		KGet(prefix + "/services/" + input.cassandrasrv.Name, &output.cassandrasrv)
	
	if osr.Err != nil {
		logger.Error("getCassandraResources_Boot", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyCassandraResources_Boot (bootRes *cassandraResources_Boot, serviceBrokerNamespace string, all bool) {
	// todo: add to retry queue on fail
	
	if all {
		go func() {odel (serviceBrokerNamespace, "routes", bootRes.route.Name)}()
		go func() {kdel (serviceBrokerNamespace, "services", bootRes.service.Name)}()
	}
	
	go func() {kdel (serviceBrokerNamespace, "pods", bootRes.cassandrapod.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", bootRes.cassandrasrv.Name)}()
}
	
func (job *cassandraOrchestrationJob) createCassandraResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) error {
	var input cassandraResources_HA
	err := loadCassandraResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return err
	}
	
	var output cassandraResources_HA
	
	/*
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/replicationcontrollers", &input.cassandrarc1, &output.cassandrarc1).
		KPost(prefix + "/services", &input.cassandrasrv1, &output.cassandrasrv1).
		KPost(prefix + "/replicationcontrollers", &input.cassandrarc2, &output.cassandrarc2).
		KPost(prefix + "/services", &input.cassandrasrv2, &output.cassandrasrv2).
		KPost(prefix + "/replicationcontrollers", &input.cassandrarc3, &output.cassandrarc3).
		KPost(prefix + "/services", &input.cassandrasrv3, &output.cassandrasrv3)
	
	if osr.Err != nil {
		logger.Error("createCassandraResources_HA", osr.Err)
	}
	
	return osr.Err
	*/
	go func() {
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.cassandrarc1, &output.cassandrarc1); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.cassandrasrv1, &output.cassandrasrv1); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.cassandrarc2, &output.cassandrarc2); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.cassandrasrv2, &output.cassandrasrv2); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.cassandrarc3, &output.cassandrarc3); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.cassandrasrv3, &output.cassandrasrv3); err != nil {
			return
		}
	}()
	
	return nil
}
	
func getCassandraResources_HA (instanceId, serviceBrokerNamespace, rootPassword string) (*cassandraResources_HA, error) {
	var output cassandraResources_HA
	
	var input cassandraResources_HA
	err := loadCassandraResources_HA(instanceId, rootPassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.cassandrasrv1.Name, &output.cassandrasrv1).
		KGet(prefix + "/replicationcontrollers/" + input.cassandrarc1.Name, &output.cassandrarc1).
		KGet(prefix + "/services/" + input.cassandrasrv2.Name, &output.cassandrasrv2).
		KGet(prefix + "/replicationcontrollers/" + input.cassandrarc2.Name, &output.cassandrarc2).
		KGet(prefix + "/services/" + input.cassandrasrv3.Name, &output.cassandrasrv3).
		KGet(prefix + "/replicationcontrollers/" + input.cassandrarc3.Name, &output.cassandrarc3)
	
	if osr.Err != nil {
		logger.Error("getCassandraResources_HA", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyCassandraResources_HA (haRes *cassandraResources_HA, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.cassandrasrv1.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.cassandrarc1)}()
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.cassandrasrv2.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.cassandrarc2)}()
	
	go func() {kdel (serviceBrokerNamespace, "services", haRes.cassandrasrv3.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &haRes.cassandrarc3)}()
}

//===============================================================
// 
//===============================================================

func (job *cassandraOrchestrationJob) kpost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
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

func (job *cassandraOrchestrationJob) opost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
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
				logger.Error("watch HA cassandra rc error", status.Err)
				close(cancel)
				return
			} else {
				//logger.Debug("watch cassandra HA rc, status.Info: " + string(status.Info))
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
