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
	Name     string `json:"name"`
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

func Register(name string, handler HandlerDriver) {
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

func GenGUID() string {
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

func OC() *OpenshiftClient {
	return theOC
}

func EndPointSuffix() string {
	return endpointSuffix
}

func EtcdImage() string {
	return etcdImage
}

func EtcdbootImage() string {
	return etcdbootImage
}

func ZookeeperImage() string {
	return zookeeperImage
}

func ZookeeperExhibitorImage() string {
	return zookeeperexhibitorImage
}

func RedisImage() string {
	return redisImage
}

func RedisPhpAdminImage() string {
	return redisphpadminImage
}

func KafkaImage() string {
	return kafkaImage
}

func StormImage() string {
	return stormImage
}

func CassandraImage() string {
	return cassandraImage
}

func TensorFlowImage() string {
	return tensorflowImage
}

func NiFiImage() string {
	return nifiImage
}

func KettleImage() string {
	return kettleImage
}

func SimpleFileUplaoderImage() string {
	return simplefileuplaoderImage
}

func RabbitmqImage() string {
	return rabbitmqImage
}

func SparkImage() string {
	return sparkImage
}

func ZepplinImage() string {
	return zepplinImage
}


var theOC *OpenshiftClient
var endpointSuffix string

var etcdImage string
var etcdbootImage string
var zookeeperImage string
var zookeeperexhibitorImage string
var redisImage string
var redisphpadminImage string
var kafkaImage string
var stormImage string
var cassandraImage string
var tensorflowImage string
var nifiImage string
var kettleImage string
var simplefileuplaoderImage string
var rabbitmqImage string
var sparkImage string
var zepplinImage string

func init() {
	theOC = newOpenshiftClient (
		getenv("OPENSHIFTADDR"), 
		getenv("OPENSHIFTUSER"), 
		getenv("OPENSHIFTPASS"),
		getenv("SBNAMESPACE"),
	)
	
	endpointSuffix = getenv("ENDPOINTSUFFIX")
	etcdImage = getenv("ETCDIMAGE")
	etcdbootImage = getenv("ETCDBOOTIMAGE")
	zookeeperImage = getenv("ZOOKEEPERIMAGE")
	zookeeperexhibitorImage = getenv("ZOOKEEPEREXHIBITORIMAGE")
	redisImage = getenv("REDISIMAGE")
	redisphpadminImage = getenv("REDISPHPADMINIMAGE")
	kafkaImage = getenv("KAFKAIMAGE")
	stormImage = getenv("STORMIMAGE")
	cassandraImage = getenv("CASSANDRAIMAGE")
	tensorflowImage = getenv("TENSORFLOWIMAGE")
	nifiImage = getenv("NIFIIMAGE")
	kettleImage = getenv("KETTLEIMAGE")
	simplefileuplaoderImage = getenv("SIMPLEFILEUPLOADERIMAGE")
	rabbitmqImage = getenv("RABBITMQIMAGE")
	sparkImage = getenv("SPARKIMAGE")
	zepplinImage = getenv("ZEPPLINIMAGE")
}

