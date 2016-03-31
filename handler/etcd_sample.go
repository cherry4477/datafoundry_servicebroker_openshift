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
	
	"k8s.io/kubernetes/pkg/util/yaml"
	kapi "k8s.io/kubernetes/pkg/api/v1"
	routeapi "github.com/openshift/origin/route/api/v1"
)

func init() {
	register("etcd_sample_Sample", &Etcd_sampleHandler{})

}

type Etcd_sampleHandler struct{}

func (handler *Etcd_sampleHandler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, ServiceInfo, error) {
	//初始化到openshift的链接
	
	
	
	applicationName := "app"
	DashboardURL := "http://dashboard"

	//赋值隐藏属性
	myServiceInfo := ServiceInfo{
		Url:      DashboardURL,
		Database: applicationName, //用来存储marathon的实例名字
	}

	provsiondetail := brokerapi.ProvisionedServiceSpec{DashboardURL: DashboardURL, IsAsync: false}

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

func Yaml2Json(yamlData []byte) ([]byte, error) {
	var err error
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(yamlData))
	_ = decoder
	
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
