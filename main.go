package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/AndrewVos/colour"
)

type Server struct {
	User    string `json:"user"`
	Address string `json:"address"`
}

type ServerConfiguration struct {
	Name    string   `json:"name"`
	Tasks   []Task   `json:"tasks"`
	Servers []Server `json:"servers"`
}

type Task struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

func main() {
	b, err := ioutil.ReadFile("deployment-configuration.json")
	if err != nil {
		FatalRedf("I couldn't read your deployment-configuration.json. Are you sure it exists?\n%v\n", err)
	}

	var serverConfigurations []ServerConfiguration
	err = json.Unmarshal(b, &serverConfigurations)
	if err != nil {
		FatalRedf("I couldn't decode your deployment-configuration.json\n%v\n", err)
	}

	if len(os.Args) > 1 {
		command := os.Args[1]
		executeCommand(command, serverConfigurations)
	} else {
		printCommands(serverConfigurations)
	}
}

func printCommands(serverConfigurations []ServerConfiguration) {
	fmt.Printf("Usage: deploydeploydeploy <command>\n\n")
	fmt.Printf("Commands:\n")
	for _, serverConfiguration := range serverConfigurations {
		fmt.Printf("%v:\n", serverConfiguration.Name)
		for _, task := range serverConfiguration.Tasks {
			fmt.Printf("  %v:%v\n", serverConfiguration.Name, task.Name)
		}
	}
}

func executeCommand(command string, serverConfigurations []ServerConfiguration) {
	for _, serverConfiguration := range serverConfigurations {
		for _, task := range serverConfiguration.Tasks {
			if fmt.Sprintf("%v:%v", serverConfiguration.Name, task.Name) == command {
				for _, server := range serverConfiguration.Servers {
					fmt.Printf(colour.Blue("Executing %q on %q\n"), task.Script, server.Address)
					executeTask(server, task)
				}
				return
			}
		}
	}
	FatalRedf("I couldn't find the command %q\n", command)
}

func executeTask(server Server, task Task) {
	script, err := ioutil.ReadFile(task.Script)
	if err != nil {
		FatalRedf("I couldn't read from your script %q:\n%v\n", task.Script, err)
	}

	cmd := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=no", fmt.Sprintf("%v@%v", server.User, server.Address))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		FatalRedf("I couldn't connect to stdin of ssh:\n%v\n", err)
	}
	stdin.Write(script)
	stdin.Close()

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		FatalRedf("I couldn't launch ssh:\n%v\n", err)
	}

	err = cmd.Wait()
	if err != nil {
		FatalRedf("I had some problems running %q on %q:\n%v\n", task.Script, server.Address, err)
	}
}

func FatalRedf(format string, v ...interface{}) {
	fmt.Printf(colour.Red(format), v...)
	os.Exit(1)
}

func Bluef(format string, v ...interface{}) {
	fmt.Printf(colour.Blue(format), v...)
}
