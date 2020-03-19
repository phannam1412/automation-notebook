package yaml_config

import (
	"common"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type AutoLayout struct {
	info map[string]map[string]AutoLayoutItem
}

type RunPythonCode struct {
	Content string `yaml:"content"`
	FromDockerContainer string `yaml:"from docker container"`
}

type AutoLayoutItem struct {
	Type string `yaml:"type"`
	OnSubmitSaveInputToFile string `yaml:"on submit save input to file"`
	RunPythonCodeOnInputSubmit *RunPythonCode `yaml:"run python code on input submit"`
}

func NewAutoLayout(yamlStr string) (*AutoLayout, error) {
	out := map[string]map[string]AutoLayoutItem{}
	err := yaml.Unmarshal([]byte(yamlStr), &out)
	if err != nil {
		return nil, err
	}
	return &AutoLayout{
		info: out,
	}, nil
}

func (this *AutoLayout) ProcessInput(formula string, input string) string {
	output := ""
	var info map[string]AutoLayoutItem
	var ok bool
	if info, ok = this.info[formula]; !ok {
		panic(fmt.Errorf("no auto layout formula " + formula))
	}
	// save input to file
	for _,v := range info {
		if v.Type == "input" && v.OnSubmitSaveInputToFile != "" {
			wd, err := os.Getwd()
			common.PanicOnError(err)
			common.PanicOnError(ioutil.WriteFile(wd + "/" + v.OnSubmitSaveInputToFile, []byte(input), 0777))
			break
		}
	}
	for name,v := range info {
		if v.RunPythonCodeOnInputSubmit != nil {
			func (name string) {
				wd, err := os.Getwd()
				common.PanicOnError(err)
				common.PanicOnError(ioutil.WriteFile(wd + "/tmp/process.py", []byte(v.RunPythonCodeOnInputSubmit.Content), 0777))
				err = common.RunLinuxCommand("docker exec " + v.RunPythonCodeOnInputSubmit.FromDockerContainer + " python /app/tmp/process.py", func(text string) {
					output += "===> name\n\n"
				}, nil)
				if err != nil {
					//output += err.Error() + "\n\n"
				}
			}(name)
		}
	}
	return output
}

func (this *AutoLayout) Render(formula string) (string, error) {
	output := ""
	var info map[string]AutoLayoutItem
	var ok bool
	if info, ok = this.info[formula]; !ok {
		return "", fmt.Errorf("no auto layout formula " + formula)
	}
	for name,v := range info {
		if v.Type == "input" {
			output += fmt.Sprintf(`
    <div style="display: flex;flex: 1;flex-direction: row;margin: 5px;">
        <div style="display: flex;flex:1;padding: 5px;align-items: center;">%s</div>
        <div style="display: flex;flex:2;padding: 5px;flex-direction: column;min-height: 200px;">
            <textarea style="display: flex;flex: 3;"></textarea>
            <div style="display: flex;flex: 1;">
                <input type="submit" />
            </div>
        </div>
    </div>
`, name)
		} else {
			output += fmt.Sprintf(`
    <div style="display: flex;flex: 1;flex-direction: row;margin: 5px;min-height: 100px;">
        <div style="display: flex;flex:1;padding: 5px;align-items: center;">%s</div>
		<textarea style="display: flex; flex: 2"></textarea>
    </div>
`, name)
		}
	}
	output = fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Auto layout</title>
	<script src="/public/ws.js"></script>
</head>
<body>
	%s
</body>
</html>
`, output)
	return output, nil
}
