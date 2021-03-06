package main

import (
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var config map[interface{}]interface{}

var defaultConfig []byte = []byte(`servers:
  lobby:
    address: "127.0.0.1:30000"
default_server: lobby
force_default_server: true
`)

func loadConfig() error {
	os.Mkdir("config", 0777)

	_, err := os.Stat("config/multiserver.yml")
	if os.IsNotExist(err) {
		os.WriteFile("config/multiserver.yml", defaultConfig, 0666)
	}

	data, err := os.ReadFile("config/multiserver.yml")
	if err != nil {
		return err
	}

	config = make(map[interface{}]interface{})

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	return nil
}

// ConfKey returns a key from the configuration
func ConfKey(key string) interface{} {
	if config == nil {
		loadConfig()
	}

	keys := strings.Split(key, ":")
	c := config
	for i := 0; i < len(keys)-1; i++ {
		if c[keys[i]] == nil {
			return nil
		}
		c = c[keys[i]].(map[interface{}]interface{})
	}

	return c[keys[len(keys)-1]]
}
