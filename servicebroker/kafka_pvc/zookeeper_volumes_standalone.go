package kafka_pvc

import (
	//marathon "github.com/gambol99/go-marathon"
	//kapi "golang.org/x/build/kubernetes/api"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	//"net/http"
	//"net"
	"bytes"
	//"crypto/sha1"
	//"encoding/base64"
	//"encoding/json"
	//"github.com/pivotal-cf/brokerapi"
	//"strconv"
	"strings"
	"time"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	//"sync"


	//"k8s.io/kubernetes/pkg/util/yaml"
	//routeapi "github.com/openshift/origin/route/api/v1"
	kapi "k8s.io/kubernetes/pkg/api/v1"

	oshandler "github.com/asiainfoLDP/datafoundry_servicebroker_openshift/handler"
	dcapi "github.com/openshift/origin/deploy/api/v1"
	"fmt"
)

//==============================================================
//
//==============================================================

//const ZookeeperServcieBrokerName_Standalone = "ZooKeeper_volumes_standalone"
//
//func init() {
//	oshandler.Register(ZookeeperServcieBrokerName_Standalone, &Zookeeper_freeHandler{})
//
//	logger = lager.NewLogger(ZookeeperServcieBrokerName_Standalone)
//	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
//}

//var logger lager.Logger

//==============================================================
//
//==============================================================

//type Zookeeper_freeHandler struct{}
//
//func (handler *Zookeeper_freeHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, planInfo oshandler.PlanInfo, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
//	return newZookeeperHandler().DoProvision(instanceID, details, planInfo, asyncAllowed)
//}
//
//func (handler *Zookeeper_freeHandler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
//	return newZookeeperHandler().DoLastOperation(myServiceInfo)
//}
//
//func (handler *Zookeeper_freeHandler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
//	return newZookeeperHandler().DoDeprovision(myServiceInfo, asyncAllowed)
//}
//
//func (handler *Zookeeper_freeHandler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
//	return newZookeeperHandler().DoBind(myServiceInfo, bindingID, details)
//}
//
//func (handler *Zookeeper_freeHandler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
//	return newZookeeperHandler().DoUnbind(myServiceInfo, mycredentials)
//}

//==============================================================
//
//==============================================================

//func volumeBaseName(instanceId string) string {
//	return "zkp-" + instanceId
//}

func peerPvcName0(volumes []oshandler.Volume) string {
	if len(volumes) > 0 {
		return volumes[0].Volume_name
	}
	return ""
}

func peerPvcName1(volumes []oshandler.Volume) string {
	if len(volumes) > 1 {
		return volumes[1].Volume_name
	}
	return ""
}

func peerPvcName2(volumes []oshandler.Volume) string {
	if len(volumes) > 2 {
		return volumes[2].Volume_name
	}
	return ""
}

func peerPvcName3(volumes []oshandler.Volume) string {
	if len(volumes) > 0 {
		return volumes[0].Volume_name
	}
	return ""
}

func peerPvcName4(volumes []oshandler.Volume) string {
	if len(volumes) > 2 {
		return volumes[2].Volume_name
	}
	return ""
}

//==============================================================
//
//==============================================================

type Zookeeper_Handler struct {
}

func newZookeeperHandler() *Zookeeper_Handler {
	return &Zookeeper_Handler{}
}

