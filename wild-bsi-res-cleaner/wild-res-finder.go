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

var instanceIdsInETCD_ButNotFoundInDF = make([]string, 0, 1000)
var instancePodRandomIdsInDF_NoRelatedBSIs = make([]string, 0, 1000)

var instancePodRandomIdsInETCD = map[string]bool{}
var instancePodRandomIdsInDF_ButNotInETCD = map[string]bool{}

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
			if os.IsNotExist(err) && allowNotExist {
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
							instanceIdsInETCD_ButNotFoundInDF = append(instanceIdsInETCD_ButNotFoundInDF, instanceId)
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
						instancePodRandomIdsInDF_NoRelatedBSIs = append(instancePodRandomIdsInDF_NoRelatedBSIs, randomId)
					}
				}
			}
		}
	}

	collect("wild-bsi-info-openshift")
}

func collectUnrecordedBsiRandomIds() {
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

			const substr = `","url":"`
			start := bytes.Index(line, []byte(substr))
			if start >= 0 {
				start += len(substr)
				end := bytes.Index(line[start:], []byte(`","`))
				if end >= 0 {
					end += start
					randomId := string(line[start:end])
					if len(randomId) > 0 {
						instancePodRandomIdsInETCD[randomId] = true
					}
				}
			}
		}
	}
	
	collect2 := func(filename string, allowNotExist bool) {
		f, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) && allowNotExist {
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

			const substr = `sb-`
			start := bytes.Index(line, []byte(substr))
			if start >= 0 {
				start += len(substr)
				end := bytes.Index(line[start:], []byte(`-`))
				if end >= 0 {
					end += start
					randomId := string(line[start:end])
					if len(randomId) > 0 && instancePodRandomIdsInETCD[randomId] == false {
						instancePodRandomIdsInDF_ButNotInETCD[randomId] = true
					}
				}
			}
		}
	}

	collect("bsi-databaseshare-instances-info-in-etcd")
	collect("bsi-openshift-instances-info-in-etcd")
	collect2("bsi-pods-region-north1", false)
	collect2("bsi-pods-region-north2", true)
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
		for _, randomId := range instancePodRandomIdsInDF_NoRelatedBSIs {
			fmt.Println(randomId)
		}
		fmt.Println()
	case "unrecorded-bsi-random-ids":
		fmt.Println()
		collectUnrecordedBsiRandomIds()
		for randomId, _ := range instancePodRandomIdsInDF_ButNotInETCD {
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
	for _, instanceId := range instanceIdsInETCD_ButNotFoundInDF {
		log.Println(instanceId)
	}
	
	log.Println()

	log.Println("===== Wild openshift BSI random IDs (please save them into: wild-bsi-random-ids)")
	for _, randomId := range instancePodRandomIdsInDF_NoRelatedBSIs {
		log.Println(randomId)
	}
	*/
}