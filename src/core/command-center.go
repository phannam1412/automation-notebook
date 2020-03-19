package core

import (
	"common"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"yaml_config"
)

type CommandCenter struct {
	commands                 map[string]common.CommandHandler
	time                     int64
	config                   yaml_config.IConfig
	curl                     yaml_config.ICurl
	automatedCheckCollection *yaml_config.AutomatedCheckCollection
}

func NewCommandCenter(config yaml_config.IConfig, curl yaml_config.ICurl, test *yaml_config.AutomatedCheckCollection) *CommandCenter {
	return &CommandCenter{
		commands:                 nil,
		time:                     0,
		config:                   config,
		curl:                     curl,
		automatedCheckCollection: test,
	}
}

func (this *CommandCenter) reloadCommandsFromCodeFiles(newCommands map[string]common.CommandHandler) error {
	goRoot, err := this.config.GetStringByKey("go root")
	if err != nil {
		return err
	}
	return filepath.Walk("formula", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		extension := filepath.Ext(info.Name())
		if extension != ".go" {
			return nil
		}
		name := info.Name()
		if name == "" {
			return nil
		}
		nameWithoutExtension := name[:len(name) - len(extension)]
		newCommands[nameWithoutExtension] = func(w common.IWriter, param string, forceStopChan chan bool) error {
			cmd := exec.Command(goRoot, "run", "formula/" + info.Name(), param)
			proxyWriter := common.NewProxyWriter(w)
			cmd.Stdout = proxyWriter
			cmd.Stderr = proxyWriter
			err := cmd.Start()
			if err != nil {
				return err
			}
			finishChan := make(chan error)
			go func() {
				err = cmd.Wait()
				if err != nil {
					finishChan <- err
					return
				}
				finishChan <- nil
			}()
			go func() {
				<- forceStopChan
				finishChan <- nil
			}()
			return <- finishChan

		}
		return nil
	})
}

