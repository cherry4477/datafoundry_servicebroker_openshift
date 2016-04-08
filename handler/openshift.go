package handler

import (
	//"fmt"
	"errors"
	//marathon "github.com/gambol99/go-marathon"
	//"github.com/pivotal-cf/brokerapi"
	"time"
	//"strings"
	"bytes"
	"bufio"
	//"io"
	"io/ioutil"
	"crypto/tls"
	"net/http"
	//"net/url"
	"encoding/base64"
	"encoding/json"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	
	"github.com/openshift/origin/pkg/cmd/util/tokencmd"
	
	kapi "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/util/yaml"
	//"github.com/ghodss/yaml"
)

type OpenshiftClient struct {
	host    string
	//authUrl string
	oapiUrl string
	kapiUrl string
	
	namespace   string
	username    string
	password    string
	bearerToken string
}

func newOpenshiftClient(host, username, password, defaultNamespace string) *OpenshiftClient {
	host = "https://" + host
	oc := &OpenshiftClient{
		host:    host,
		//authUrl: host + "/oauth/authorize?response_type=token&client_id=openshift-challenging-client",
		oapiUrl: host + "/oapi/v1",
		kapiUrl: host + "/api/v1",
		
		namespace: defaultNamespace,
		username:  username,
		password:  password,
	}
	
	go oc.updateBearerToken()
	
	return oc
}

func (oc *OpenshiftClient) updateBearerToken () {
	for {
		clientConfig := &kclient.Config{}
		clientConfig.Host = oc.host
		clientConfig.Insecure = true
		//clientConfig.Version =
		
		token, err := tokencmd.RequestToken(clientConfig, nil, oc.username, oc.password)
		if err != nil {
			println("RequestToken error: ", err.Error())
			
			time.Sleep(15 * time.Second)
		} else {
			//clientConfig.BearerToken = token
			oc.bearerToken = token
			
			println("RequestToken token: ", token)
			
			time.Sleep(3 * time.Hour)
		}
	}
}

func (oc *OpenshiftClient) request (method string, url string, body []byte, timeout time.Duration) (*http.Response, error) {
	token := oc.bearerToken
	if token == "" {
		return nil, errors.New("token is blank")
	}
	
	var req *http.Request
	var err error
	if len(body) == 0 {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	}
	
	if err != nil {
		return nil, err
	}
	
	//for k, v := range headers {
	//	req.Header.Add(k, v)
	//}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer " + token)
	
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transCfg,
		Timeout: timeout,
	}
	return client.Do(req)
}

type WatchStatus struct {
	Info []byte
	Err  error
}

func (oc *OpenshiftClient) doWatch (url string) (<-chan WatchStatus, chan<- struct{}, error) {
	res, err := oc.request("GET", url, nil, 0)
	if err != nil {
		return nil, nil, err
	}
	//if res.Body == nil {
	//	return nil, nil, errors.New("response.body is nil")
	//}
	
	statuses := make(chan WatchStatus, 5)
	canceled := make(chan struct{}, 1)
	
	go func() {
		defer func () {
			close(statuses)
			res.Body.Close()
		}()
		
		reader := bufio.NewReader(res.Body)
		for {
			select {
			case <- canceled:
				return
			default:
			}
			
			line, err := reader.ReadBytes('\n')
			if err != nil {
				statuses <- WatchStatus{nil, err}
				return
			}
			
			statuses <- WatchStatus{line, nil}
		}
	}()
	
	return statuses, canceled, nil
}

func (oc *OpenshiftClient) OWatch (uri string) (<-chan WatchStatus, chan<- struct{}, error) {
	return oc.doWatch(oc.oapiUrl + "/watch" + uri)
}

func (oc *OpenshiftClient) KWatch (uri string) (<-chan WatchStatus, chan<- struct{}, error) {
	return oc.doWatch(oc.kapiUrl + "/watch" + uri)
}

const GeneralRequestTimeout = time.Duration(10) * time.Second

/*
func (oc *OpenshiftClient) doRequest (method, url string, body []byte) ([]byte, error) {
	res, err := oc.request(method, url, body, GeneralRequestTimeout)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	
	return ioutil.ReadAll(res.Body)
}

func (oc *OpenshiftClient) ORequest (method, uri string, body []byte) ([]byte, error) {
	return oc.doRequest(method, oc.oapiUrl + uri, body)
}

func (oc *OpenshiftClient) KRequest (method, uri string, body []byte) ([]byte, error) {
	return oc.doRequest(method, oc.kapiUrl + uri, body)
}
*/

type OpenshiftREST struct {
	oc  *OpenshiftClient
	err error
}

func NewOpenshiftREST(oc *OpenshiftClient) *OpenshiftREST {
	return &OpenshiftREST{oc: oc}
}

