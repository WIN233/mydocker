package main

import (
	log "github.com/sirupsen/logrus"
	"mydocker/assets"
	"os/user"
)

func main() {
	homeDir := GetHomeDir()
	err := assets.ExtractBusybox(homeDir + "/h")
	if err != nil {
		log.Errorf(err.Error())
		return
	}
}

func GetHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}
