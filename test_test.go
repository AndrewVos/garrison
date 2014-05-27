package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var currentContainerPort = 45000

type dockerContainer struct {
	id   string
	ip   string
	port int
}

func NewDockerContainer() dockerContainer {
	port := currentContainerPort
	currentContainerPort += 1
	cmd := exec.Command("docker", "run", "-d", "-p", fmt.Sprintf("%v:22", port), "garrison/test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}
	id := strings.Replace(string(out), "\n", "", -1)

	cmd = exec.Command("docker", "inspect", "--format", "{{ .NetworkSettings.IPAddress }}", id)
	out, err = cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	ip := strings.Replace(string(out), "\n", "", -1)
	return dockerContainer{id: id, ip: ip, port: port}
}

func (c dockerContainer) Kill() {
	cmd := exec.Command("docker", "kill", c.id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		panic(err)
	}
}

func buildTestDockerContainer() {
	dockerFile := `
FROM ubuntu

RUN echo "deb http://archive.ubuntu.com/ubuntu precise main universe" > /etc/apt/sources.list
RUN apt-get update

RUN apt-get install -y openssh-server
RUN mkdir /var/run/sshd

RUN passwd -d root
RUN echo "PermitEmptyPasswords yes" > /etc/ssh/sshd_config

CMD /usr/sbin/sshd -D`

	file, err := os.Create("Dockerfile")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(dockerFile))
	cmd := exec.Command("docker", "build", "-t", "garrison/test", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

func captureStdout(f func()) string {
	tempFile, _ := ioutil.TempFile("", "stdout")
	oldStdout := os.Stdout
	os.Stdout = tempFile
	f()
	os.Stdout = oldStdout
	tempFile.Close()
	b, _ := ioutil.ReadFile(tempFile.Name())
	return string(b)
}