//func (handler *Zookeeper_Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, planInfo oshandler.PlanInfo, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
//	//初始化到openshift的链接
//
//	serviceSpec := brokerapi.ProvisionedServiceSpec{IsAsync: asyncAllowed}
//	serviceInfo := oshandler.ServiceInfo{}
//
//	//if asyncAllowed == false {
//	//	return serviceSpec, serviceInfo, errors.New("Sync mode is not supported")
//	//}
//	serviceSpec.IsAsync = true
//
//	//instanceIdInTempalte   := instanceID // todo: ok?
//	instanceIdInTempalte := strings.ToLower(oshandler.NewThirteenLengthID())
//	//serviceBrokerNamespace := ServiceBrokerNamespace
//	serviceBrokerNamespace := oshandler.OC().Namespace()
//	zookeeperUser := "super" // oshandler.NewElevenLengthID()
//	zookeeperPassword := oshandler.GenGUID()
//
//	volumeBaseName := volumeBaseName(instanceIdInTempalte)
//	volumes := []oshandler.Volume{
//		// one master volume
//		{
//			Volume_size: planInfo.Volume_size,
//			Volume_name: volumeBaseName + "-0",
//		},
//		// two slave volumes
//		{
//			Volume_size: planInfo.Volume_size,
//			Volume_name: volumeBaseName + "-1",
//		},
//		{
//			Volume_size: planInfo.Volume_size,
//			Volume_name: volumeBaseName + "-2",
//		},
//	}
//
//	println()
//	println("instanceIdInTempalte = ", instanceIdInTempalte)
//	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
//	println()
//
//	serviceInfo.Url = instanceIdInTempalte
//	serviceInfo.Database = serviceBrokerNamespace // may be not needed
//	serviceInfo.User = zookeeperUser
//	serviceInfo.Password = zookeeperPassword
//
//	serviceInfo.Volumes = volumes
//
//	// ...
//	go func() {
//		// create volume
//
//		result := oshandler.StartCreatePvcVolumnJob(
//			volumeBaseName,
//			serviceInfo.Database,
//			serviceInfo.Volumes,
//		)
//
//		err := <-result
//		if err != nil {
//			logger.Error("zookeeper create volume", err)
//			return
//		}
//
//		println("createRedisResources_Master ...")
//
//		// todo: consider if DoDeprovision is called now, ...
//
//		// create master res
//
//		output, err := createZookeeperResources_Master(
//			serviceInfo.Url,
//			serviceInfo.Database,
//			serviceInfo.User,
//			serviceInfo.Password,
//			volumes,
//		)
//
//		if err != nil {
//			println(" redis createRedisResources_Master error: ", err)
//			logger.Error("redis createRedisResources_Master error", err)
//
//			DestroyZookeeperResources_Master(output, serviceBrokerNamespace)
//			oshandler.DeleteVolumns(serviceInfo.Database, volumes)
//
//			return
//		}
//	}()
//
//	serviceSpec.DashboardURL = "" // "http://" + net.JoinHostPort(output.route.Spec.Host, "80")
//
//	return serviceSpec, serviceInfo, nil
//}

/*
func (handler *Zookeeper_Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, planInfo oshandler.PlanInfo, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, oshandler.ServiceInfo, error) {
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
	zookeeperUser := "super" // oshandler.NewElevenLengthID()
	zookeeperPassword := oshandler.GenGUID()

	println()
	println("instanceIdInTempalte = ", instanceIdInTempalte)
	println("serviceBrokerNamespace = ", serviceBrokerNamespace)
	println()

	// master zookeeper

	output, err := CreateZookeeperResources_Master(instanceIdInTempalte, serviceBrokerNamespace, zookeeperUser, zookeeperPassword)

	if err != nil {
		DestroyZookeeperResources_Master(output, serviceBrokerNamespace)

		return serviceSpec, serviceInfo, err
	}

	serviceInfo.Url = instanceIdInTempalte
	serviceInfo.Database = serviceBrokerNamespace // may be not needed
	serviceInfo.User = zookeeperUser
	serviceInfo.Password = zookeeperPassword

	serviceSpec.DashboardURL = "" // "http://" + net.JoinHostPort(output.route.Spec.Host, "80")

	return serviceSpec, serviceInfo, nil
}
*/

