package handler

import (
	"common"
	"core"
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os/user"
	"strings"
	"sync"
	"time"
	"yaml_config"
)

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

type SearchResultItem struct {
	Value string `json:"value"`
}

type SearchResult struct {
	Suggestions []SearchResultItem `json:"suggestions"`
}

func NewSearchResultFromStringArray(lines []string) *SearchResult {
	output := SearchResult{}
	for _, v := range lines {
		item := SearchResultItem{Value: v}
		output.Suggestions = append(output.Suggestions, item)
	}
	return &output
}

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

func GetCommandCenter(config yaml_config.IConfig, curl yaml_config.ICurl, test *yaml_config.AutomatedCheckCollection) (*core.CommandCenter, error) {
	commandCenter := core.NewCommandCenter(config, curl, test)
	err := commandCenter.Reload()
	return commandCenter, err
}

func GetYamlCurl(config yaml_config.IConfig) (*yaml_config.CurlCollection, error) {
	data, err := ioutil.ReadFile("config/curl.yml")
	if err != nil {
		return nil, err
	}
	curl, err := yaml_config.NewCurlCollection(config, data)
	if err != nil {
		return nil, err
	}
	return curl, nil
}

func GetIntegrationTest(config yaml_config.IConfig, curl yaml_config.ICurl) (*yaml_config.AutomatedCheckCollection, error) {
	data, err := ioutil.ReadFile("config/automated-check.yml")
	if err != nil {
		return nil, err
	}
	test, err := yaml_config.NewAutomatedCheckCollection(config, curl, data)
	if err != nil {
		return nil, err
	}
	return test, nil
}

func GetFuzzySearch(commandCenter *core.CommandCenter) (*common.FuzzySearch, error) {
	lines, err := commandCenter.GetCommandNames()
	if err != nil {
		return nil, err
	}
	return common.NewFuzzySearch(lines), nil
}

func handleError(w http.ResponseWriter, err error, logWatcherChannels map[int]chan LogItem, processId int) {
	fmt.Println(err)
	w.WriteHeader(500)
	if processId == 0 {
		_,_ = w.Write([]byte(err.Error()))
		return
	}
	item := LogItem{
		ProcessId: processId,
		Log:       err.Error(),
	}
	for k := range logWatcherChannels {
		logWatcherChannels[k] <- item
	}
}

func getHistory() ([]byte, error) {
	b, err := ioutil.ReadFile("history.txt")
	if err != nil {
		return nil, err
	}
	history := string(b)
	lines := strings.Split(history, "\n")
	m := map[string]bool{}
	var res []string
	for a := len(lines)  - 1;a >= 0;a-- {
		v := lines[a]
		if v == "" {
			continue
		}
		if _, ok := m[v]; !ok {
			m[v] = true
			res = append(res, v)
		}
	}
	return json.Marshal(res)
}

type LogItem struct {
	ProcessId int
	Log string
}

func StartWatcher(reloadFn func()) {
	watcher, err := fsnotify.NewWatcher()
	common.PanicOnError(err)
	err = watcher.Add("config")
	common.PanicOnError(err)
	err = watcher.Add("formula")
	common.PanicOnError(err)
	var changes []string

	// consumer
	go func() {
		var mutex sync.Mutex
		for {
			time.Sleep(time.Second)
			if len(changes) > 0 {
				mutex.Lock()
				changes = nil
				mutex.Unlock()
				reloadFn()
			}
		}
	}()

	// producer
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				changes = append(changes, event.Name)
				fmt.Printf("EVENT! %#v\n", event)
			case err := <-watcher.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}()
}

