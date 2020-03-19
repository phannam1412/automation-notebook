package core

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"yaml_config"
)

func GetSshItemByKey(key string) (*yaml_config.SshItem, error) {
	b, err := ioutil.ReadFile("config/ssh.yml")
	if err != nil {
		return nil, err
	}
	output := map[string]yaml_config.SshItem{}
	err = yaml.Unmarshal(b, output)
	if err != nil {
		return nil, err
	}
	if v, ok := output[key]; ok {
		return &v, nil
	}
	return nil, nil
}