//func (handler *Zookeeper_Handler) DoLastOperation(myServiceInfo *oshandler.ServiceInfo) (brokerapi.LastOperation, error) {
//
//	// assume in provisioning
//
//	volumeJob := oshandler.GetCreatePvcVolumnJob(volumeBaseName(myServiceInfo.Url))
//	if volumeJob != nil {
//		return brokerapi.LastOperation{
//			State:       brokerapi.InProgress,
//			Description: "in progress.",
//		}, nil
//	}
//
//	// ...
//
//	master_res, _ := GetZookeeperResources_Master(myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password, myServiceInfo.Volumes)
//
//	//ok := func(rc *kapi.ReplicationController) bool {
//	//	if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
//	//		return false
//	//	}
//	//	return true
//	//}
//	ok := func(rc *kapi.ReplicationController) bool {
//		if rc == nil || rc.Name == "" || rc.Spec.Replicas == nil || rc.Status.Replicas < *rc.Spec.Replicas {
//			return false
//		}
//		n, _ := statRunningPodsByLabels(myServiceInfo.Database, rc.Labels)
//		return n >= *rc.Spec.Replicas
//	}
//
//	//println("num_ok_rcs = ", num_ok_rcs)
//
//	if ok(&master_res.rc1) && ok(&master_res.rc2) && ok(&master_res.rc3) {
//		return brokerapi.LastOperation{
//			State:       brokerapi.Succeeded,
//			Description: "Succeeded!",
//		}, nil
//	} else {
//		return brokerapi.LastOperation{
//			State:       brokerapi.InProgress,
//			Description: "In progress.",
//		}, nil
//	}
//}

//func (handler *Zookeeper_Handler) DoDeprovision(myServiceInfo *oshandler.ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
//	// ...
//
//	go func() {
//		// ...
//		volumeJob := oshandler.GetCreatePvcVolumnJob(volumeBaseName(myServiceInfo.Url))
//		if volumeJob != nil {
//			volumeJob.Cancel()
//
//			// wait job to exit
//			for {
//				time.Sleep(7 * time.Second)
//				if nil == oshandler.GetCreatePvcVolumnJob(volumeBaseName(myServiceInfo.Url)) {
//					break
//				}
//			}
//		}
//
//		println("to destroy master resources")
//
//		master_res, _ := GetZookeeperResources_Master(myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password, myServiceInfo.Volumes)
//		// under current frame, it is not a good idea to return here
//		//if err != nil {
//		//	return brokerapi.IsAsync(false), err
//		//}
//		DestroyZookeeperResources_Master(master_res, myServiceInfo.Database)
//
//		println("to destroy volumes:", myServiceInfo.Volumes)
//
//		oshandler.DeleteVolumns(myServiceInfo.Database, myServiceInfo.Volumes)
//	}()
//
//	return brokerapi.IsAsync(false), nil
//}

//func (handler *Zookeeper_Handler) DoBind(myServiceInfo *oshandler.ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, oshandler.Credentials, error) {
//	// todo: handle errors
//
//	master_res, _ := GetZookeeperResources_Master(myServiceInfo.Url, myServiceInfo.Database, myServiceInfo.User, myServiceInfo.Password, myServiceInfo.Volumes)
//
//	host, port, err := master_res.ServiceHostPort(myServiceInfo.Database)
//	if err != nil {
//		return brokerapi.Binding{}, oshandler.Credentials{}, nil
//	}
//
//	mycredentials := oshandler.Credentials{
//		Uri:      "",
//		Hostname: host,
//		Port:     port,
//		Username: myServiceInfo.User,
//		Password: myServiceInfo.Password,
//	}
//
//	myBinding := brokerapi.Binding{Credentials: mycredentials}
//
//	return myBinding, mycredentials, nil
//}
//
//func (handler *Zookeeper_Handler) DoUnbind(myServiceInfo *oshandler.ServiceInfo, mycredentials *oshandler.Credentials) error {
//	// do nothing
//
//	return nil
//}

//==============================================================
// interfaces for other service brokers which depend on zk
//==============================================================

