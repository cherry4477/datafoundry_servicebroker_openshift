package zookeeper


import (
	"fmt"
	//"errors"
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
	"crypto/sha1"
	"encoding/base64"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	//"sync"
	
	"github.com/pivotal-golang/lager"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	//routeapi "github.com/openshift/origin/route/api/v1"
	
	oshandlder "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
)

//==============================================================
// 
//==============================================================

const ZookeeperServcieBrokerName_Standalone = "ZooKeeper_standalone"

func init() {
	oshandlder.Register(ZookeeperServcieBrokerName_Standalone, &Zookeeper_freeHandler{})
	
	logger = lager.NewLogger(ZookeeperServcieBrokerName_Standalone)
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
}

var logger lager.Logger

//==============================================================
// 
//==============================================================

type Zookeeper_freeHandler struct{}

func (handler *Zookeeper_freeHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandlder.ServiceInfo, error) {
	return newZookeeperHandler().DoProvision(instanceID, details, asyncAllowed)
}

func (handler *Zookeeper_freeHandler) DoLastOperation(myServiceInfo *oshandlder.ServiceInfo) (brokerapi.LastOperation, error) {
	return newZookeeperHandler().DoLastOperation(myServiceInfo)
}

func (handler *Zookeeper_freeHandler) DoDeprovision(myServiceInfo *oshandlder.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	return newZookeeperHandler().DoDeprovision(myServiceInfo, asyncAllowed)
}

func (handler *Zookeeper_freeHandler) DoBind(myServiceInfo *oshandlder.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandlder.Credentials, error) {
	return newZookeeperHandler().DoBind(myServiceInfo, bindingID, details)
}

func (handler *Zookeeper_freeHandler) DoUnbind(myServiceInfo *oshandlder.ServiceInfo, mycredentials *oshandlder.Credentials) error {
	return newZookeeperHandler().DoUnbind(myServiceInfo, mycredentials)
}

//==============================================================
// 
//==============================================================

type Zookeeper_Handler struct{
}

func newZookeeperHandler() *Zookeeper_Handler {
	return &Zookeeper_Handler{}
}

func (handler *Zookeeper_Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandlder.ServiceInfo, error) {
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
	zookeeperUser := "super" // oshandlder.NewElevenLengthID()
	zookeeperPassword := oshandlder.GenGUID()
	
	println()
	println("instanceIdInTempalte = ", instanceIdInTempalte)
	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
	println()
	
	// master zookeeper
	
	output, err := createZookeeperResources_Master(instanceIdInTempalte, serviceBrokerNamespace, zookeeperUser, zookeeperPassword)

	if err != nil {
		destroyZookeeperResources_Master(output, serviceBrokerNamespace)
		
		return serviceSpec, serviceInfo, err
	}
	
	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.User = zookeeperUser
	serviceInfo.Password = zookeeperPassword
	
	serviceSpec.DashboardURL = ""
	
	return serviceSpec, serviceInfo, nil
}

func (handler *Zookeeper_Handler) DoLastOperation(myServiceInfo *oshandlder.ServiceInfo) (brokerapi.LastOperation, error) {
	
	// assume in provisioning
	
	// the job may be finished or interrupted or running in another instance.
	
	master_res, _ := getZookeeperResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	
	//ok := func(rc *kapi.ReplicationController) bool {
	//	if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
	//		return false
	//	}
	//	return true
	//}
	ok := func(rc *kapi.ReplicationController) bool {
		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
			return false
		}
		n, _ := statRunningPodsByLabels (myServiceInfo.Database, rc.Labels)
		return n >= *rc.Spec.Replicas
	}
	
	//println("num_ok_rcs = ", num_ok_rcs)
	
	if ok (&master_res.rc1) && ok (&master_res.rc2) && ok (&master_res.rc3) {
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

func (handler *Zookeeper_Handler) DoDeprovision(myServiceInfo *oshandlder.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	
	println("to destroy resources")
	
	master_res, _ := getZookeeperResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	// under current frame, it is not a good idea to return here
	//if err != nil {
	//	return brokerapi.IsAsync(false), err
	//}
	
	destroyZookeeperResources_Master (master_res, myServiceInfo.Database)
	
	return brokerapi.IsAsync(false), nil
}

func (handler *Zookeeper_Handler) DoBind(myServiceInfo *oshandlder.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandlder.Credentials, error) {
	// todo: handle errors
	
	master_res, _ := getZookeeperResources_Master (myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password)
	
	host := fmt.Sprintf("%s.%s.svc.cluster.local", master_res.service.Name, myServiceInfo.Database)
	port := strconv.Itoa(master_res.service.Spec.Ports[0].Port)
	
	mycredentials := oshandlder.Credentials{
		Uri:      "",
		Hostname: host,
		Port:     port,
		Username: myServiceInfo.User,
		Password: myServiceInfo.Password,
	}

	myBinding := brokerapi.Binding{Credentials: mycredentials}

	return myBinding, mycredentials, nil
}

func (handler *Zookeeper_Handler) DoUnbind(myServiceInfo *oshandlder.ServiceInfo, mycredentials *oshandlder.Credentials) error {
	// do nothing
	
	return nil
}

//=======================================================================
// 
//=======================================================================

var ZookeeperTemplateData_Master []byte = nil

func loadZookeeperResources_Master(instanceID, zookeeperUser, zookeeperPassword string, res *zookeeperResources_Master) error {
	if ZookeeperTemplateData_Master == nil {
		f, err := os.Open("zookeeper.yaml")
		if err != nil {
			return err
		}
		ZookeeperTemplateData_Master, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		zookeeper_image := oshandlder.ZookeeperImage()
		zookeeper_image = strings.TrimSpace(zookeeper_image)
		if len(zookeeper_image) > 0 {
			ZookeeperTemplateData_Master = bytes.Replace(
				ZookeeperTemplateData_Master, 
				[]byte("http://etcd-image-place-holder/zookeeper-openshift-orchestration"), 
				[]byte(zookeeper_image), 
				-1)
		}
	}
	
	// ...
	
	// invalid operation sha1.Sum(([]byte)(zookeeperPassword))[:] (slice of unaddressable value)
	//sum := (sha1.Sum([]byte(zookeeperPassword)))[:]
	//zoo_password := zookeeperUser + ":" + base64.StdEncoding.EncodeToString (sum)
	
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%s", zookeeperUser, zookeeperPassword)))
	zoo_password := fmt.Sprintf("%s:%s", zookeeperUser, base64.StdEncoding.EncodeToString (sum[:]))
	
	// todo: max length of res names in kubernetes is 24
	
	yamlTemplates := ZookeeperTemplateData_Master
	
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("super:password-place-holder"), []byte(zoo_password), -1)	
	
	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	
	decoder := oshandlder.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.service).
		Decode(&res.svc1).
		Decode(&res.svc2).
		Decode(&res.svc3).
		Decode(&res.rc1).
		Decode(&res.rc2).
		Decode(&res.rc3)
	
	return decoder.Err
}

