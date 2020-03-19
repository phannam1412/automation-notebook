package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func Log(runningProcesses *map[int]string,
	finishedProcesses *map[int]string,
	storedLogs *map[int]string,
	logWatcherAutoIncrementId *int,
	logWatcherChannels *map[int]chan LogItem) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("new connection opened for viewing log")
		query := r.URL.Query()
		input := query.Get("process_id")
		var processIdsForWatching []int
		if input != "" {
			tmp := strings.Split(input, ",")
			for _, v := range tmp {
				id, err := strconv.Atoi(v)
				if err != nil {
					handleError(w, err, *logWatcherChannels, 0)
				}
				processIdsForWatching = append(processIdsForWatching, id)
			}
		}


		// filter invalid processIds
		var filtered []int
		for _, processIdForWatching := range processIdsForWatching {
			found := false
			for runningProcessId := range *runningProcesses {
				if runningProcessId == processIdForWatching {
					found = true
					break
				}
			}
			if found {
				filtered = append(filtered, processIdForWatching)
			}
		}
		for _, processIdForWatching := range processIdsForWatching {
			found := false
			for finishedProcessId := range *finishedProcesses {
				if finishedProcessId == processIdForWatching {
					found = true
					break
				}
			}
			if found {
				filtered = append(filtered, processIdForWatching)
			}
		}
		processIdsForWatching = filtered

		w.WriteHeader(200)
		if f, ok := w.(http.Flusher); ok {

			// work-around for buffering
			_, _ = w.Write([]byte(strings.Repeat(" ", 5000)))
			f.Flush()

			logChannel := make(chan LogItem, 10000)
			*logWatcherAutoIncrementId++
			index := *logWatcherAutoIncrementId
			(*logWatcherChannels)[index] = logChannel
			notify := w.(http.CloseNotifier).CloseNotify()
			go func() {
				<-notify
				fmt.Println("connection for viewing log is closed")
				delete(*logWatcherChannels, index)
			}()

			// display stored log for this log channel
			for _, v := range processIdsForWatching {
				if val, ok := (*storedLogs)[v]; ok {
					_,_ = w.Write([]byte(val))
				}
			}
			f.Flush()

			for {
				logItem := <- logChannel
				// subscribe to all process logs
				if len(processIdsForWatching) == 0 {
					_, err := w.Write([]byte(logItem.Log))
					if err != nil {
						return
					}
					f.Flush()
					continue
				}
				// subscribe to only selected process logs
				for _, v := range processIdsForWatching {
					if v == logItem.ProcessId {
						_, err := w.Write([]byte(logItem.Log))
						if err != nil {
							return
						}
						f.Flush()
					}
				}
			}
		}
	}
}
