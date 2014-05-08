package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type ServerConfiguration struct {
	Name    string   `json:"name"`
	Tasks   []Task   `json:"tasks"`
	Servers []string `json:"servers"`
}

type Task struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

func main() {
	b, err := ioutil.ReadFile("deployment-configuration.json")
	if err != nil {
		fmt.Println("I couldn't read your deployment-configuration.json. Are you sure it exists?")
		panic(err)
	}

	var serverConfigurations []ServerConfiguration
	err = json.Unmarshal(b, &serverConfigurations)
	if err != nil {
		fmt.Println("I couldn't decode your deployment-configuration.json.")
		panic(err)
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
					fmt.Printf("Executing %q on %q\n", task.Script, server)
				}
				return
			}
		}
	}
	fmt.Printf("I couldn't find the command %q\n", command)
}
