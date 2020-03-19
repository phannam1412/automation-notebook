package handler

import (
	"common"
	"core"
	"fmt"
	"net/http"
	"os"
	"strings"
	"yaml_config"
)

func RunCommand(processAutoIncrementId *int,
	commandCenter *core.CommandCenter,
	finishedProcesses *map[int]string,
	logWatcherChannels *map[int]chan LogItem,
	runningProcesses *map[int]string,
	storedLogs *map[int]string,
	config *yaml_config.Config,
	forceStopChannels *map[int]chan bool) func(w http.ResponseWriter, r *http.Request) {
	maxStoredLogCharacters, err := config.GetIntByKey("maximum stdout characters to be stored for a process")
	PanicOnError(err)
	return func(w http.ResponseWriter, r *http.Request) {
		fullCommand := r.PostFormValue("command")
		command := fullCommand
		param := ""
		if strings.Contains(command, ":") {
			pieces := strings.Split(command, ":")
			command = pieces[0]
			param = pieces[1]
		}
		*processAutoIncrementId++
		processId := *processAutoIncrementId
		commandToBeExecuted, err := commandCenter.GetCommandInfo(command)
		if err != nil {
			(*finishedProcesses)[processId] = "NOT FOUND: " + fullCommand
			handleError(w, err, *logWatcherChannels, 0)
			return
		}
		(*runningProcesses)[processId] = fullCommand
		fmt.Printf("START command %s\n", fullCommand)

		// append file
		f, err := os.OpenFile("history.txt", os.O_APPEND|os.O_WRONLY, 0777)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if _, err = f.WriteString(fullCommand + "\n"); err != nil {
			panic(err)
		}

		writer := func(text string) {
			(*storedLogs)[processId] += text // WARNING: should mutex lock to prevent concurrent write
			if len((*storedLogs)[processId]) > maxStoredLogCharacters {
				(*storedLogs)[processId] = (*storedLogs)[processId][int(maxStoredLogCharacters / 5):]
			}
			item := LogItem{
				ProcessId: processId,
				Log:       text,
			}
			for _, ch := range *logWatcherChannels {
				ch <- item
			}
		}
		forceStopChan := make(chan bool)
		(*forceStopChannels)[processId] = forceStopChan
		go func() {
			writer(">>> RUNNING COMMAND " + fullCommand + "\n")
			err = commandToBeExecuted(writer, param, forceStopChan)
			writer(fmt.Sprintf(">>> END COMMAND command %s\n", fullCommand))
			delete((*forceStopChannels), processId)
			delete(*runningProcesses, processId)
			(*finishedProcesses)[processId] = fullCommand
			if err != nil {

				// store error log
				// @todo duplicate with store stdout log from /run route
				maxStoredLogCharacters, err2 := config.GetIntByKey("maximum stdout characters to be stored for a process")
				common.PanicOnError(err2)
				(*storedLogs)[processId] += err.Error()
				if len((*storedLogs)[processId]) > maxStoredLogCharacters {
					(*storedLogs)[processId] = (*storedLogs)[processId][int(maxStoredLogCharacters / 5):]
				}

				item := LogItem{
					ProcessId: processId,
					Log:       "ERROR: " + err.Error(),
				}
				for _, ch := range *logWatcherChannels {
					ch <- item
				}
				return
			}
		}()
		w.WriteHeader(200)
	}
}
