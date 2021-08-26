package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"

	"github.com/kubeovn/ces-controller/pkg/as3"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed adc-init-template.json
var adcInitTemplate string

func init() {
	configFile := "/Users/alauda/go/src/github.com/kubeovn/ces-controller/cmd/ces/conf.yaml"
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(fmt.Errorf("failed to read config file %s: %v", configFile, err))
	}
	var c as3.As3Config
	err = yaml.Unmarshal(configData, &c)
	if err != nil {
		panic(err)
	}
	as3.SetAs3Config(c)
}
