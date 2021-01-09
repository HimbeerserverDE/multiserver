package multiserver

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

var Config map[interface{}]interface{}

// LoadConfig loads the configuration file
func LoadConfig() error {
	data, err := ioutil.ReadFile("config/multiserver.yml")
	if err != nil {
		return err
	}

	Config = make(map[interface{}]interface{})

	err = yaml.Unmarshal(data, &Config)
	if err != nil {
		return err
	}

	return nil
}

// GetKey returns a key in the configuration
func GetConfKey(key string) interface{} {
	keys := strings.Split(key, ":")
	c := Config
	for i := 0; i < len(keys)-1; i++ {
		if c[keys[i]] == nil {
			return nil
		}
		c = c[keys[i]].(map[interface{}]interface{})
	}

	return c[keys[len(keys)-1]]
}
