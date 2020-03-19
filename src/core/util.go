package core

import (
	"common"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/user"
	"repository"
	"yaml_config"
)

func GetConfig() (*yaml_config.Config, error) {
	b, err := ioutil.ReadFile("config/config.yml")
	if err != nil {
		return nil, err
	}
	configData := map[string]string{}
	err = yaml.Unmarshal(b, configData)
	if err != nil {
		return nil, err
	}
	u, err  := user.Current()
	if err != nil {
		return nil, err
	}
	b, err = ioutil.ReadFile("config/config."+u.Username+".yml")
	if err == nil {
		additionalConfigData := map[string]string{}
		err := yaml.Unmarshal(b, additionalConfigData)
		if err != nil {
			return nil, err
		}
		configData = common.StringArrayMerge(configData, additionalConfigData)
	}
	config, err := yaml_config.NewConfig(configData)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func RunBashScriptFromWorkingDirectoryConfig(config yaml_config.IConfig, script string, workingDirectoryConfig string, writer common.IWriter, forceStop chan bool) error {
	wd, err := os.Getwd()
	common.PanicOnError(err)
	if workingDirectoryConfig != "" {
		workDir, err := config.GetStringByKey(workingDirectoryConfig)
		if err != nil {
			return err
		}
		wd = workDir
	}
	return common.RunBashScript(script, wd, writer, forceStop)
}

func GetCurlItemFromConfig(configKey string) *yaml_config.CurlItem {
	b, err := ioutil.ReadFile("config/curl.yml")
	common.PanicOnError(err)
	items := map[string]yaml_config.CurlItem{}
	common.PanicOnError(yaml.Unmarshal(b, items))
	if v, ok := items[configKey]; ok {
		return &v
	}
	panic(fmt.Errorf("curl config key %s does not exist", configKey))
}

func GetAutomatedCheckItemFromConfig(configKey string) *yaml_config.AutomatedCheckItem {
	b, err := ioutil.ReadFile("config/automated-check.yml")
	common.PanicOnError(err)
	items := map[string]yaml_config.AutomatedCheckItem{}
	common.PanicOnError(yaml.Unmarshal(b, items))
	if v, ok := items[configKey]; ok {
		return &v
	}
	panic(fmt.Errorf("curl config key %s does not exist", configKey))
}

func ReducerFilterPassedJobs(jobs []common.Job) []common.Job {
	mysqlItem := GetMysqlItemFromConfig("help db")
	db, err := mysqlItem.GetConnection()
	common.PanicOnError(err)
	return repository.FilterByJobExistsAndPassed(db, jobs)
}
