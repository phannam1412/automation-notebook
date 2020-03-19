package handler

import (
	"net/http"
	"strconv"
)

func CloseProcess(runningProcesses *map[int]string, finishedProcesses *map[int]string, storedLogs *map[int]string, forceStopChannels *map[int]chan bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		param := r.PostFormValue("process_id")
		processId, err := strconv.Atoi(param)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		if _, ok := (*finishedProcesses)[processId]; ok {
			delete(*finishedProcesses, processId)
			delete(*storedLogs, processId)
		}
		if _, ok := (*runningProcesses)[processId]; ok {
			(*finishedProcesses)[processId] = (*runningProcesses)[processId]
			delete(*runningProcesses, processId)
			(*forceStopChannels)[processId] <- true
		}
		w.WriteHeader(200)
	}
}
