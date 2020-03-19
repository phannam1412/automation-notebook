package main

import (
	"common"
	"core"
	"fmt"
	"handler"
	"net/http"
	"strings"
	"time"
)

func main() {
	config, err := core.GetConfig()
	common.PanicOnError(err)
	curl, err := handler.GetYamlCurl(config)
	common.PanicOnError(err)
	integrationTest, err := handler.GetIntegrationTest(config, curl)
	common.PanicOnError(err)
	commandCenter, err := handler.GetCommandCenter(config, curl, integrationTest)
	common.PanicOnError(err)
	fuzzySearch, err := handler.GetFuzzySearch(commandCenter)
	common.PanicOnError(err)
	logWatcherChannels := map[int]chan handler.LogItem{}
	runningProcceses := map[int]string{}
	finishedProcesses := map[int]string{}
	forceStopChannels := map[int]chan bool{}
	processAutoIncrementId := 0
	logWatcherAutoIncrementId := 0
	storedLogs := map[int]string{}
	handler.PanicOnError(err)
	reloadFn := func() {
		fmt.Println("reloading...")
		newConfig, err := core.GetConfig()
		if err != nil {
			fmt.Println(err)
			return
		}
		*config = *newConfig
		newCurl, err := handler.GetYamlCurl(config)
		if err != nil {
			fmt.Println(err)
			return
		}
		*curl = *newCurl
		newIntegrationTest, err := handler.GetIntegrationTest(config, curl)
		if err != nil {
			fmt.Println(err)
			return
		}
		*integrationTest = *newIntegrationTest
		newCommandCenter, err := handler.GetCommandCenter(config, curl, integrationTest)
		if err != nil {
			fmt.Println(err)
			return
		}
		*commandCenter = *newCommandCenter
		newFuzzySearch, err := handler.GetFuzzySearch(commandCenter)
		if err != nil {
			fmt.Println(err)
			return
		}
		*fuzzySearch = *newFuzzySearch
		fmt.Println("reloaded successfully !!!")
	}
	handler.StartWatcher(reloadFn)
	// reload command center continuously
	reloadInterval, err := config.GetIntByKey("reload command suggestion interval in seconds")
	handler.PanicOnError(err)
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(reloadInterval))
			fmt.Println("reloading...")
			reloadFn()
		}
	}()
	http.HandleFunc("/public/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, strings.TrimLeft(r.RequestURI, "/"))
	})
	http.HandleFunc("/search", handler.Search(fuzzySearch))
	http.HandleFunc("/run", handler.RunCommand(&processAutoIncrementId, commandCenter, &finishedProcesses, &logWatcherChannels, &runningProcceses, &storedLogs, config, &forceStopChannels))
	http.HandleFunc("/close-process", handler.CloseProcess(&runningProcceses, &finishedProcesses, &storedLogs, &forceStopChannels))
	http.HandleFunc("/log", handler.Log(&runningProcceses, &finishedProcesses, &storedLogs, &logWatcherAutoIncrementId, &logWatcherChannels))
	http.HandleFunc("/status", handler.Status(&runningProcceses, &finishedProcesses))
	http.HandleFunc("/p/", handler.Extension(config))
	http.HandleFunc("/", handler.All)
	err = http.ListenAndServe(":1234", nil)
	if err != nil {
		panic(err)
	}
}
