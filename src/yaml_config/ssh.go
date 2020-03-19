package yaml_config

import (
	"common"
	"fmt"
)

type SshItem struct {
	Host string `yaml:"host"`
	User string `yaml:"user"`
	Port string `yaml:"port"`
	WorkingDirectory string `yaml:"working directory"`
}

func (this *SshItem) Exec(command string, writer common.IWriter, forceStop chan bool) error {
	return common.RunLinuxCommand(fmt.Sprintf("ssh %s@%s -p %s \"%s\"",
		this.User,
		this.Host,
		this.Port,
		command,
		), writer, forceStop)
}

func (this *SshItem) CopyFromRemoteToLocal(localFileNameToBeSaved string, writer common.IWriter, forceStop chan bool) error {
	return common.RunLinuxCommand(fmt.Sprintf("scp -P %s %s@%s:db.sql data/%s.sql",
		this.Port,
		this.User,
		this.Host,
		localFileNameToBeSaved,
	), writer, forceStop)
}