func (osr *OpenshiftREST) doRequest (method, url string, bodyParams interface{}, into interface{}) *OpenshiftREST {
	if osr.err != nil {
		return osr
	}
	
	var body []byte
	if bodyParams != nil {
		body, osr.err = json.Marshal(bodyParams)
		if osr.err != nil {
			return osr
		}
	}
	
	//res, osr.err := oc.request(method, url, body, GeneralRequestTimeout) // non-name error
	res, err := osr.oc.request(method, url, body, GeneralRequestTimeout)
	osr.err = err
	if err != nil {
		return osr
	}
	defer res.Body.Close()
	
	var data []byte
	data, osr.err = ioutil.ReadAll(res.Body)
	if osr.err != nil {
		return osr
	}
	
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		osr.err = errors.New(string(data))
	} else {
		if into != nil {
			//println("into data = ", string(data), "\n")
		
			osr.err = json.Unmarshal(data, into)
		}
	}
	
	return osr
}

// o 

func (osr *OpenshiftREST) OGet (uri string, into interface{}) *OpenshiftREST {
	return osr.doRequest("GET", osr.oc.oapiUrl + uri, nil, into)
}

func (osr *OpenshiftREST) ODelete (uri string, into interface{}) *OpenshiftREST {
	return osr.doRequest("DELETE", osr.oc.oapiUrl + uri, &kapi.DeleteOptions{}, into)
}

func (osr *OpenshiftREST) OPost (uri string, body interface{}, into interface{}) *OpenshiftREST {
	return osr.doRequest("POST", osr.oc.oapiUrl + uri, body, into)
}

func (osr *OpenshiftREST) OPut (uri string, body interface{}, into interface{}) *OpenshiftREST {
	return osr.doRequest("PUT", osr.oc.oapiUrl + uri, body, into)
}

// k 

func (osr *OpenshiftREST) KGet (uri string, into interface{}) *OpenshiftREST {
	return osr.doRequest("GET", osr.oc.kapiUrl + uri, nil, into)
}

func (osr *OpenshiftREST) KDelete (uri string, into interface{}) *OpenshiftREST {
	return osr.doRequest("DELETE", osr.oc.kapiUrl + uri, &kapi.DeleteOptions{}, into)
}

func (osr *OpenshiftREST) KPost (uri string, body interface{}, into interface{}) *OpenshiftREST {
	return osr.doRequest("POST", osr.oc.kapiUrl + uri, body, into)
}

func (osr *OpenshiftREST) KPut (uri string, body interface{}, into interface{}) *OpenshiftREST {
	return osr.doRequest("PUT", osr.oc.kapiUrl + uri, body, into)
}

//===============================================================
// 
//===============================================================

// maybe the replace order is important, so using slice other than map would be better
/*
func Yaml2Json(yamlTemplates []byte, replaces map[string]string) ([][]byte, error) {
	var err error
	
	for old, rep := range replaces {
		etcdTemplateData = bytes.Replace(etcdTemplateData, []byte(old), []byte(rep), -1)
	}
	
	templates := bytes.Split(etcdTemplateData, []byte("---"))
	for i := range templates {
		templates[i] = bytes.TrimSpace(templates[i])
		println("\ntemplates[", i, "] = ", string(templates[i]))
	}
	
	return templates, err
}
*/

/*
func Yaml2Json(yamlTemplates []byte, replaces map[string]string) ([][]byte, error) {
	var err error
	decoder := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(yamlData))
	_ = decoder
	
	
	for {
		var t interface{}
		err = decoder.Decode(&t)
		m, ok := v.(map[string]interface{})
		if ok {
			
		}
	}
}
*/

/*
func Yaml2Json(yamlTemplates []byte, replaces map[string]string) ([][]byte, error) {
	for old, rep := range replaces {
		yamlTemplates = bytes.Replace(yamlTemplates, []byte(old), []byte(rep), -1)
	}
	
	jsons := [][]byte{}
	templates := bytes.Split(yamlTemplates, []byte("---"))
	for i := range templates {
		//templates[i] = bytes.TrimSpace(templates[i])
		println("\ntemplates[", i, "] = ", string(templates[i]))
		
		json, err := yaml.YAMLToJSON(templates[i])
		if err != nil {
			return jsons, err
		}
		
		jsons = append(jsons, json)
		println("\njson[", i, "] = ", string(jsons[i]))
	}
	
	return jsons, nil
}
*/



type YamlDecoder struct {
	decoder *yaml.YAMLToJSONDecoder
	err     error
}

func NewYamlDecoder(yamlData []byte) *YamlDecoder {
	return &YamlDecoder{
			decoder: yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(yamlData)),
		}
}

func (d *YamlDecoder) Error() error {
	return d.err
}

func (d *YamlDecoder) Decode(into interface{}) *YamlDecoder {
	if d.err == nil {
		d.err = d.decoder.Decode(into)
	}
	
	return d
}

func NewElevenLengthID() string {
	t := time.Now().UnixNano()
	bs := make([]byte, 8)
	for i := uint(0); i < 8; i ++ {
		bs[i] = byte((t >> i) & 0xff)
	}
	return string(base64.StdEncoding.EncodeToString(bs))
}