func watchZookeeperOrchestration(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string, volumes []oshandler.Volume) (result <-chan bool, cancel chan<- struct{}, err error) {
	var input ZookeeperResources_Master
	err = loadZookeeperResources_Master(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword, volumes, &input)
	if err != nil {
		return
	}

	fmt.Println("----->input.dc1.Name:", input.dc1.Name)
	fmt.Println("----->input.dc2.Name:", input.dc2.Name)
	fmt.Println("----->input.dc3.Name:", input.dc3.Name)

	/*
		rc1 := &input.rc1
		rc2 := &input.rc2
		rc3 := &input.rc3
		uri1 := "/namespaces/" + serviceBrokerNamespace + "/replicationcontrollers/" + rc1.Name
		uri2 := "/namespaces/" + serviceBrokerNamespace + "/replicationcontrollers/" + rc2.Name
		uri3 := "/namespaces/" + serviceBrokerNamespace + "/replicationcontrollers/" + rc3.Name
		statuses1, cancel1, err := oshandler.OC().KWatch (uri1)
		if err != nil {
			return
		}
		statuses2, cancel2, err := oshandler.OC().KWatch (uri2)
		if err != nil {
			close(cancel1)
			return
		}
		statuses3, cancel3, err := oshandler.OC().KWatch (uri3)
		if err != nil {
			close(cancel1)
			close(cancel2)
			return
		}

		close_all := func() {
			close(cancel1)
			close(cancel2)
			close(cancel3)
		}
	*/

	var output ZookeeperResources_Master
	err = getZookeeperResources_Master(serviceBrokerNamespace, &input, &output)
	if err != nil {
		//close_all()
		return
	}

	dc1 := &output.dc1
	dc2 := &output.dc2
	dc3 := &output.dc3

	theresult := make(chan bool)
	result = theresult
	cancelled := make(chan struct{})
	cancel = cancelled

	go func() {
		ok := func(dc *dcapi.DeploymentConfig) bool {
			fmt.Println("----->dc.Labels:", dc.Labels)
			podCount, err := statRunningPodsByLabels(serviceBrokerNamespace, dc.Labels)
			fmt.Println("podCount:", podCount)
			if err != nil {
				fmt.Println("statRunningPodsByLabels err:", err)
				return false
			}
			if dc == nil || dc.Name == "" || dc.Spec.Replicas == 0 || podCount < dc.Spec.Replicas {
				return false
			}
			n, _ := statRunningPodsByLabels(serviceBrokerNamespace, dc.Labels)
			return n >= dc.Spec.Replicas
		}

		for {
			if ok(dc1) && ok(dc2) && ok(dc3) {
				theresult <- true

				//close_all()
				return
			}

			//var status oshandler.WatchStatus
			var valid bool
			//var rc **kapi.ReplicationController
			select {
			case <-cancelled:
				valid = false
			//case status, valid = <- statuses1:
			//	//rc = &rc1
			//	break
			//case status, valid = <- statuses2:
			//	//rc = &rc2
			//	break
			//case status, valid = <- statuses3:
			//	//rc = &rc3
			//	break
			case <-time.After(15 * time.Second):
				// bug: pod phase change will not trigger rc status change.
				// so need this case
				continue
			}

			/*
				if valid {
					if status.Err != nil {
						valid = false
						logger.Error("watch master rcs error", status.Err)
					} else {
						var wrcs watchReplicationControllerStatus
						if err := json.Unmarshal(status.Info, &wrcs); err != nil {
							valid = false
							logger.Error("parse master rc status", err)
						//} else {
						//	*rc = &wrcs.Object
						}
					}
				}

				println("> WatchZookeeperOrchestration valid:", valid)
			*/

			if !valid {
				theresult <- false

				//close_all()
				return
			}
		}
	}()

	return
}

//=======================================================================
// the zookeeper functions may be called by outer packages
//=======================================================================

var ZookeeperTemplateData_Master []byte = nil

