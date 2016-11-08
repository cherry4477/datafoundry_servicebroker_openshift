package handler

import (
	"os"
	"encoding/json"
	"time"
	"io/ioutil"
	"errors"
	"fmt"
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
println("dfRequest 000")

	var body []byte
	if bodyParams != nil {
		body, err = json.Marshal(bodyParams)
		if err != nil {
			return
		}
	}
	
	res, err := request(DfRequestTimeout, method, url, bearerToken, body)
println("dfRequest 111, err=", err)
	if err != nil {
		return
	}
	defer res.Body.Close()
	
	data, err := ioutil.ReadAll(res.Body)
println("dfRequest 222, err=", err)
	if err != nil {
		return
	}
	
	//println("22222 len(data) = ", len(data), " , res.StatusCode = ", res.StatusCode)
	
println("dfRequest 333, res.StatusCode=", res.StatusCode)

	if res.StatusCode < 200 || res.StatusCode >= 400 {
		err = errors.New(string(data))
println("dfRequest 444, err=", err)
	} else {
		if into != nil {
			//println("into data = ", string(data), "\n")
		
			err = json.Unmarshal(data, into)
println("dfRequest 555, err=", err)
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

//router.POST("/lapi/v1/namespaces/:namespace/volumes", CreateVolume)
println("CreateVolumn: http://datafoundry.test.app.dataos.io/lapi/v1/namespaces/:namespace/volumes", )
println("CreateVolumn:", url)

	options := &VolumnCreateOptions{
		volumnName,
		size,
		kapi.ObjectMeta {
			Annotations: map[string]string {
				"dadafoundry.io/create-by": oc.username,
			},
		},
	}

fmt.Println("CreateVolumn: options=", options)
	
	err := dfRequest("POST", url, oc.BearerToken(), options, nil)

	return err
}

func DeleteVolumn(volumnName string) error {
	oc := OC()

	url := DfProxyApiPrefix() + "/namespaces/" + oc.Namespace() + "/volumes/" + volumnName
	
	err := dfRequest("DELETE", url, oc.BearerToken(), nil, nil)
	
	return err
}

