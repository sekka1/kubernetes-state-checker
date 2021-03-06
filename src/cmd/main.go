package main

import (
	"checker"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"reflect"
	"strconv"
	"unsafe"

	yaml "gopkg.in/yaml.v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type conf struct {
	Some                   string          `yaml:"some"`
	Random                 int64           `yaml:"random"`
	KubernetesStateChecker []checker.Check `yaml:"kubernetes-state-checker"`
}

func (c *conf) getConf() *conf {

	yamlFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

func main() {

	// Get kubeconfig
	// kubeconfig setup example: https://github.com/kubernetes/client-go/blob/master/examples/out-of-cluster-client-configuration/main.go
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Get input yaml with checks
	var c conf
	c.getConf()

	for _, aCheck := range c.KubernetesStateChecker {

		// convert to yaml
		valuesYaml, err := yaml.Marshal(aCheck)
		if err != nil {
			panic(err.Error())
		}

		// Execute the check runner
		chk := checker.New(
			BytesToString(valuesYaml),
			clientset,
			aCheck.Ttype,
			aCheck.Name,
			aCheck.Description,
			aCheck.Namespace,
			aCheck.Values,
		)
		results := chk.Run()

		fmt.Println(fmt.Sprintf(`+----------------------+----------------------------------------------------------------------+------+-------------+
	| Test Type          | Name                                                                 | Pass | Message     |
	+----------------------+----------------------------------------------------------------------+------+-------------+
	| %s | %s | %s | %s  |
	+----------------------+----------------------------------------------------------------------+------+-------------+`,
			aCheck.Ttype, c.KubernetesStateChecker[0].Name, strconv.FormatBool(results.DidPass), results.Message))

	}

}

func BytesToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{bh.Data, bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}