func (this *CommandCenter) reloadCommandsFromCurl(newCommands map[string]common.CommandHandler) error {
	data, err := ioutil.ReadFile("config/curl.yml")
	if err != nil {
		return err
	}
	out := map[string]interface{}{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	for k := range out {
		func(k string) {
			newCommands["curl " + k] = func(w common.IWriter, param string, forceStop chan bool) error {
				this.curl.EnableVerbose()
				this.curl.SetWriter(w)
				return this.curl.RunForKey(k)
			}
		}(k)
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromFormulaConfig(newCommands map[string]common.CommandHandler) error {
	type BashScriptInfo struct {
		Content string `yaml:"content"`
		WorkingDirectoryConfig string `yaml:"working directory config"`
	}
	type Command struct {
		OpenUrl string `yaml:"open url"`
		Output string `yaml:"output"`
		RunLinuxCommand string `yaml:"run linux command"`
		RunLinuxCommandByCsv string `yaml:"run linux command by csv"`
		RunBashScript *BashScriptInfo `yaml:"run bash script"`
	}
	data, err := ioutil.ReadFile("config/formula.yml")
	if err != nil {
		return err
	}
	out := map[string][]Command{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	for k := range out {
		func(k string, commands []Command) {
			newCommands[k] = func(w common.IWriter, param string, forceStop chan bool) (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = r.(error)
					}
				}()
				for k := range commands {
					command := commands[k]
					if command.RunLinuxCommand != "" {
						w("executing " + command.RunLinuxCommand + "\n")
						common.PanicOnError(common.RunLinuxCommand(command.RunLinuxCommand, w, forceStop))
					}
					if command.RunBashScript != nil {
						w("running bash script... \n")
						common.PanicOnError(RunBashScriptFromWorkingDirectoryConfig(this.config, command.RunBashScript.Content, command.RunBashScript.WorkingDirectoryConfig, w, forceStop))
					}
					if command.RunLinuxCommandByCsv != "" {
						w("executing " + command.RunLinuxCommandByCsv + "\n")
						common.PanicOnError(common.RunLinuxCommandByCsvWithDirectory("", command.RunLinuxCommandByCsv, w, forceStop))
					}
					if command.Output != "" {
						w(command.Output)
					}
					if command.OpenUrl != "" {
						cmd := exec.Command("sudo","-u","namph12","firefox","-new-tab","-url",command.OpenUrl)
						common.PanicOnError(cmd.Run())
					}
				}
				return nil
			}
		}(k, out[k])
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromIntegrationTest(newCommands map[string]common.CommandHandler) error {
	data, err := ioutil.ReadFile("config/automated-check.yml")
	if err != nil {
		return err
	}
	out := map[string]yaml_config.AutomatedCheckItem{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	for k := range out {
		func(k string) {
			newCommands["integration test for " + k] = func(w common.IWriter, param string, forceStop chan bool) error {
				w(">>>> START integration test for " + k + "...\n")
				this.automatedCheckCollection.SetWriter(yaml_config.IAutomatedCheckWriter(w))
				err := this.automatedCheckCollection.Run(k)
				w(">>>> END integration test for " + k + "...\n")
				return err
			}
			info := out[k]
			if info.Group != nil {
				if _, ok := newCommands["group test for " + *info.Group]; !ok {
					newCommands["group test for " + *info.Group] = func(w common.IWriter, param string, forceStop chan bool) error {
						w(fmt.Sprintf("============================== BEGIN RUNNING GROUP: %s ==============================\n", *info.Group))
						this.automatedCheckCollection.SetWriter(yaml_config.IAutomatedCheckWriter(w))
						err := this.automatedCheckCollection.RunGroup(*info.Group)
						w(fmt.Sprintf("============================== END RUNNING GROUP: %s ==============================\n", *info.Group))
						return err
					}
				}
			}
		}(k)
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromDockerCompose(newCommands map[string]common.CommandHandler) error {
	type DockerComposeServiceDefinition struct {
		Image         string      `yaml:"image,omitempty"`
		ContainerName string      `yaml:"container_name,omitempty"`
		Ports         []string    `yaml:"ports,omitempty"`
		Volumes       []string    `yaml:"volumes,omitempty"`
		Command       interface{} `yaml:"command,omitempty"`
		DependsOn     []string    `yaml:"depends_on,omitempty"`
		Environments  interface{} `yaml:"environment,omitempty"`
		Deploy        interface{} `yaml:"deploy,omitempty"`
		Build         interface{} `yaml:"build,omitempty"`
	}
	type DockerComposeDefinition struct {
		Version string                                     `yaml:"version,omitempty"`
		Services map[string]DockerComposeServiceDefinition `yaml:"services,omitempty"`
	}
	type YamlDockerComposeItem struct {
		WorkingDirectory string `yaml:"working directory"`
		DockerComposeDefinitionFromConfigPath string `yaml:"docker-compose definition from config path"`
		DockerComposeDefinition *DockerComposeDefinition `yaml:"docker-compose definition"`
	}
	data, err := ioutil.ReadFile("config/docker-compose.yml")
	if err != nil {
		return err
	}
	out := map[string]YamlDockerComposeItem{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	for k := range out {
		func(dockerComposeConfigName string) {
			info := out[dockerComposeConfigName]
			definition := info.DockerComposeDefinition
			workingDirectory := ""
			if info.DockerComposeDefinitionFromConfigPath != "" {
				path, err := this.config.GetStringByKey(info.DockerComposeDefinitionFromConfigPath)
				if err != nil {
					fmt.Println(err)
					return
				}
				workingDirectory = path
				b, err := ioutil.ReadFile(path + "/docker-compose.yml")
				if err != nil {
					fmt.Println(err)
					return
				}
				err = yaml.Unmarshal(b, &definition)
				if err != nil {
					fmt.Println(err)
					return
				}
			}
			if info.WorkingDirectory != "" {
				workingDirectory = info.WorkingDirectory
			}
			if info.DockerComposeDefinitionFromConfigPath != "" && workingDirectory != "" {
				newCommands[fmt.Sprintf("stop all containers of %s", dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(workingDirectory, "docker-compose stop", w, forceStop)
				}
				newCommands[fmt.Sprintf("view status of all containers of %s", dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(workingDirectory, "docker-compose ps", w, forceStop)
				}
				newCommands[fmt.Sprintf("start (create) all containers of %s", dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(workingDirectory, "docker-compose up", w, forceStop)
				}
			}
			if workingDirectory != "" && dockerComposeConfigName != "" && info.DockerComposeDefinition != nil {
				newCommands[fmt.Sprintf("sync docker-compose.yml of %s", dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
					data, err := yaml.Marshal(info.DockerComposeDefinition)
					if err != nil {
						return err
					}
					return ioutil.WriteFile(workingDirectory + "/docker-compose.yml", data, 0777)
				}
			}
			for serviceName := range definition.Services {
				func(serviceName string) {
					if workingDirectory != "" {
						newCommands[fmt.Sprintf("view logs container %s of %s", serviceName, dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
							return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose logs --tail 10000 -f %s",serviceName), w, forceStop)
						}
						newCommands[fmt.Sprintf("start (create) container %s of %s", serviceName, dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
							return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose up %s",serviceName), w, forceStop)
						}
						newCommands[fmt.Sprintf("recreate container %s of %s", serviceName, dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
							err := common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose stop %s",serviceName), w, forceStop)
							if err != nil {
								return err
							}
							err = common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose rm -f %s",serviceName), w, forceStop)
							if err != nil {
								return err
							}
							return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose up %s",serviceName), w, forceStop)
						}
						newCommands[fmt.Sprintf("stop container %s of %s", serviceName, dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
							return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose stop %s",serviceName), w, forceStop)
						}
						newCommands[fmt.Sprintf("restart container %s of %s", serviceName, dockerComposeConfigName)] = func(w common.IWriter, param string, forceStop chan bool) error {
							return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("docker-compose restart %s",serviceName), w, forceStop)
						}
					}
				}(serviceName)
			}
		}(k)
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromDocker(newCommands map[string]common.CommandHandler) error {
	type YamlDocker struct {
		ContainerName string `yaml:"container name"`
		FromGitRepo string `yaml:"from git repo"`
		FromGitBranch string `yaml:"from git branch"`
		AdditionalCommands map[string]string `yaml:"additional commands"`
		CreateContainerFromDockerRunCommand string `yaml:"create container from docker run command"`
		RemoteAccessUsingSshConfigFor string `yaml:"remote access using ssh config for"`
		SupportMySqlDatabases []string `yaml:"support mysql databases"`
		SupportPhp bool `yaml:"support php"`
		WorkingDirectory string `yaml:"working directory"`
	}
	data, err := ioutil.ReadFile("config/docker.yml")
	if err != nil {
		return err
	}
	out := map[string]YamlDocker{}
	err = yaml.Unmarshal(data, out)
	if err != nil {
		return err
	}
	for k := range out {
		func(k string) {
			info := out[k]
			// get container name
			containerName := ""
			if info.ContainerName != "" {
				containerName = info.ContainerName
			}
			if info.FromGitRepo != "" {
				newCommands[fmt.Sprintf("clone source for %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("cd tmps/repos && git clone %s", info.FromGitRepo), w, forceStop)
				}
			}
			if info.CreateContainerFromDockerRunCommand != "" && info.WorkingDirectory != "" {
				newCommands[fmt.Sprintf("create container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(info.WorkingDirectory, info.CreateContainerFromDockerRunCommand, w, forceStop)
				}
			}
			if info.CreateContainerFromDockerRunCommand != "" && info.ContainerName != "" {
				newCommands[fmt.Sprintf("create container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(info.WorkingDirectory, info.CreateContainerFromDockerRunCommand, w, forceStop)
				}
				newCommands[fmt.Sprintf("recreate container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					_ = common.RunLinuxCommand(fmt.Sprintf("docker stop %s", info.ContainerName), w, forceStop)
					_ = common.RunLinuxCommand(fmt.Sprintf("docker rm -f %s", info.ContainerName), w, forceStop)
					return common.RunLinuxCommandWithDirectory(info.WorkingDirectory, info.CreateContainerFromDockerRunCommand, w, forceStop)
				}
			}
			if containerName != "" {
				newCommands[fmt.Sprintf("start container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					err := common.RunLinuxCommand(fmt.Sprintf("docker start %s", containerName), w, forceStop)
					if err != nil {
						return err
					}
					return common.RunLinuxCommand("docker container ps", w, forceStop)
				}
				newCommands[fmt.Sprintf("inspect container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("docker inspect %s", containerName), w, forceStop)
				}
				newCommands[fmt.Sprintf("stop container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("docker stop %s", containerName), w, forceStop)
				}
				newCommands[fmt.Sprintf("restart container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("docker restart %s", containerName), w, forceStop)
				}
				newCommands[fmt.Sprintf("view logs container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("docker logs --tail 10000 -f %s", containerName), w, forceStop)
				}
				newCommands[fmt.Sprintf("remove container %s", k)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommand(fmt.Sprintf("docker stop %s && docker rm %s", containerName, containerName), w, forceStop)
				}
			}
		}(k)
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromGitRepos(newCommands map[string]common.CommandHandler) error {
	type Item struct {
		Repo string `yaml:"repo"`
		WorkingDirectoryFromConfig string `yaml:"working directory from config"`
		WorkingDirectory string `yaml:"working directory"`
		Branch string `yaml:"branch"`
	}
	data, err := ioutil.ReadFile("config/git-repo.yml")
	if err != nil {
		return err
	}
	items := map[string]Item{}
	err = yaml.Unmarshal(data, items)
	if err != nil {
		return err
	}
	for name := range items {
		func(name string) {
			item := items[name]
			workingDirectory := item.WorkingDirectory
			if item.Repo != "" && item.WorkingDirectoryFromConfig != "" {
				var err error
				workingDirectory, err = this.config.GetStringByKey(item.WorkingDirectoryFromConfig)
				common.PanicOnError(err)
			}
			if item.Repo != "" && workingDirectory != "" {
				newCommands[fmt.Sprintf("git clone %s", name)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("git clone %s", item.Repo), w, forceStop)
				}
			}
			if item.Repo != "" && workingDirectory != "" && item.Branch != "" {
				pieces := strings.Split(item.Repo, "/")
				pieces = strings.Split(pieces[1], ".")
				repo := pieces[0]
				newCommands[fmt.Sprintf("git pull latest code for %s", name)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return common.RunLinuxCommandWithDirectory(workingDirectory, fmt.Sprintf("cd %s && git checkout %s && git pull", repo, item.Branch), w, forceStop)
				}
			}
		}(name)
	}
	return nil
}

func (this *CommandCenter) reloadCommandsFromMysql(newCommands map[string]common.CommandHandler) error {
	data, err := ioutil.ReadFile("config/mysql.yml")
	if err != nil {
		return err
	}
	items := map[string]yaml_config.MysqlItem{}
	err = yaml.Unmarshal(data, items)
	if err != nil {
		return err
	}
	for name := range items {
		func(name string) {
			item := items[name]
			if item.CanExport() {
				newCommands[fmt.Sprintf("export database %s", name)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return item.Export(func(key string) (item *yaml_config.SshItem, e error) {
						return GetSshItemByKey(key)
					},w, forceStop)
				}
			}
			if item.CanImport() {
				newCommands[fmt.Sprintf("import database %s", name)] = func(w common.IWriter, param string, forceStop chan bool) error {
					return item.Import(w, forceStop)
				}
			}
			newCommands[fmt.Sprintf("view tables of %s", name)] = func(w common.IWriter, param string, forceStop chan bool) error {
				return item.RunSql(func(key string) (item *yaml_config.SshItem, e error) {
					return GetSshItemByKey(key)
				},"SHOW TABLES;", w, forceStop)
			}
		}(name)
	}
	return nil
}

func (this *CommandCenter) Reload() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	this.time = time.Now().Unix()
	newCommands := map[string]common.CommandHandler{}
	common.PanicOnError(this.reloadCommandsFromCurl(newCommands))
	common.PanicOnError(this.reloadCommandsFromIntegrationTest(newCommands))
	common.PanicOnError(this.reloadCommandsFromFormulaConfig(newCommands))
	common.PanicOnError(this.reloadCommandsFromCodeFiles(newCommands))
	common.PanicOnError(this.reloadCommandsFromDocker(newCommands))
	common.PanicOnError(this.reloadCommandsFromDockerCompose(newCommands))
	common.PanicOnError(this.reloadCommandsFromGitRepos(newCommands))
	common.PanicOnError(this.reloadCommandsFromMysql(newCommands))
	this.commands = newCommands
	return nil
}

func (this *CommandCenter) GetCommandNames() ([]string, error) {
	res := make([]string, 0, len(this.commands))
	for k := range this.commands {
		res = append(res, k)
	}
	return res, nil
}

func (this *CommandCenter) GetCommandInfo(commandName string) (common.CommandHandler, error) {
	var val common.CommandHandler
	ok := true
	if val, ok = this.commands[commandName]; !ok {
		return nil, fmt.Errorf("key %s not exists", commandName)
	}
	return val, nil
}