func loadZookeeperResources_Master(instanceID, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string, volumes []oshandler.Volume, res *ZookeeperResources_Master) error {
	/*
		if ZookeeperTemplateData_Master == nil {
			f, err := os.Open("zookeeper.yaml")
			if err != nil {
				return err
			}
			ZookeeperTemplateData_Master, err = ioutil.ReadAll(f)
			if err != nil {
				return err
			}
			zookeeper_image := oshandler.ZookeeperImage()
			zookeeper_image = strings.TrimSpace(zookeeper_image)
			if len(zookeeper_image) > 0 {
				ZookeeperTemplateData_Master = bytes.Replace(
					ZookeeperTemplateData_Master,
					[]byte("http://zookeeper-image-place-holder/zookeeper-openshift-orchestration"),
					[]byte(zookeeper_image),
					-1)
			}
		}
	*/

	if ZookeeperTemplateData_Master == nil {

		f, err := os.Open("zk-before-kafka-pvc.yaml")
		if err != nil {
			return err
		}
		ZookeeperTemplateData_Master, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		zookeeper_image := oshandler.KafkaVolumeImage()
		zookeeper_image = strings.TrimSpace(zookeeper_image)
		if len(zookeeper_image) > 0 {
			ZookeeperTemplateData_Master = bytes.Replace(
				ZookeeperTemplateData_Master,
				[]byte("http://kafka-image-place-holder/kafka-openshift-orchestration"),
				[]byte(zookeeper_image),
				-1)
		}
	}

	// ...

	// ...

	peerPvcName0 := peerPvcName0(volumes)
	peerPvcName1 := peerPvcName1(volumes)
	peerPvcName2 := peerPvcName2(volumes)

	// invalid operation sha1.Sum(([]byte)(zookeeperPassword))[:] (slice of unaddressable value)
	//sum := (sha1.Sum([]byte(zookeeperPassword)))[:]
	//zoo_password := zookeeperUser + ":" + base64.StdEncoding.EncodeToString (sum)

	yamlTemplates := ZookeeperTemplateData_Master

	yamlTemplates = bytes.Replace(yamlTemplates, []byte("instanceid"), []byte(instanceID), -1)

	yamlTemplates = bytes.Replace(yamlTemplates, []byte("zk-pvc-name-replace1"), []byte(peerPvcName0), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("zk-pvc-name-replace2"), []byte(peerPvcName1), -1)
	yamlTemplates = bytes.Replace(yamlTemplates, []byte("zk-pvc-name-replace3"), []byte(peerPvcName2), -1)

	//println("========= Boot yamlTemplates ===========")
	//println(string(yamlTemplates))
	//println()

	decoder := oshandler.NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.dc1).
		Decode(&res.dc2).
		Decode(&res.dc3).
		Decode(&res.svc1).
		Decode(&res.svc2).
		Decode(&res.svc3).
		Decode(&res.svc4)

	return decoder.Err
}

type ZookeeperResources_Master struct {
	dc1 dcapi.DeploymentConfig
	dc2 dcapi.DeploymentConfig
	dc3 dcapi.DeploymentConfig

	svc1 kapi.Service
	svc2 kapi.Service
	svc3 kapi.Service
	svc4 kapi.Service
}

//func (masterRes *ZookeeperResources_Master) ServiceHostPort(serviceBrokerNamespace string) (string, string, error) {
//
//	client_port := oshandler.GetServicePortByName(&masterRes.service, "client")
//	if client_port == nil {
//		return "", "", errors.New("client port not found")
//	}
//
//	host := fmt.Sprintf("%s.%s.svc.cluster.local", masterRes.service.Name, serviceBrokerNamespace)
//	port := strconv.Itoa(client_port.Port)
//
//	return host, port, nil
//}

