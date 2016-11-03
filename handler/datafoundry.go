package handler

import (
	"os"
	"encoding/json"
	"time"
	"io/ioutil"
	"errors"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

var dfProxyApiPrefix string
func DfProxyApiPrefix() string {
	if dfProxyApiPrefix == "" {
		addr := os.Getenv("DATAFOUNDRYPROXYADDR")
		if addr == "" {
			logger.Error("int dfProxyApiPrefix error:", errors.New("DATAFOUNDRYPROXYADDR env is not set"))
		}

		dfProxyApiPrefix = "http://" + addr + "/lapi/v1"
	}
	return dfProxyApiPrefix
}

const DfRequestTimeout = time.Duration(8) * time.Second

func dfRequest(method, url, bearerToken string, bodyParams interface{}, into interface{}) (err error) {
	
	var body []byte
	if bodyParams != nil {
		body, err = json.Marshal(bodyParams)
		if err != nil {
			return
		}
	}
	
	res, err := request(DfRequestTimeout, method, url, bearerToken, body)
	if err != nil {
		return
	}
	defer res.Body.Close()
	
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	
	//println("22222 len(data) = ", len(data), " , res.StatusCode = ", res.StatusCode)
	
	if res.StatusCode < 200 || res.StatusCode >= 400 {
		err = errors.New(string(data))
	} else {
		if into != nil {
			//println("into data = ", string(data), "\n")
		
			err = json.Unmarshal(data, into)
		}
	}
	
	return
}

type VolumnCreateOptions struct {
	Name string     `json:"name,omitempty"`
	Size int        `json:"size,omitempty"`
	kapi.ObjectMeta `json:"metadata,omitempty"`
}

func CreateVolumn(volumnName string, size int) error {
	oc := OC()

	url := DfProxyApiPrefix() + "/namespaces/" + oc.Namespace() + "/volumes"

	options := &VolumnCreateOptions{
		volumnName,
		size,
		kapi.ObjectMeta {
			Annotations: map[string]string {
				"dadafoundry.io/create-by": oc.username,
			},
		},
	}
	
	err := dfRequest("POST", url, oc.BearerToken(), options, nil)

	return err
}

func DeleteVolumn(volumnName string) error {
	oc := OC()

	url := DfProxyApiPrefix() + "/namespaces/" + oc.Namespace() + "/volumes/" + volumnName
	
	err := dfRequest("DELETE", url, oc.BearerToken(), nil, nil)
	
	return err
}

