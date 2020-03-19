package yaml_config

import (
	"common"
	"database/sql"
	"fmt"
)

type MysqlItem struct {
	DatabaseName string `yaml:"database name"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	DockerContainer string `yaml:"docker container"`
	RemoteServerFromSshConfig string `yaml:"remote server from ssh config"`
}

func (this *MysqlItem) CanExport() bool {
	return this.DatabaseName != "" && this.User != "" && this.DockerContainer != ""
}

func (this *MysqlItem) CanImport() bool {
	return this.DatabaseName != "" && this.User != "" && this.DockerContainer != ""
}

func (this *MysqlItem) Export(getSshItemByKey func(key string) (*SshItem, error), w common.IWriter, forceStop chan bool) error {
	userOrEmpty := this.User
	if userOrEmpty != "" {
		userOrEmpty = "-u " + userOrEmpty
	}
	passOrEmpty := this.Pass
	if passOrEmpty != "" {
		passOrEmpty = "-p" + passOrEmpty
	}
	if this.DatabaseName != "" &&
		this.DockerContainer != "" &&
		this.RemoteServerFromSshConfig != "" {
		sshItem, err := getSshItemByKey(this.RemoteServerFromSshConfig)
		if err != nil {
			return err
		}
		err = sshItem.Exec(fmt.Sprintf("docker exec %s bash -c 'mysqldump %s %s %s > /db.sql'",
			this.DockerContainer,
			userOrEmpty,
			passOrEmpty,
			this.DatabaseName,
			), w, forceStop)
		if err != nil {
			return err
		}
		err = sshItem.Exec(fmt.Sprintf("docker cp %s:/db.sql db.sql", this.DockerContainer), w, forceStop)
		if err != nil {
			return err
		}
		err = sshItem.CopyFromRemoteToLocal(this.DatabaseName, w, forceStop)
		if err != nil {
			return err
		}
	}
	if this.DatabaseName != "" &&
		this.DockerContainer != "" &&
		this.RemoteServerFromSshConfig == "" {
		return common.RunBashScript(fmt.Sprintf(`
						#!/bin/bash
						docker exec %s bash -c 'mysqldump %s %s %s > /db.sql'
						docker cp %s:/db.sql data/%s.sql
					`,
			this.DockerContainer,
			userOrEmpty,
			passOrEmpty,
			this.DatabaseName,
			this.DockerContainer,
			this.DatabaseName), "", w, forceStop)
	}
	return fmt.Errorf("cannot export database because it does not satisfy criteria for exporting")
}

func (this *MysqlItem) GetHostOrDefault() string {
	if this.Host == "" {
		return "localhost"
	}
	return this.Host
}

func (this *MysqlItem) GetPortOrDefault() int {
	if this.Port == 0 {
		return 3306
	}
	return this.Port
}

func (this *MysqlItem) GetConnection() (*sql.DB, error) {
	return sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", this.User, this.Pass, this.GetHostOrDefault(), this.GetPortOrDefault(), this.DatabaseName))
}

func (this *MysqlItem) FetchAll(db *sql.DB, query string, args ...interface{}) ([]map[string]interface{}, error) {
	return common.FetchAll(db, query, args...)
}

func (this *MysqlItem) RunSql(getSshItemByKey func(key string) (*SshItem, error), sqlCommand string, w common.IWriter, forceStop chan bool) error {
	userOrEmpty := this.User
	if userOrEmpty != "" {
		userOrEmpty = "-u " + userOrEmpty
	}
	passOrEmpty := this.Pass
	if passOrEmpty != "" {
		passOrEmpty = "-p" + passOrEmpty
	}
	command := fmt.Sprintf("docker exec %s mysql %s %s %s -e '%s'",
		this.DockerContainer,
		userOrEmpty,
		passOrEmpty,
		this.DatabaseName,
		sqlCommand,
		)
	if this.RemoteServerFromSshConfig != "" {
		sshItem, err := getSshItemByKey(this.RemoteServerFromSshConfig)
		if err != nil {
			return err
		}
		return sshItem.Exec(command, w, forceStop)
	}
	return common.RunLinuxCommand(command, w, forceStop)
}

func (this *MysqlItem) Import(w common.IWriter, forceStop chan bool) error {
	if !this.CanImport() {
		return fmt.Errorf("cannot import database because it does not satisfy criteria for importing")
	}
	return common.RunBashScript(fmt.Sprintf(`
						#!/bin/bash
						docker cp data/%s.sql %s:/db.sql
						docker exec %s bash -c 'mysql -u %s -p%s %s < /db.sql'
					`,
		this.DatabaseName,
		this.DockerContainer,
		this.DockerContainer,
		this.User,
		this.Pass,
		this.DatabaseName), "", w, forceStop)
}