package handler

import (
	//"fmt"
	"errors"
	//marathon "github.com/gambol99/go-marathon"
	//"github.com/pivotal-cf/brokerapi"
	"time"
	"strings"
	"bytes"
	"bufio"
	"net/http"
	"net/url"
	"encoding/base64"
	//"golang.org/x/build/kubernetes"
	//"golang.org/x/oauth2"
)

type OpenshiftClient struct {
	authUrl string
	apiUrl  string
	
	username    string
	password    string
	bearerToken string
}

func newOpenshiftClient(host, username, password string) *OpenshiftClient {
	host = "https://" + host
	oc := &OpenshiftClient{
		authUrl: host + "/oauth/authorize?response_type=token&client_id=openshift-challenging-client",
		apiUrl:  host + "/oapi/v1",
		
		username: username,
		password: password,
	}
	
	go oc.updateBearerToken()
	
	return oc
}

func newRequest (method string, url string, headers map[string]string, body []byte) (*http.Request, error) {
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
	
	return req, nil
}

func (oc *OpenshiftClient) updateBearerToken () {
	client := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	
	sleepForAWhile := func() {
		time.Sleep(15 * time.Second)
	}
	
	for {
		req, err := newRequest("GET", oc.authUrl, nil, nil)
		if err != nil {
			println("111, error: ", err.Error())
			
			sleepForAWhile()
			continue
		}
		req.Header.Set("X-CSRF-Token", "1")
		req.Header.Set("Authorization", "Basic " + base64.StdEncoding.EncodeToString([]byte(oc.username + ":" + oc.password)))
		println("111, Authorization: ", req.Header.Get("Authorization"))
		
		res, err := client.Do(req)
		if err != nil {
			println("222, error: ", err.Error())
			
			sleepForAWhile()
			continue
		}
		
		if res.StatusCode == http.StatusFound {
			u, err := url.Parse(res.Header.Get("Location"))
			if err != nil {
				println("333, error: ", err.Error())
				
				sleepForAWhile()
				continue
			}

			if errorCode := u.Query().Get("error"); len(errorCode) > 0 {
				//errorDescription := u.Query().Get("error_description")
				println("444, error_description: ", u.Query().Get("error_description"))
				
				sleepForAWhile()
				continue
			}

			fragmentValues, err := url.ParseQuery(u.Fragment)
			if err != nil {
				println("555, error: ", err.Error())
				
				sleepForAWhile()
				continue
			}

			if accessToken := fragmentValues.Get("access_token"); len(accessToken) == 0 {
				println("666, len(accessToken) == 0")
				
				sleepForAWhile()
				continue
			} else {
				oc.bearerToken = accessToken
				
				println("new token: ", oc.bearerToken)
			}
		}
		
		time.Sleep(3 * time.Hour)
	}
}

func (oc *OpenshiftClient) doRequest (method string, uri string, headers map[string]string, body []byte) (*http.Response, error) {
	token := oc.bearerToken
	if token == "" {
		return nil, errors.New("token is blank")
	}
	
	req, err := newRequest("GET", oc.host + uri, nil, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Add("Authorization", "Bearer " + token)
	
	client := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	return client.Do(req)
}

type WatchStatus struct {
	Info []byte
	Err  error
}

func (oc *OpenshiftClient) Watch (uri string) (<-chan WatchStatus, chan<- struct{}, error) {
	res, err := oc.doRequest("GET", uri, nil, nil)
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

func (oc *OpenshiftClient) request (method string, headers map[string]string, body string) (*http.Response, error) {
	return nil, nil
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

