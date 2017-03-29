package main

import (
	"log"
	"os/exec"
	"syscall"
)

func main() {
	cmd := exec.Command("./bin/imagenie")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}