func createZookeeperResources_Master(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string, volumes []oshandler.Volume) (*ZookeeperResources_Master, error) {
	var input ZookeeperResources_Master
	err := loadZookeeperResources_Master(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword, volumes, &input)
	if err != nil {
		return nil, err
	}

	var output ZookeeperResources_Master

	osr := oshandler.NewOpenshiftREST(oshandler.OC())

	// here, not use job.post
	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		OPost(prefix+"/deploymentconfigs", &input.dc1, &output.dc1).
		OPost(prefix+"/deploymentconfigs", &input.dc2, &output.dc2).
		OPost(prefix+"/deploymentconfigs", &input.dc3, &output.dc3).
		KPost(prefix+"/services", &input.svc1, &output.svc1).
		KPost(prefix+"/services", &input.svc2, &output.svc2).
		KPost(prefix+"/services", &input.svc3, &output.svc3).
		KPost(prefix+"/services", &input.svc4, &output.svc4)

	if osr.Err != nil {
		logger.Error("createZookeeperResources_Master", osr.Err)
	}

	return &output, osr.Err
}

func GetZookeeperResources_Master(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword string, volumes []oshandler.Volume) (*ZookeeperResources_Master, error) {
	var output ZookeeperResources_Master

	var input ZookeeperResources_Master
	err := loadZookeeperResources_Master(instanceId, serviceBrokerNamespace, zookeeperUser, zookeeperPassword, volumes, &input)
	if err != nil {
		return &output, err
	}

	err = getZookeeperResources_Master(serviceBrokerNamespace, &input, &output)
	return &output, err
}

func getZookeeperResources_Master(serviceBrokerNamespace string, input, output *ZookeeperResources_Master) error {
	osr := oshandler.NewOpenshiftREST(oshandler.OC())

	prefix := "/namespaces/" + serviceBrokerNamespace
	osr.
		OGet(prefix+"/deploymentconfigs/"+input.dc1.Name, &output.dc1).
		OGet(prefix+"/deploymentconfigs/"+input.dc2.Name, &output.dc2).
		OGet(prefix+"/deploymentconfigs/"+input.dc3.Name, &output.dc3).
		KGet(prefix+"/services/"+input.svc1.Name, &output.svc3).
		KGet(prefix+"/services/"+input.svc2.Name, &output.svc3).
		KGet(prefix+"/services/"+input.svc3.Name, &output.svc3).
		KGet(prefix+"/services/"+input.svc4.Name, &output.svc3)

	if osr.Err != nil {
		logger.Error("getZookeeperResources_Master", osr.Err)
	}

	return osr.Err
}

func destroyZookeeperResources_Master(masterRes *ZookeeperResources_Master, serviceBrokerNamespace string) {
	// todo: add to retry queue on fail

	go func() { odel(serviceBrokerNamespace, "deploymentconfigs", masterRes.dc1.Name) }()
	go func() { odel(serviceBrokerNamespace, "deploymentconfigs", masterRes.dc2.Name) }()
	go func() { odel(serviceBrokerNamespace, "deploymentconfigs", masterRes.dc3.Name) }()
	go func() { kdel(serviceBrokerNamespace, "services", masterRes.svc1.Name) }()
	go func() { kdel(serviceBrokerNamespace, "services", masterRes.svc2.Name) }()
	go func() { kdel(serviceBrokerNamespace, "services", masterRes.svc3.Name) }()
	go func() { kdel(serviceBrokerNamespace, "services", masterRes.svc4.Name) }()
}

//===============================================================
//
//===============================================================

/*
func kdel_rc (serviceBrokerNamespace string, rc *kapi.ReplicationController) {
	kdel (serviceBrokerNamespace, "replicationcontrollers", rc.Name)
}
*/

// https://hub.docker.com/r/mbabineau/zookeeper-exhibitor/
// https://hub.docker.com/r/netflixoss/exhibitor/

// todo:
// set ACL: https://godoc.org/github.com/samuel/go-zookeeper/zk#Conn.SetACL
// github.com/samuel/go-zookeeper/zk

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

/* need this?

# zoo.cfg

# Enable regular purging of old data and transaction logs every 24 hours
autopurge.purgeInterval=24
autopurge.snapRetainCount=5

The last two autopurge.* settings are very important for production systems.
They instruct ZooKeeper to regularly remove (old) data and transaction logs.
The default ZooKeeper configuration does not do this on its own,
and if you do not set up regular purging ZooKeeper will quickly run out of disk space.

*/