type zookeeperResources_Master struct {
	service kapi.Service
	
	svc1  kapi.Service
	svc2  kapi.Service
	svc3  kapi.Service
	rc1   kapi.ReplicationController
	rc2   kapi.ReplicationController
	rc3   kapi.ReplicationController
}
	
func createZookeeperResources_Master (instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string) (*zookeeperResources_Master, error) {
	var input zookeeperResources_Master
	err := loadZookeeperResources_Master(instanceId, zookeeperUser, zookeeperPassword, &input)
	if err != nil {
		return nil, err
	}
	
	var output zookeeperResources_Master
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KPost(prefix + "/services", &input.service, &output.service).
		KPost(prefix + "/services", &input.svc1, &output.svc1).
		KPost(prefix + "/services", &input.svc2, &output.svc2).
		KPost(prefix + "/services", &input.svc3, &output.svc3).
		KPost(prefix + "/replicationcontrollers", &input.rc1, &output.rc1).
		KPost(prefix + "/replicationcontrollers", &input.rc2, &output.rc2).
		KPost(prefix + "/replicationcontrollers", &input.rc3, &output.rc3)
	
	if osr.Err != nil {
		logger.Error("createZookeeperResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}
	
func getZookeeperResources_Master (instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string) (*zookeeperResources_Master, error) {
	var output zookeeperResources_Master
	
	var input zookeeperResources_Master
	err := loadZookeeperResources_Master(instanceId, zookeeperUser, zookeeperPassword, &input)
	if err != nil {
		return &output, err
	}
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC())
	
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		KGet(prefix + "/services/" + input.service.Name, &output.service).
		KGet(prefix + "/services/" + input.svc1.Name, &output.svc1).
		KGet(prefix + "/services/" + input.svc2.Name, &output.svc2).
		KGet(prefix + "/services/" + input.svc3.Name, &output.svc3).
		KGet(prefix + "/replicationcontrollers/" + input.rc1.Name, &output.rc1).
		KGet(prefix + "/replicationcontrollers/" + input.rc2.Name, &output.rc2).
		KGet(prefix + "/replicationcontrollers/" + input.rc3.Name, &output.rc3)
	
	if osr.Err != nil {
		logger.Error("getZookeeperResources_Master", osr.Err)
	}
	
	return &output, osr.Err
}

func destroyZookeeperResources_Master (masterRes *zookeeperResources_Master, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail

	go func() {kdel (serviceBrokerNamespace, "services", masterRes.service.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.svc1.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.svc2.Name)}()
	go func() {kdel (serviceBrokerNamespace, "services", masterRes.svc3.Name)}()
	go func() {kdel_rc (serviceBrokerNamespace, &masterRes.rc1)}()
	go func() {kdel_rc (serviceBrokerNamespace, &masterRes.rc2)}()
	go func() {kdel_rc (serviceBrokerNamespace, &masterRes.rc3)}()
}

//===============================================================
// 
//===============================================================

func kpost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	
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

func opost (serviceBrokerNamespace, typeName string, body interface{}, into interface{}) error {
	println("to create ", typeName)
	
	uri := fmt.Sprintf("/namespaces/%s/%s", serviceBrokerNamespace, typeName)
	i, n := 0, 5
RETRY:
	
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
				logger.Error("watch HA zookeeper rc error", status.Err)
				close(cancel)
				return
			} else {
				//logger.Debug("watch zookeeper HA rc, status.Info: " + string(status.Info))
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
	
	osr := oshandlder.NewOpenshiftREST(oshandlder.OC()).KList(uri, labels, &pods)
	if osr.Err != nil {
		return 0, osr.Err
	}
	
	nrunnings := 0
	
	for i := range pods.Items {
		pod := &pods.Items[i]
		
		println("\n pods.Items[", i, "].Status.Phase =", pod.Status.Phase, "\n")
		
		if pod.Status.Phase == kapi.PodRunning {
			nrunnings ++
		}
	}
	
	return nrunnings, nil
}


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