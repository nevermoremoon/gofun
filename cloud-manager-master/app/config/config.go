package config

import (
	"fmt"
	"github.com/toolkits/file"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"strings"
)

func Parse(cfg string) error {
	if cfg == "" {
		return fmt.Errorf("Please specify configuration file")
	}

	if !file.IsExist(cfg) {
		return fmt.Errorf("Configuration file %s is not exist.", cfg)
	}

	bs, err := ioutil.ReadFile(cfg)
	if err != nil {
		return fmt.Errorf("Read configuration file %s fail: %s", cfg, err.Error())
	}

	var c ConfYaml
	err = yaml.Unmarshal(bs, &c)
	if err != nil {
		return fmt.Errorf("Parse configuration file %s fail: %s", cfg, err.Error())
	}

	c.InitCloud()
	G = &c

	if G.N9eInfo.Endpoint == "" {
		return fmt.Errorf("N9e endpoint not prefix.")
	} else {
		G.N9eInfo.Endpoint = strings.TrimRight(G.N9eInfo.Endpoint, "/")
	}

	log.Printf("[I] %+v", Version)
	log.Printf("[I] load configuration file %s successfully", cfg)
	/*
	b, err := json.Marshal(c)
	if err == nil {
		log.Printf("[I] %+v", string(b))
	}
	*/
	return nil
}
