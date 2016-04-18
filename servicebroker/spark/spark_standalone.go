package spark


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
	"strconv"
	"strings"
	"bytes"
	"encoding/json"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	"sync"
	
	"github.com/pivotal-golang/lager"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
	
	oshandlder "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
)

//==============================================================
// 
//==============================================================

const SparkServcieBrokerName_Free = "spark_openshift_free"
const SparkServcieBrokerName_HighAvailable = "spark_openshift_highavailable"

func init() {
	oshandlder.Register(SparkServcieBrokerName_Free, &Spark_freeHandler{})
	oshandlder.Register(SparkServcieBrokerName_HighAvailable, &Spark_haHandler{})
	
	logger = lager.NewLogger("spark_openshift")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

type Spark_freeHandler struct{}

func (handler *Spark_freeHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandlder.ServiceInfo, error) {
	return newSparkHandler(1).DoProvision(instanceID, details, asyncAllowed)
}

func (handler *Spark_freeHandler) DoLastOperation(myServiceInfo *oshandlder.ServiceInfo) (brokerapi.LastOperation, error) {
	return newSparkHandler(1).DoLastOperation(myServiceInfo)
}

func (handler *Spark_freeHandler) DoDeprovision(myServiceInfo *oshandlder.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	return newSparkHandler(1).DoDeprovision(myServiceInfo, asyncAllowed)
}

func (handler *Spark_freeHandler) DoBind(myServiceInfo *oshandlder.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandlder.Credentials, error) {
	return newSparkHandler(1).DoBind(myServiceInfo, bindingID, details)
}

func (handler *Spark_freeHandler) DoUnbind(myServiceInfo *oshandlder.ServiceInfo, mycredentials *oshandlder.Credentials) error {
	return newSparkHandler(1).DoUnbind(myServiceInfo, mycredentials)
}

//...

type Spark_haHandler struct{}

func (handler *Spark_haHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandlder.ServiceInfo, error) {
	return newSparkHandler(3).DoProvision(instanceID, details, asyncAllowed)
}

func (handler *Spark_haHandler) DoLastOperation(myServiceInfo *oshandlder.ServiceInfo) (brokerapi.LastOperation, error) {
	return newSparkHandler(3).DoLastOperation(myServiceInfo)
}

func (handler *Spark_haHandler) DoDeprovision(myServiceInfo *oshandlder.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	return newSparkHandler(3).DoDeprovision(myServiceInfo, asyncAllowed)
}

func (handler *Spark_haHandler) DoBind(myServiceInfo *oshandlder.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandlder.Credentials, error) {
	return newSparkHandler(3).DoBind(myServiceInfo, bindingID, details)
}

func (handler *Spark_haHandler) DoUnbind(myServiceInfo *oshandlder.ServiceInfo, mycredentials *oshandlder.Credentials) error {
	return newSparkHandler(3).DoUnbind(myServiceInfo, mycredentials)
}

//==============================================================
// 
//==============================================================

type Spark_Handler struct{
	numWorkers int
}

func newSparkHandler(numWorkers int) *Spark_Handler {
	return &Spark_Handler{
			numWorkers: numWorkers,
		}
}

func (handler *Spark_Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandlder.ServiceInfo, error) {
	//初始化到openshift的链接
	
	serviceSpec := brokerapi.ProvisionedServiceSpec{IsAsync: asyncAllowed}
	serviceInfo := oshandlder.ServiceInfo{}
	
	//if asyncAllowed == false {
	//	return serviceSpec, serviceInfo, errors.New("Sync mode is not supported")
	//}
	serviceSpec.IsAsync = true
	
	//instanceIdInTempalte   := instanceID // todo: ok?
	instanceIdInTempalte   := strings.ToLower(oshandlder.NewThirteenLengthID())
	//serviceBrokerNamespace := ServiceBrokerNamespace
	serviceBrokerNamespace := oshandlder.OC().Namespace()
	sparkSecret := oshandlder.GenGUID()
	
	println()
	println("instanceIdInTempalte = ", instanceIdInTempalte)
	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
	println()
	
	// master spark
	
	output, err := createSparkResources_Master(instanceIdInTempalte, serviceBrokerNamespace, sparkSecret)

	if err != nil {
		destroySparkResources_Master(output, serviceBrokerNamespace)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	//serviceInfo.User = oshandlder.NewElevenLengthID()
	serviceInfo.Password = sparkSecret
	
	startSparkOrchestrationJob(&sparkOrchestrationJob{
		cancelled:  false,
		cancelChan: make(chan struct{}),
		
		isProvisioning:    true,
		serviceInfo:       &serviceInfo,
		planNumWorkers:    handler.numWorkers,
		masterResources:   output,
		slavesResources:   nil,
		zeppelinResources: nil,
	})
	
	serviceSpec.DashboardURL = output.webroute.Spec.Host
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Spark_Handler) DoLastOperation(myServiceInfo *oshandlder.ServiceInfo) (brokerapi.LastOperation, error) {
	// try to get state from running job
	job := getSparkOrchestrationJob (myServiceInfo.Url)
	if job != nil {
		return brokerapi.LastOperation{
			State:       brokerapi.InProgress,
			Description: "In progress .",
		}, nil
	}
	
	// assume in provisioning
	
	// the job may be finished or interrupted or running in another instance.
	
	// check master route, if it doesn't exist, return failed
	master_res, _ := getSparkResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
	if master_res.webroute.Name == "" {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: "Failed!",
		}, nil
	}
	
	// check workers
	
	workers_res, _ := getSparkResources_Workers (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password, handler.numWorkers)
	
	if workers_res.workerrc.Name == "" || workers_res.workerrc.Spec.Replicas == nil || workers_res.workerrc.Status.Replicas < *workers_res.workerrc.Spec.Replicas {
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
	
	// todo: check zeppelin
}

func (handler *Spark_Handler) DoDeprovision(myServiceInfo *oshandlder.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	go func() {
		job := getSparkOrchestrationJob (myServiceInfo.Url)
		if job != nil {
			job.cancel()
			
			// wait job to exit
			for {
				time.Sleep(7 * time.Second)
				if nil == getSparkOrchestrationJob (myServiceInfo.Url) {
					break
				}
			}
		}
		
		// ...
		
		println("to destroy resources")
		
		zeppelin_res, _ := getSparkResources_Zeppelin (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
		destroySparkResources_Zeppelin (zeppelin_res, myServiceInfo.Database)
		
		workers_res, _ := getSparkResources_Workers (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password, handler.numWorkers)
		destroySparkResources_Workers (workers_res, myServiceInfo.Database)
		
		master_res, _ := getSparkResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
		destroySparkResources_Master (master_res, myServiceInfo.Database)
	}()
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Spark_Handler) DoBind(myServiceInfo *oshandlder.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandlder.Credentials, error) {
	// todo: handle errors
	
	master_res, _ := getSparkResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
	
	zeppelin_res, _ := getSparkResources_Zeppelin (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.Password)
	
	mycredentials := oshandlder.Credentials{
		Uri:      "http://" + zeppelin_res.route.Spec.Host,
		Hostname: master_res.webroute.Spec.Host,
		Port:     "80",
		Username: "",
		Password: myServiceInfo.Password,
	}

	myBinding := brokerapi.Binding{Credentials: mycredentials}

	return myBinding, mycredentials, nil
}

func (handler *Spark_Handler) DoUnbind(myServiceInfo *oshandlder.ServiceInfo, mycredentials *oshandlder.Credentials) error {
	// do nothing
	
	return nil
}

//===============================================================
// 
//===============================================================

var sparkOrchestrationJobs = map[string]*sparkOrchestrationJob{}
var sparkOrchestrationJobsMutex sync.Mutex

func getSparkOrchestrationJob (instanceId string) *sparkOrchestrationJob {
	sparkOrchestrationJobsMutex.Lock()
	defer sparkOrchestrationJobsMutex.Unlock()
	
	return sparkOrchestrationJobs[instanceId]
}

func startSparkOrchestrationJob (job *sparkOrchestrationJob) {
	sparkOrchestrationJobsMutex.Lock()
	defer sparkOrchestrationJobsMutex.Unlock()
	
	if sparkOrchestrationJobs[job.serviceInfo.Url] == nil {
		sparkOrchestrationJobs[job.serviceInfo.Url] = job
		go func() {
			job.run()
			
			sparkOrchestrationJobsMutex.Lock()
			delete(sparkOrchestrationJobs, job.serviceInfo.Url)
			sparkOrchestrationJobsMutex.Unlock()
		}()
	}
}

type sparkOrchestrationJob struct {
	//instanceId string // use serviceInfo.
	
	cancelled bool
	cancelChan chan struct{}
	cancelMetex sync.Mutex
	
	isProvisioning bool // false for deprovisionings
	
	serviceInfo    *oshandlder.ServiceInfo
	planNumWorkers int
	
	masterResources *sparkResources_Master
	slavesResources   *sparkResources_Workers
	zeppelinResources   *sparkResources_Zeppelin
}

func (job *sparkOrchestrationJob) cancel() {
	job.cancelMetex.Lock()
	defer job.cancelMetex.Unlock()
	
	if ! job.cancelled {
		job.cancelled = true
		close (job.cancelChan)
	}
}

type watchReplicationControllerStatus struct {
	// The type of watch update contained in the message
	Type string `json:"type"`
	// Pod details
	Object kapi.ReplicationController `json:"object"`
}

func (job *sparkOrchestrationJob) run() {
	serviceInfo := job.serviceInfo
	rc := job.masterResources.masterrc
	uri := "/namespaces/" + serviceInfo.Database + "/replicationcontrollers/" + rc.Name
	statuses, cancel, err := oshandlder.OC().KWatch (uri)
	if err != nil {
		logger.Error("start watching master rc", err)
		job.isProvisioning = false
		destroySparkResources_Master (job.masterResources, serviceInfo.Database)
		return
	}
	
	for {
		var status oshandlder.WatchStatus
		select {
		case <- job.cancelChan:
			close(cancel)
			return
		case status, _ = <- statuses:
			break
		}
		
		if status.Err != nil {
			close(cancel)
			
			logger.Error("watch master rc error", status.Err)
			job.isProvisioning = false
			destroySparkResources_Master (job.masterResources, serviceInfo.Database)
			return
		} else {
			//logger.Debug("watch etcd pod, status.Info: " + string(status.Info))
		}
		
		var wrcs watchReplicationControllerStatus
		if err := json.Unmarshal(status.Info, &wrcs); err != nil {
			close(cancel)
			
			logger.Error("parse master rc status", err)
			job.isProvisioning = false
			destroySparkResources_Master (job.masterResources, serviceInfo.Database)
			return
		}
		
		if wrcs.Object.Spec.Replicas == nil {
			close(cancel)
			
			logger.Error("master rc error", err)
			job.isProvisioning = false
			destroySparkResources_Master (job.masterResources, serviceInfo.Database)
			return
		}
		
		println("watch master rs status.replicas: ", wrcs.Object.Status.Replicas)
		
		if wrcs.Object.Status.Replicas >= *wrcs.Object.Spec.Replicas {
			// master running now, to create worker resources
			close(cancel)
			break
		}
	}
	
	if job.cancelled { return }
	
	time.Sleep(7 * time.Second)
	
	if job.cancelled { return }
	
	// create worker resources
	
	err = job.createSparkResources_Workers (serviceInfo.Url, serviceInfo.Database, serviceInfo.Password)
	if err != nil {
		// todo
	}
	
	// create worker resources
	
	err = job.createSparkResources_Zeppelin (serviceInfo.Url, serviceInfo.Database, serviceInfo.Password)
	if err != nil {
		// todo
	}
}

//=======================================================================
// 
//=======================================================================

var SparkTemplateData_Master []byte = nil

func loadSparkResources_Master(instanceID, sparkSecret string, res *sparkResources_Master) error {
	if SparkTemplateData_Master == nil {
		f, err := os.Open("spark-master.yaml")
		if err != nil {
			return err
		}
		SparkTemplateData_Master, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		endpoint_postfix := oshandlder.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			SparkTemplateData_Master = bytes.Replace(
				SparkTemplateData_Master, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
	}
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := SparkTemplateData_Master
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("test1234"), []byte(sparkSecret), -1)	
	
	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandlder.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.masterrc).
		Decode(&res.mastersvc).
		Decode(&res.webroute).
		Decode(&res.websvc)
	
	return decoder.Err
}

var SparkTemplateData_Workers []byte = nil

func loadSparkResources_Workers(instanceID, sparkSecret string, numWorkers int, res *sparkResources_Workers) error {
	if SparkTemplateData_Workers == nil {
		f, err := os.Open("spark-worker.yaml")
		if err != nil {
			return err
		}
		SparkTemplateData_Workers, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
	}
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := SparkTemplateData_Workers
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("test1234"), []byte(sparkSecret), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("num-workers-place-holder"), []byte(strconv.Itoa(numWorkers)), -1)
	
	//println("========= HA yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandlder.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.workerrc)
	
	return decoder.Err
}

var SparkTemplateData_Zeppelin []byte = nil

func loadSparkResources_Zeppelin(instanceID, sparkSecret string, res *sparkResources_Zeppelin) error {
	if SparkTemplateData_Zeppelin == nil {
		f, err := os.Open("spark-zeppelin.yaml")
		if err != nil {
			return err
		}
		SparkTemplateData_Zeppelin, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		endpoint_postfix := oshandlder.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			SparkTemplateData_Zeppelin = bytes.Replace(
				SparkTemplateData_Zeppelin, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
	}
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := SparkTemplateData_Zeppelin
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("test1234"), []byte(sparkSecret), -1)	
	
	//println("========= HA yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandlder.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.rc).
		Decode(&res.svc).
		Decode(&res.route)
	
	return decoder.Err
}

type sparkResources_Master struct {
	masterrc  kapi.ReplicationController
	mastersvc kapi.Service
	webroute  routeapi.Route
	websvc    kapi.Service
}

type sparkResources_Workers struct {
	workerrc  kapi.ReplicationController
}

type sparkResources_Zeppelin struct {
	rc     kapi.ReplicationController
	svc    kapi.Service
	route  routeapi.Route
}
	
func createSparkResources_Master (instanceId, serviceBrokerNamespace, sparkSecret string) (*sparkResources_Master, error) {
	var input sparkResources_Master
	err := loadSparkResources_Master(instanceId, sparkSecret, &input)
	if err != nil {
		return nil, err
	}
	
	var output sparkResources_Master
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/replicationcontrollers", &input.masterrc, &output.masterrc).
		KPost(prefix + "/services", &input.mastersvc, &output.mastersvc).
		OPost(prefix + "/routes", &input.webroute, &output.webroute).
		KPost(prefix + "/services", &input.websvc, &output.websvc)
	
	if osr.Err != nil {
		logger.Error("createSparkResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}
	
func getSparkResources_Master (instanceId, serviceBrokerNamespace, sparkSecret string) (*sparkResources_Master, error) {
	var output sparkResources_Master
	
	var input sparkResources_Master
	err := loadSparkResources_Master(instanceId, sparkSecret, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/replicationcontrollers/" + input.masterrc.Name, &output.masterrc).
		KGet(prefix + "/services/" + input.mastersvc.Name, &output.mastersvc).
		OGet(prefix + "/routes/" + input.webroute.Name, &output.webroute).
		KGet(prefix + "/services/" + input.websvc.Name, &output.websvc)
	
	if osr.Err != nil {
		logger.Error("getSparkResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}

func destroySparkResources_Master (masterRes *sparkResources_Master, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail

	go func() {kdel (serviceBrokerNamespace, "replicationcontrollers", masterRes.masterrc.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.mastersvc.Name)}()
	go func() {odel (serviceBrokerNamespace, "routes", masterRes.webroute.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.websvc.Name)}()
}
	
func (job *sparkOrchestrationJob) createSparkResources_Workers (instanceId, serviceBrokerNamespace, sparkSecret string) error {
	var input sparkResources_Workers
	err := loadSparkResources_Workers(instanceId, sparkSecret, job.planNumWorkers, &input)
	if err != nil {
		return err
	}
	
	var output sparkResources_Workers
	go func() {
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.workerrc, &output.workerrc); err != nil {
			return
		}
	}()
	
	return nil
}
	
func getSparkResources_Workers (instanceId, serviceBrokerNamespace, sparkSecret string, numWorkers int) (*sparkResources_Workers, error) {
	var output sparkResources_Workers
	
	var input sparkResources_Workers
	err := loadSparkResources_Workers(instanceId, sparkSecret, numWorkers, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/replicationcontrollers/" + input.workerrc.Name, &output.workerrc)
	
	if osr.Err != nil {
		logger.Error("getSparkResources_Workers", osr.Err)
	}
	
	return &output, osr.Err
}

func destroySparkResources_Workers (haRes *sparkResources_Workers, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail
	
	go func() {kdel (serviceBrokerNamespace, "replicationcontrollers", haRes.workerrc.Name)}()
}
	
func (job *sparkOrchestrationJob) createSparkResources_Zeppelin (instanceId, serviceBrokerNamespace, sparkSecret string) error {
	var input sparkResources_Zeppelin
	err := loadSparkResources_Zeppelin(instanceId, sparkSecret, &input)
	if err != nil {
		return err
	}
	
	var output sparkResources_Zeppelin
	go func() {
		if err := job.kpost (serviceBrokerNamespace, "replicationcontrollers", &input.rc, &output.rc); err != nil {
			return
		}
		if err := job.kpost (serviceBrokerNamespace, "services", &input.svc, &output.svc); err != nil {
			return
		}
		if err := job.opost (serviceBrokerNamespace, "routes", &input.route, &output.route); err != nil {
			return
		}
	}()
	
	return nil
}

func getSparkResources_Zeppelin (instanceId, serviceBrokerNamespace, sparkSecret string) (*sparkResources_Zeppelin, error) {
	var output sparkResources_Zeppelin
	
	var input sparkResources_Zeppelin
	err := loadSparkResources_Zeppelin(instanceId, sparkSecret, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/replicationcontrollers/" + input.rc.Name, &output.rc).
		KGet(prefix + "/services/" + input.svc.Name, &output.svc).
		OGet(prefix + "/routes/" + input.route.Name, &output.route)
	
	if osr.Err != nil {
		logger.Error("getSparkResources_Zeppelin", osr.Err)
	}
	
	return &output, osr.Err
}

func destroySparkResources_Zeppelin (zeppelinRes *sparkResources_Zeppelin, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail

	go func() {kdel (serviceBrokerNamespace, "replicationcontrollers", zeppelinRes.rc.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", zeppelinRes.svc.Name)}()
	go func() {odel (serviceBrokerNamespace, "routes", zeppelinRes.route.Name)}()
}

//===============================================================
// 
//===============================================================

func (job *sparkOrchestrationJob) kpost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	if job.cancelled {
		return nil
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).KPost(uri, body, into)
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

func (job *sparkOrchestrationJob) opost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	if job.cancelled {
		return nil
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).OPost(uri, body, into)
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
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).KDelete(uri, nil)
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
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).ODelete(uri, nil)
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
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).KPut(uri, rc, nil)
	if osr.Err != nil {
		logger.Error("modify HA rc", osr.Err)
		return
	}
	
	// start watching rc status
	
	statuses, cancel, err := oshandlder.OC().KWatch (uri)
	if err != nil {
		logger.Error("start watching HA rc", err)
		return
	}
	
	go func() {
		for {
			status, _ := <- statuses
			
			if status.Err != nil {
				logger.Error("watch HA spark rc error", status.Err)
				close(cancel)
				return
			} else {
				//logger.Debug("watch spark HA rc, status.Info: " + string(status.Info))
			}
			
			var wrcs watchReplicationControllerStatus
			if err := json.Unmarshal(status.Info, &wrcs); err != nil {
				logger.Error("parse master HA rc status", err)
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
	// Pod details
	Object kapi.ReplicationController `json:"object"`
}
