package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func init() {
	buildTestDockerContainer()
}

func cleanup() {
	os.Remove("garrison.json")
	os.Remove("Dockerfile")
}

func createGarrisonFile(contents string) {
	file, err := os.Create("garrison.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(contents))
}

func createBuildScript(contents string) string {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	file.Write([]byte(contents))
	return file.Name()
}

func TestRunsTasksOnAllServers(t *testing.T) {
	defer cleanup()
	container1 := NewDockerContainer()
	container2 := NewDockerContainer()
	defer container1.Kill()
	defer container2.Kill()

	script := createBuildScript("echo hostname:`hostname`")
	createGarrisonFile(`[{
		"name": "server1",
		"tasks": [{"name": "task1", "script": "` + script + `"}],
		"servers": [
		{"user": "root", "address": "127.0.0.1", "port": ` + strconv.Itoa(container1.port) + `},
		{"user": "root", "address": "127.0.0.1", "port": ` + strconv.Itoa(container2.port) + `}
		]
	}]`)
	output := captureStdout(func() {
		os.Args = []string{"garrison", "server1:task1"}
		garrison()
	})
	matcher := regexp.MustCompile("hostname:.*")
	matches := matcher.FindAllString(output, -1)
	if len(matches) != 2 {
		t.Errorf("This script should have run on two servers.\n%v\n", matches)
	}
}

func TestCrashesWhenParallelTaskFails(t *testing.T) {
	defer cleanup()
	container1 := NewDockerContainer()
	container2 := NewDockerContainer()
	defer container1.Kill()
	defer container2.Kill()

	script := createBuildScript("this-should-fail")
	createGarrisonFile(`[{
		"name": "server1",
		"tasks": [{"name": "task1", "script": "` + script + `", "parallel": true}],
		"servers": [
		{"user": "root", "address": "127.0.0.1", "port": ` + strconv.Itoa(container1.port) + `},
		{"user": "root", "address": "127.0.0.1", "port": ` + strconv.Itoa(container2.port) + `}
		]
	}]`)
	os.Args = []string{"garrison", "server1:task1"}
	var errors []error
	captureStdout(func() {
		errors = garrison()
	})
	if len(errors) != 2 {
		t.Error("Expected some errors")
	}
}
