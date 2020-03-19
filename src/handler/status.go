package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

func Status(runningProcceses *map[int]string,
	finishedProcesses *map[int]string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		type ResultItem struct {
			Command string `json:"command"`
			ProcessId int `json:"process_id"`
		}
		type Result struct {
			RunningJobs  []ResultItem `json:"running_process_ids"`
			FinishedJobs []ResultItem `json:"finished_jobs"`
		}
		res := &Result{}
		for k := range *finishedProcesses {
			res.FinishedJobs = append(res.FinishedJobs, ResultItem{
				Command:   (*finishedProcesses)[k],
				ProcessId: k,
			})
		}
		for k := range *runningProcceses {
			res.RunningJobs = append(res.RunningJobs, ResultItem{
				Command:   (*runningProcceses)[k],
				ProcessId: k,
			})
		}
		sort.Slice(res.RunningJobs, func(i, j int) bool {
			return strings.Compare(res.RunningJobs[i].Command, res.RunningJobs[j].Command) > 0
		})
		sort.Slice(res.FinishedJobs, func(i, j int) bool {
			return strings.Compare(res.FinishedJobs[i].Command, res.FinishedJobs[j].Command) > 0
		})
		b, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		_,_ = w.Write(b)
	}
}
