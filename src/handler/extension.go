package handler

import (
	"common"
	"net/http"
	"strings"
	"yaml_config"
)

func Extension(config *yaml_config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		output := ""
		writer := func(text string) {
			output += text
		}
		parts := strings.Split(r.RequestURI, "/")
		goroot, err := config.GetStringByKey("go root")
		common.PanicOnError(err)
		err = common.RunLinuxCommand(goroot + " run extension/" + parts[len(parts) - 1] + "/main.go", writer, nil)
		if err != nil {
			w.WriteHeader(500)
			_, _ = w.Write([]byte(err.Error()))
			_, _ = w.Write([]byte(output))
			return
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(output))
	}
}
