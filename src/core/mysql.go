package core

import (
	"common"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"yaml_config"
)

func GetMysqlItemFromConfig(configKey string) *yaml_config.MysqlItem {
	b, err := ioutil.ReadFile("config/mysql.yml")
	common.PanicOnError(err)
	items := map[string]yaml_config.MysqlItem{}
	common.PanicOnError(yaml.Unmarshal(b, items))
	if v, ok := items[configKey]; ok {
		return &v
	}
	panic(fmt.Errorf("mysql config key %s does not exist", configKey))
}

