package kettle


import (
	"fmt"
	"errors"
	//marathon "github.com/gambol99/go-marathon"
	//kapi "golang.org/x/build/kubernetes/api"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	//"net/http"
	//"net"
	"github.com/pivotal-cf/brokerapi"
	//"time"
	"strconv"
	"strings"
	"bytes"
	"encoding/json"
	//"crypto/sha1"
	//"encoding/base64"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	//"sync"
	
	"github.com/pivotal-golang/lager"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
	//deployapi "github.com/openshift/origin/deploy/api/v1"
	
	oshandler "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
)

//==============================================================
// 
//==============================================================

const KettleServcieBrokerName_Standalone = "Kettle_standalone"

func init() {
	oshandler.Register(KettleServcieBrokerName_Standalone, &Kettle_freeHandler{})
	
	logger = lager.NewLogger(KettleServcieBrokerName_Standalone)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

type Kettle_freeHandler struct{}

func (handler *Kettle_freeHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
	return newKettleHandler().DoProvision(instanceID, details, asyncAllowed)
}

func (handler *Kettle_freeHandler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
	return newKettleHandler().DoLastOperation(myServiceInfo)
}

func (handler *Kettle_freeHandler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	return newKettleHandler().DoDeprovision(myServiceInfo, asyncAllowed)
}

func (handler *Kettle_freeHandler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
	return newKettleHandler().DoBind(myServiceInfo, bindingID, details)
}

func (handler *Kettle_freeHandler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
	return newKettleHandler().DoUnbind(myServiceInfo, mycredentials)
}

//==============================================================
// 
//==============================================================

type Kettle_Handler struct{
}

func newKettleHandler() *Kettle_Handler {
	return &Kettle_Handler{}
}

func (handler *Kettle_Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
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
	kettleUser := oshandler.NewElevenLengthID()
	kettlePassword := oshandler.GenGUID()
	
	println()
	println("instanceIdInTempalte = ", instanceIdInTempalte)
	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
	println()
	
	// master kettle
	
	output, err := createKettleResources_Master(instanceIdInTempalte, serviceBrokerNamespace, kettleUser, kettlePassword)

	if err != nil {
		destroyKettleResources_Master(output, serviceBrokerNamespace)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.User = kettleUser
	serviceInfo.Password = kettlePassword
	
	serviceSpec.DashboardURL = fmt.Sprintf("http://%s:%s@%s", kettleUser, kettlePassword, output.route.Spec.Host)
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Kettle_Handler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
	
	// assume in provisioning
	
	// the job may be finished or interrupted or running in another instance.
	
	master_res, _ := getKettleResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	
	//ok := func(rc *kapi.ReplicationController) bool {
	//	if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
	//		return false
	//	}
	//	return true
	//}
	ok := func(rc *kapi.ReplicationController) bool {
	println("rc =", rc)
	println("rc.Name =", rc.Name)
	println("rc.Spec.Replicas =", rc.Spec.Replicas)
	println("rc.Status.Replicas =", rc.Status.Replicas)
		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
			return false
		}
		n, _ := statRunningPodsByLabels (myServiceInfo.Database, rc.Labels)
	println("n =", n)
		return n >= *rc.Spec.Replicas
	}
	
	//println("num_ok_rcs = ", num_ok_rcs)
	/*
	rcs, err := oshandler.GetReplicationControllersByLabels(myServiceInfo.Database, master_res.dc.Spec.Selector)
	if err != nil {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: "Error: " + err.Error(),
		}, nil
	}
	if len(rcs) != 1 {
		return brokerapi.LastOperation{
			State:       brokerapi.Failed,
			Description: fmt.Sprintf("Error: number of rcs (%d) is not 1.", len(rcs)),
		}, nil
	}
	rc := &rcs[0]
	*/
	
	rc := &master_res.rc
	
	if ok (rc) {
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

func (handler *Kettle_Handler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	// ...
	
	println("to destroy resources")
	
	master_res, _ := getKettleResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	destroyKettleResources_Master (master_res, myServiceInfo.Database)
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Kettle_Handler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
	// todo: handle errors
	
	master_res, err := getKettleResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	if err != nil {
		return brokerapi.Binding{}, oshandler.Credentials{}, err
	}
	
	web_port := oshandler.GetServicePortByName(&master_res.service, "web")
	if web_port == nil {
		return brokerapi.Binding{}, oshandler.Credentials{}, errors.New("web port not found")
	}
	
	host := fmt.Sprintf("%s.%s.svc.cluster.local", master_res.service.Name, myServiceInfo.Database)
	port := strconv.Itoa(web_port.Port)
	//host := master_res.routeMQ.Spec.Host
	//port := "80"
	
	mycredentials := oshandler.Credentials{
		Uri:      "",
		Hostname: host,
		Port:     port,
		Username: myServiceInfo.User,
		Password: myServiceInfo.Password,
	}

	myBinding := brokerapi.Binding{Credentials: mycredentials}

	return myBinding, mycredentials, nil
}

func (handler *Kettle_Handler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
	// do nothing
	
	return nil
}

//=======================================================================
// 
//=======================================================================

var KettleTemplateData_Master []byte = nil

func loadKettleResources_Master(instanceID, kettleUser, kettlePassword string, res *kettleResources_Master) error {
	if KettleTemplateData_Master == nil {
		f, err := os.Open("kettle.yaml")
		if err != nil {
			return err
		}
		KettleTemplateData_Master, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		endpoint_postfix := oshandler.EndPointSuffix()
		endpoint_postfix = strings.TrimSpace(endpoint_postfix)
		if len(endpoint_postfix) > 0 {
			KettleTemplateData_Master = bytes.Replace(
				KettleTemplateData_Master, 
				[]byte("endpoint-postfix-place-holder"), 
				[]byte(endpoint_postfix), 
				-1)
		}
		kettle_image := oshandler.KettleImage()
		kettle_image = strings.TrimSpace(kettle_image)
		if len(kettle_image) > 0 {
			KettleTemplateData_Master = bytes.Replace(
				KettleTemplateData_Master, 
				[]byte("http://kettle-image-place-holder/kettle-openshift-orchestration"), 
				[]byte(kettle_image), 
				-1)
		}
	}
	
	// ...
	
	yamlTemplates := KettleTemplateData_Master
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("user*****"), []byte(kettleUser), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("pass*****"), []byte(kettlePassword), -1)
	
	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.rc).
		//Decode(&res.dc).
		Decode(&res.route).
		Decode(&res.service)
	
	return decoder.Err
}

type kettleResources_Master struct {
	rc      kapi.ReplicationController
	//dc      deployapi.DeploymentConfig
	route   routeapi.Route
	service kapi.Service
}
	
func createKettleResources_Master (instanceId, serviceBrokerNamespace, kettleUser, kettlePassword string) (*kettleResources_Master, error) {
	var input kettleResources_Master
	err := loadKettleResources_Master(instanceId, kettleUser, kettlePassword, &input)
	if err != nil {
		return nil, err
	}
	
	var output kettleResources_Master
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/replicationcontrollers", &input.rc, &output.rc).
		//OPost(prefix + "/deploymentconfigs", &input.dc, &output.dc).
		OPost(prefix + "/routes", &input.route, &output.route).
		KPost(prefix + "/services", &input.service, &output.service)
	
	if osr.Err != nil {
		logger.Error("createKettleResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}
	
func getKettleResources_Master (instanceId, serviceBrokerNamespace, kettleUser, kettlePassword string) (*kettleResources_Master, error) {
	var output kettleResources_Master
	
	var input kettleResources_Master
	err := loadKettleResources_Master(instanceId, kettleUser, kettlePassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/replicationcontrollers/" + input.rc.Name, &output.rc).
		//OGet(prefix + "/deploymentconfigs/" + input.dc.Name, &output.dc).
		OGet(prefix + "/routes/" + input.route.Name, &output.route).
		KGet(prefix + "/services/" + input.service.Name, &output.service)
	
	if osr.Err != nil {
		logger.Error("getKettleResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyKettleResources_Master (masterRes *kettleResources_Master, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail

	go func() {kdel_rc (serviceBrokerNamespace, &masterRes.rc)}()
	//go func() {odel (serviceBrokerNamespace, "deploymentconfigs", masterRes.dc.Name)}()
	go func() {odel (serviceBrokerNamespace, "routes", masterRes.route.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.service.Name)}()
}

//===============================================================
// 
//===============================================================

func kpost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	
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

func opost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	
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
				logger.Error("watch HA kettle rc error", status.Err)
				close(cancel)
				return
			} else {
				//logger.Debug("watch kettle HA rc, status.Info: " + string(status.Info))
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
	// RC details
	Object kapi.ReplicationController `json:"object"`
}

func statRunningPodsByLabels(serviceBrokerNamespace string, labels map[string]string) (int, error) {
	
	println("to list pods in", serviceBrokerNamespace)
	
	uri := "/namespaces/" + serviceBrokerNamespace + "/pods"
	
	pods := kapi.PodList{}
	
	osr := oshandler.NewOpenshiftREST(oshandler.OC()).KList(uri, labels, &pods)
	if osr.Err != nil {
		logger.Error("statRunningPodsByLabels", osr.Err)
		return 0, osr.Err
	}
	
	nrunnings := 0
	
	for i := range pods.Items {
		pod := &pods.Items[i]
		
		println("\n pods.Items[", i, "].Name =", pod.Name, ", Status.Phase =", pod.Status.Phase, "\n")
		
		if pod.Status.Phase == kapi.PodRunning {
			nrunnings ++
		}
	}
	
	return nrunnings, nil
}

// todo: 

/*
bin/zkCli.sh 127.0.0.1:2181
bin/zkCli.sh -server sb-instanceid-zk:2181

echo conf|nc localhost 2181
echo cons|nc localhost 2181
echo ruok|nc localhost 2181
echo srst|nc localhost 2181
echo crst|nc localhost 2181
echo dump|nc localhost 2181
echo srvr|nc localhost 2181
echo stat|nc localhost 2181
echo mntr|nc localhost 2181
*/