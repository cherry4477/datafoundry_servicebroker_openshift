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
)

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

func init() {
	register("etcd_sample_Sample", &Etcd_sampleHandler{})

}
