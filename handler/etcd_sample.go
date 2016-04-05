package handler

import (
	//"fmt"
	"errors"
	//marathon "github.com/gambol99/go-marathon"
	//kapi "golang.org/x/build/kubernetes/api"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	//"net/http"
	"github.com/pivotal-cf/brokerapi"
	//"time"
	//"strings"
	"bytes"
	//"text/template"
	//"io"
	"io/ioutil"
	"os"
	
	//"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
)

func init() {
	register("etcd_sample_Sample", &Etcd_sampleHandler{})

}

func (oc *OpenshiftClient) t() {
	new(Etcd_sampleHandler).DoProvision(
		"test1", 
		brokerapi.ProvisionDetails{
			OrganizationGUID: "ttt",
		},
		true)
}

type Etcd_sampleHandler struct{}

func (handler *Etcd_sampleHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, ServiceInfo, error) {
	//初始化到openshift的链接
	
	
	
	var input, output etcdTemplateResources
	err := loadEtcdTemplateResources(instanceID, &input)
	if err != nil {
		return brokerapi.ProvisionedServiceSpec{}, ServiceInfo{}, err
	}
	
	osr := NewOpenshiftREST(theOC)
	
	prefix := "/namespaces/" + details.OrganizationGUID
	osr.
		KPost(prefix + "/services", &input.service, &output.service).
		OPost(prefix + "/routes", &input.route, &output.route).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc0, &output.etcdrc0).
		KPost(prefix + "/services", &input.etcdsrv0, &output.etcdsrv0).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc1, &output.etcdrc1).
		KPost(prefix + "/services", &input.etcdsrv1, &output.etcdsrv1).
		KPost(prefix + "/replicationcontrollers", &input.etcdrc2, &output.etcdrc2).
		KPost(prefix + "/services", &input.etcdsrv2, &output.etcdsrv2)
	
	if osr.err != nil {
		println("osr.err = ", osr.err.Error())
		return brokerapi.ProvisionedServiceSpec{}, ServiceInfo{}, osr.err
	}
	
println()
println("output.route.Spec.Host = ", output.route.Spec.Host)
println("output.route.Spec.Port = ", output.route.Spec.Port.TargetPort.IntVal)
println("output.route.Spec.Path = ", output.route.Spec.Path)
println()
	
	// ...
	
	applicationName := "app"
	DashboardURL := "http://"

	//赋值隐藏属性
	myServiceInfo := ServiceInfo{
		Url:      DashboardURL,
		Database: applicationName, //用来存储marathon的实例名字
	}

	provsiondetail := brokerapi.ProvisionedServiceSpec{
		DashboardURL: DashboardURL,
		IsAsync:      false, // true
	}

	return provsiondetail, myServiceInfo, nil

}

func (handler *Etcd_sampleHandler) DoLastOperation(myServiceInfo *ServiceInfo) (brokerapi.LastOperation, error) {
	//因为是同步模式，协议里面并没有说怎么处理啊，统一反馈成功吧！
	return brokerapi.LastOperation{
		State:       brokerapi.Succeeded,
		Description: "It's a sync method!",
	}, nil
}

func (handler *Etcd_sampleHandler) DoDeprovision(myServiceInfo *ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	//初始化到kubernetes的链接
	return brokerapi.IsAsync(false), errors.New("not implemented")
}

func (handler *Etcd_sampleHandler) DoBind(myServiceInfo *ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, Credentials, error) {
	//初始化到marathon的链接
	return brokerapi.Binding{}, Credentials{}, errors.New("not implemented")
}

func (handler *Etcd_sampleHandler) DoUnbind(myServiceInfo *ServiceInfo, mycredentials *Credentials) error {
	//没啥好做的

	return errors.New("not implemented")

}

//===============================================================
// 
//===============================================================

type etcdTemplateResources struct {
	service  kapi.Service
	route    routeapi.Route
	etcdrc0  kapi.ReplicationController
	etcdsrv0 kapi.Service
	etcdrc1  kapi.ReplicationController
	etcdsrv1 kapi.Service
	etcdrc2  kapi.ReplicationController
	etcdsrv2 kapi.Service
}

var etcdTemplateData []byte = nil

func loadEtcdTemplateResources(instanceID string, res *etcdTemplateResources) error {
	if etcdTemplateData == nil {
		f, err := os.Open("openshift_etcd.yaml")
		if err != nil {
			return err
		}
		etcdTemplateData, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
	}
	
	yamlTemplates := bytes.Replace(etcdTemplateData, []byte("instanceid"), []byte(instanceID), -1)
	
	
println("========= new yamlTemplates ===========")
println(string(yamlTemplates))
println()

	
	decoder := NewYamlDecoder(yamlTemplates)
	decoder.
		Decode(&res.service).
		Decode(&res.route).
		Decode(&res.etcdrc0).
		Decode(&res.etcdsrv0).
		Decode(&res.etcdrc1).
		Decode(&res.etcdsrv1).
		Decode(&res.etcdrc2).
		Decode(&res.etcdsrv2)
	
	return decoder.err
}

/*
func Yaml2Json1(yamlData []byte) error {
	var err error
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(yamlData))
	
	rc := &kapi.ReplicationController{}
	err = decoder.Decode(rc)
	
	pod1 := &kapi.Pod{}
	err = decoder.Decode(pod1)
	
	svc1 := &kapi.Service{}
	err = decoder.Decode(svc1)
	
	pod2 := &kapi.Pod{}
	err = decoder.Decode(pod2)
	
	svc2 := &kapi.Service{}
	err = decoder.Decode(svc2)
	
	pod3 := &kapi.Pod{}
	err = decoder.Decode(pod3)
	
	svc3 := &kapi.Service{}
	err = decoder.Decode(svc3)
	
	service := &kapi.Service{}
	err = decoder.Decode(service)
	
	route := &routeapi.Route{}
	err = decoder.Decode(route)	
	
	return nil, err
}
*/

/*
var etcdTemplate = template.Must(template.ParseFiles("openshift_etcd.yaml"))

func Yaml2Json2(replaces map[string]string) ([]byte, error) {
	var err error
	
	var out bytes.Buffer
	err = etcdTemplate.Execute(&out, replaces)
	
	return nil, err
}
*/

/*
var etcdTemplateData []byte = nil

// maybe the replace order is important, so using slice other than map would be better
func EtcdYaml2Json(replaces map[string]string) ([][]byte, error) {
	if etcdTemplateData == nil {
		f, err := os.Open("openshift_etcd.yaml")
		if err != nil {
			return nil, err
		}
		etcdTemplateData, err = ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
	}
	
	return Yaml2Json(etcdTemplateData, replaces)
}
*/
