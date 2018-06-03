package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	logfile = "./output.log"
)

var (
	log = logrus.New()
)

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
		os.Exit(0)
	}

	log.Formatter = new(logrus.JSONFormatter)
	log.Formatter = new(logrus.TextFormatter)
	log.Formatter.(*logrus.TextFormatter).DisableTimestamp = true
	log.Level = logrus.InfoLevel

	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
}

func main() {
	address := getHost(viper.Get("host"))

	log.WithFields(logrus.Fields{
		"address": address,
	}).Debug("Func getHost")

	user, password := getUser(viper.Get("user"))

	log.WithFields(logrus.Fields{
		"user":     user,
		"password": password,
	}).Debug("Func getUser")

	if len(password) == 0 {
		password = getPassword()

		log.WithFields(logrus.Fields{
			"password": password,
		}).Debug("Func getPassword")
	}

	var command string

	command = "cmdkey /generic:TERMSRV/" + address + " /user:" + user + " /pass:" + password
	execCommand(command)
	time.Sleep(2 * time.Second)

	command = "start mstsc /f /v:" + address
	execCommand(command)
	time.Sleep(3 * time.Second)

	command = "cmdkey /delete:TERMSRV/" + address
	execCommand(command)
}

// Host connects to server
type Host struct {
	Name    string
	Type    string
	Address string
}

// Hosts represents connecting server list
type Hosts []Host

func getHost(hostList interface{}) (address string) {
	hostListSlice, ok := hostList.([]interface{})
	if !ok {
		log.WithFields(logrus.Fields{}).Error("Argument is not a slice")
		os.Exit(0)
	}

	var hosts Hosts
	for _, v := range hostListSlice {
		hosts = append(hosts, Host{
			Name:    v.(map[interface{}]interface{})["Name"].(string),
			Type:    v.(map[interface{}]interface{})["Type"].(string),
			Address: v.(map[interface{}]interface{})["Address"].(string),
		})
	}

	templates := &promptui.SelectTemplates{
		Label:    `{{ "?" | blue }} {{ . }} - DOWN:j UP:k`,
		Active:   `▸ {{ .Name | cyan | underline }} ({{ .Address | green}})`,
		Inactive: "  {{ .Name | cyan }} ({{ .Address | green }})",
		Selected: `{{ "✔" | green }} {{ .Name | bold }}`,
		Details: `
----------- Host -----------
{{ "Name:" | faint }}	{{ .Name }}
{{ "Type:" | faint }}	{{ .Type }}
{{ "Address:" | faint }}	{{ .Address }}`,
	}

	searcher := func(input string, index int) bool {
		host := hosts[index]
		name := strings.Replace(strings.ToLower(host.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     "Host",
		Items:     hosts,
		Templates: templates,
		Size:      4,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()

	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Prompt failed")
		os.Exit(0)
	}

	log.WithFields(logrus.Fields{
		"number": i + 1,
		"name":   hosts[i].Name,
	}).Debug("You choose")

	address = hosts[i].Address
	return
}

func getUser(userList interface{}) (user string, password string) {
	userListSlice, ok := userList.([]interface{})
	if !ok {
		log.WithFields(logrus.Fields{}).Error("Argument is not a slice")
		os.Exit(0)
	}

	var users []string
	var login string
	for _, v := range userListSlice {
		if v.(map[interface{}]interface{})["Username"].(string) == "USERNAME" {
			login = v.(map[interface{}]interface{})["Domain"].(string) + "\\" + os.Getenv("USERNAME")
		} else {
			login = v.(map[interface{}]interface{})["Domain"].(string) + "\\" + v.(map[interface{}]interface{})["Username"].(string)
		}
		users = append(users, login)
	}

	prompt := promptui.SelectWithAdd{
		Label:    "User",
		Items:    users,
		AddLabel: "Other",
	}

	_, user, err := prompt.Run()

	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Prompt failed")
		os.Exit(0)
	}

	for _, v := range userListSlice {
		if v.(map[interface{}]interface{})["Username"].(string) == "USERNAME" {
			login = v.(map[interface{}]interface{})["Domain"].(string) + "\\" + os.Getenv("USERNAME")
		} else {
			login = v.(map[interface{}]interface{})["Domain"].(string) + "\\" + v.(map[interface{}]interface{})["Username"].(string)
		}
		if login == user {
			password = v.(map[interface{}]interface{})["Password"].(string)
			if password == "NA" {
				password = ""
			}
		}
	}

	return
}

func getPassword() (password string) {
	validate := func(input string) error {
		if len(input) < 6 {
			return errors.New("Password must have more than 6 characters")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Password",
		Validate: validate,
		Mask:     '*',
	}

	password, err := prompt.Run()

	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("Prompt failed")
		os.Exit(0)
	}

	return
}

func execCommand(command string) {
	log.WithFields(logrus.Fields{
		"command": command,
	}).Debug("Func execCommand")

	err := exec.Command("cmd", "/c", command).Run()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Warn("Command Exec Error")
	}
}
