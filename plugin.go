package multiserver

import (
	"io/ioutil"
	"log"
)

type plugin struct {
	Name string
	Path string
}

var plugins []plugin

func LoadPlugins() error {
	files, err := ioutil.ReadDir("plugins")
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if file.IsDir() {
			subfiles, err := ioutil.ReadDir("plugins/" + file.Name())
			if err != nil {
				return err
			}
			
			for _, subfile := range subfiles {
				if subfile.Name() == "init.lua" {
					log.Print("Loading plugin " + file.Name())
					plugins = append(plugins, plugin{Name: file.Name(), Path: "plugins/" + file.Name()})
					if err = l.DoFile("plugins/" + file.Name() + "/" + subfile.Name()); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}
