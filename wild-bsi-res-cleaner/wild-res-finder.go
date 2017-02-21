package main

import (
	"log"
	"fmt"
	"os"
	"bufio"
	"bytes"
	"io"
	"flag"
)

var instanceIdsInETCD_Openshift = map[string]bool{}
var instanceIdsInETCD_DatabaseShare = map[string]bool{}

var instanceIdsInDF = map[string]bool{}

var getInstanceInfoCommands_Openshift = make([]string, 0, 1000)
var getInstanceInfoCommands_DatabaseShare = make([]string, 0, 1000)

var instanceIdsInDF_NotFound = make([]string, 0, 1000)
var instancePodRandomIdsInDF = make([]string, 0, 1000)

func collectInstanceIdsInETCD() {
	collect := func(m map[string]bool, filename string, prefix []byte) {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		reader := bufio.NewReader(f)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatal(err)
			}
			if bytes.HasPrefix(line, prefix) {
				m[string(line[len(prefix):])] = false
			}
		}
	}
	
	collect(instanceIdsInETCD_DatabaseShare, 
		"bsi-databaseshare-instances-in-etcd", 
		[]byte("/servicebroker/databaseshare/instance/"))
	collect(instanceIdsInETCD_Openshift, 
		"bsi-openshift-instances-in-etcd", 
		[]byte("/servicebroker/openshift/instance/"))
}

func collectInstanceIdsInDF() {
	prefix := []byte("instance_id: ")
	collect := func(filename string, allowNotExist bool) {
		f, err := os.Open(filename)
		if err != nil {
			if err == os.ErrNotExist && allowNotExist {
				return
			}
			
			log.Fatal(err)
		}
		reader := bufio.NewReader(f)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatal(err)
			}
			line = bytes.TrimSpace(line)
			if bytes.HasPrefix(line, prefix) {
				instanceIdsInDF[string(line[len(prefix):])] = false
			}
		}
	}
	
	collect("bsis-region-north1", false)
	collect("bsis-region-north2", true)
}

func collectInstancePodRandomIdsInDF() {
	collect := func(filename string) {
		f, err := os.Open(filename)
		if err != nil {
			log.Fatal(err)
		}
		reader := bufio.NewReader(f)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatal(err)
			}
			
			if bytes.Index(line, []byte("Key not found")) >= 0 {
				const substr = "/servicebroker/openshift/instance/"
				start := bytes.Index(line, []byte(substr))
				if start >= 0 {
					start += len(substr)
					end := bytes.Index(line[start:], []byte("/_info"))
					if end >= 0 {
						end += start
						instanceId := string(line[start:end])
						if len(instanceId) > 0 {
							instanceIdsInDF_NotFound = append(instanceIdsInDF_NotFound, instanceId)
						}
					}
				}
				continue
			}

			const substr = `","url":"`
			start := bytes.Index(line, []byte(substr))
			if start >= 0 {
				start += len(substr)
				end := bytes.Index(line[start:], []byte(`","`))
				if end >= 0 {
					end += start
					randomId := string(line[start:end])
					if len(randomId) > 0 {
						instancePodRandomIdsInDF = append(instancePodRandomIdsInDF, randomId)
					}
				}
			}
		}
	}

	collect("wild-bsi-info-openshift")
}

func createCommand_GetWildInstanceInfo(instanceId, bsiGroup string) string {
	return "$ETCDCTL get /servicebroker/" +
		bsiGroup +
		"/instance/" +
		instanceId +
		"/_info"
}

func printHelp() {
	log.Print(`
	bsi-openshift-wild-instances-in-etcd     : output wild bsi instance IDs for service-brokers-openshift
	bsi-databaseshare-wild-instances-in-etcd : output wild bsi instance IDs for service-brokers-databaseshare
`)
}

func main() {
	log.SetFlags(0)
	
	flag.Parse()
	
	collectInstanceIdsInETCD()
	collectInstanceIdsInDF()

	tag := func(toTag, toCheck map[string]bool) {
		for id := range toTag {
			_, ok := toCheck[id]
			if ok {
				toTag[id] = true
			}
		}
	}

	tag(instanceIdsInDF, instanceIdsInETCD_Openshift)
	tag(instanceIdsInDF, instanceIdsInETCD_DatabaseShare)
	tag(instanceIdsInETCD_Openshift, instanceIdsInDF)
	tag(instanceIdsInETCD_DatabaseShare, instanceIdsInDF)
	
	/*
	log.Println("===== Following bsis have no instances corresponded (maybe they are ceated in north2):")
	for id, has := range instanceIdsInDF {
		if !has {
			log.Println(id)
		}
	}
	
	log.Println()
	
	log.Println("===== Following instance not used by any bsi (databaseshare):")
	for id, has := range instanceIdsInETCD_DatabaseShare {
		if !has {
			log.Println(id)
			getInstanceInfoCommands_DatabaseShare = append(
				getInstanceInfoCommands_DatabaseShare,
				createCommand_GetWildInstanceInfo(id, "databaseshare"),
			)
		}
	}
	
	log.Println()
	
	log.Println("===== Following instance not used by any bsi (openshift):")
	for id, has := range instanceIdsInETCD_Openshift {
		if !has {
			log.Println(id)
			getInstanceInfoCommands_Openshift = append(
				getInstanceInfoCommands_Openshift,
				createCommand_GetWildInstanceInfo(id, "openshift"),
			)
		}
	}
	*/
	
	
	switch flag.Arg(0) {
	default:
		printHelp()
		
	case "bsi-openshift-wild-instances-in-etcd":
		fmt.Println()
		for id, has := range instanceIdsInETCD_Openshift {
			if !has {
				fmt.Println(id)
			}
		}
		fmt.Println()
	case "bsi-databaseshare-wild-instances-in-etcd":
		fmt.Println()
		for id, has := range instanceIdsInETCD_DatabaseShare {
			if !has {
				fmt.Println(id)
			}
		}
		fmt.Println()
	case "wild-bsi-random-ids":
		fmt.Println()
		collectInstancePodRandomIdsInDF()
		for _, randomId := range instancePodRandomIdsInDF {
			fmt.Println(randomId)
		}
		fmt.Println()
	}
	
	/*
	log.Println()

	log.Println("===== Commands to get bsi info (databaseshare):")
	for _, command := range getInstanceInfoCommands_DatabaseShare {
		log.Println(command)
	}
	
	log.Println()

	log.Println("===== Commands to get bsi info (openshift):")
	for _, command := range getInstanceInfoCommands_Openshift {
		log.Println(command)
	}

	// ...
	
	log.Println()

	log.Println("===== bsi instances not found:")
	for _, instanceId := range instanceIdsInDF_NotFound {
		log.Println(instanceId)
	}
	
	log.Println()

	log.Println("===== Wild openshift BSI random IDs (please save them into: wild-bsi-random-ids)")
	for _, randomId := range instancePodRandomIdsInDF {
		log.Println(randomId)
	}
	*/
}