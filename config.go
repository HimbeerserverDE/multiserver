package multiserver

import (
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var config map[interface{}]interface{}

var defaultConfig []byte = []byte(`host: "0.0.0.0:33000"
player_limit: -1
servers:
  lobby:
    address: "127.0.0.1:30000"
default_server: lobby
force_default_server: true
`)

// LoadConfig loads the configuration file
func loadConfig() error {
	os.Mkdir("config", 0775)

	_, err := os.Stat("config/multiserver.yml")
	if os.IsNotExist(err) {
		ioutil.WriteFile("config/multiserver.yml", defaultConfig, 0664)
	}

	data, err := ioutil.ReadFile("config/multiserver.yml")
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

// GetKey returns a key in the configuration
func GetConfKey(key string) interface{} {
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
