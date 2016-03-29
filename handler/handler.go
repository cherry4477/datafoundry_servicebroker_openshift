package handler

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/pivotal-cf/brokerapi"
	"io"
	"os"
)

type ServiceInfo struct {
	Service_name   string `json:"service_name"`
	Plan_name      string `json:"plan_name"`
	Url            string `json:"url"`
	Admin_user     string `json:"admin_user,omitempty"`
	Admin_password string `json:"admin_password,omitempty"`
	Database       string `json:"database,omitempty"`
	User           string `json:"user"`
	Password       string `json:"password"`
}

type Credentials struct {
	Uri      string `json:"uri"`
	Hostname string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

type HandlerDriver interface {
	DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, ServiceInfo, error)
	DoLastOperation(myServiceInfo *ServiceInfo) (brokerapi.LastOperation, error)
	DoDeprovision(myServiceInfo *ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error)
	DoBind(myServiceInfo *ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, Credentials, error)
	DoUnbind(myServiceInfo *ServiceInfo, mycredentials *Credentials) error
}

type Handler struct {
	driver HandlerDriver
}

var handlers = make(map[string]HandlerDriver)

func register(name string, handler HandlerDriver) {
	if handler == nil {
		panic("handler: Register handler is nil")
	}
	if _, dup := handlers[name]; dup {
		panic("handler: Register called twice for handler " + name)
	}
	handlers[name] = handler
}

func New(name string) (*Handler, error) {
	handler, ok := handlers[name]
	if !ok {
		return nil, fmt.Errorf("Can't find handler %s", name)
	}
	return &Handler{driver: handler}, nil
}

func (handler *Handler) DoProvision(instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (brokerapi.ProvisionedServiceSpec, ServiceInfo, error) {
	return handler.driver.DoProvision(instanceID, details, asyncAllowed)
}

func (handler *Handler) DoLastOperation(myServiceInfo *ServiceInfo) (brokerapi.LastOperation, error) {
	return handler.driver.DoLastOperation(myServiceInfo)
}

func (handler *Handler) DoDeprovision(myServiceInfo *ServiceInfo, asyncAllowed bool) (brokerapi.IsAsync, error) {
	return handler.driver.DoDeprovision(myServiceInfo, asyncAllowed)
}

func (handler *Handler) DoBind(myServiceInfo *ServiceInfo, bindingID string, details brokerapi.BindDetails) (brokerapi.Binding, Credentials, error) {
	return handler.driver.DoBind(myServiceInfo, bindingID, details)
}

func (handler *Handler) DoUnbind(myServiceInfo *ServiceInfo, mycredentials *Credentials) error {
	return handler.driver.DoUnbind(myServiceInfo, mycredentials)
}

func getmd5string(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getguid() string {
	b := make([]byte, 48)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return getmd5string(base64.URLEncoding.EncodeToString(b))
}

func getenv(env string) string {
	env_value := os.Getenv(env)
	if env_value == "" {
		fmt.Println("FATAL: NEED ENV", env)
		fmt.Println("Exit...........")
		os.Exit(2)
	}
	fmt.Println("ENV:", env, env_value)
	return env_value
}

var oc *OpenshiftClient
func init() {
	oc = newOpenshiftClient(
		getenv("OPENSHIFTADDR"), 
		getenv("OPENSHIFTUSER"), 
		getenv("OPENSHIFTPASS"),
	)
}
