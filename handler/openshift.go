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
	"crypto/tls"
	"net/http"
	//"net/url"
	//"encoding/base64"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
	
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	
	"github.com/openshift/origin/pkg/cmd/util/tokencmd"
)

type OpenshiftClient struct {
	host    string
	authUrl string
	oapiUrl string
	kapiUrl string
	
	username    string
	password    string
	bearerToken string
}

func newOpenshiftClient(host, username, password string) *OpenshiftClient {
	host = "https://" + host
	oc := &OpenshiftClient{
		host:    host,
		authUrl: host + "/oauth/authorize?response_type=token&client_id=openshift-challenging-client",
		oapiUrl: host + "/oapi/v1",
		kapiUrl: host + "/api/v1",
		
		username: username,
		password: password,
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
			
			oc.t()
			
			time.Sleep(3 * time.Hour)
		}
	}
}

func (oc *OpenshiftClient) t() {
	status, _, err := oc.Watch("/watch/servicebrokers/sb-marathon")
	if err != nil {
		println("Watch error: ", err.Error())
	}
	
	select {
	case s := <- status:
		if s.Info != nil {
			println("s.Info = ", string(s.Info))
		}
		if s.Err != nil {
			println("s.Err = ", s.Err.Error())
		}
	}
}

func (oc *OpenshiftClient) doRequest (method string, url string, headers map[string]string, body []byte) (*http.Response, error) {
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
	
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req.Header.Add("Authorization", "Bearer " + token)
	
	println("Authorization = ", req.Header.Get("Authorization"))
	
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transCfg,
		Timeout: time.Duration(10) * time.Second,
	}
	return client.Do(req)
}

type WatchStatus struct {
	Info []byte
	Err  error
}

func (oc *OpenshiftClient) Watch (uri string) (<-chan WatchStatus, chan<- struct{}, error) {
	res, err := oc.doRequest("GET", oc.oapiUrl + uri, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	if res.Body == nil {
		return nil, nil, errors.New("response.body is nil")
	}
	
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

func (oc *OpenshiftClient) Get (uri string) (string, error) {
	return "", nil
}

func (oc *OpenshiftClient) Post (uri string, body string) (string, error) {
	return "", nil
}

func (oc *OpenshiftClient) Update (uri string, body string) (string, error) {
	return "", nil
}

func (oc *OpenshiftClient) Delete (uri string) (string, error) {
	return "", nil
}

