package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/AndrewVos/colour"
)

type Server struct {
	User    string `json:"user"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type ServerConfiguration struct {
	Name    string   `json:"name"`
	Tasks   []Task   `json:"tasks"`
	Servers []Server `json:"servers"`
}

type Task struct {
	Name        string            `json:"name"`
	Script      string            `json:"script"`
	Parallel    bool              `json:"parallel"`
	Environment map[string]string `json:"environment"`
}

func main() {
	garrison()
}

func garrison() {
	fileName := "garrison.json"
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		FatalRedf("I couldn't read your %v. Are you sure it exists?\n%v\n", fileName, err)
	}

	var serverConfigurations []ServerConfiguration
	err = json.Unmarshal(b, &serverConfigurations)
	if err != nil {
		FatalRedf("I couldn't decode your %v\n%v\n", fileName, err)
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "--completion-help" {
			printCompletionCommands(serverConfigurations)
			return
		}
		command := os.Args[1]
		executeCommand(command, serverConfigurations)
	} else {
		printCommands(serverConfigurations)
	}
}

func printCommands(serverConfigurations []ServerConfiguration) {
	fmt.Printf("Usage: %v <command>\n\n", os.Args[0])
	fmt.Println("Commands:")
	for _, serverConfiguration := range serverConfigurations {
		var addresses []string
		for i, s := range serverConfiguration.Servers {
			addresses = append(addresses, fmt.Sprintf("%d: %v", i, s.Address))
		}
		fmt.Printf("%v: {%v}\n", serverConfiguration.Name, strings.Join(addresses, ", "))

		for _, task := range serverConfiguration.Tasks {
			fmt.Printf("  %v:[index|address:]%v\n", serverConfiguration.Name, task.Name)
		}

	}
}

func printCompletionCommands(serverConfigurations []ServerConfiguration) {
	for _, serverConfiguration := range serverConfigurations {
		for _, task := range serverConfiguration.Tasks {
			fmt.Printf("%v:%v\n", serverConfiguration.Name, task.Name)
			for i, _ := range serverConfiguration.Servers {
				fmt.Printf("%v:%v:%v\n", serverConfiguration.Name, i, task.Name)
			}
			for _, server := range serverConfiguration.Servers {
				fmt.Printf("%v:%v:%v\n", serverConfiguration.Name, server.Address, task.Name)
			}
		}
	}
}

func executeCommand(command string, serverConfigurations []ServerConfiguration) {
	for _, serverConfiguration := range serverConfigurations {
		for _, task := range serverConfiguration.Tasks {
			if fmt.Sprintf("%v:%v", serverConfiguration.Name, task.Name) == command {
				var wg sync.WaitGroup
				for _, server := range serverConfiguration.Servers {
					fmt.Printf(colour.Blue("Executing %q on %q\n"), task.Script, server.Address)
					if task.Parallel && len(serverConfiguration.Servers) > 1 {
						wg.Add(1)
						out := &DelayedStdWriter{Out: os.Stdout}
						go func(server Server, task Task, out *DelayedStdWriter) {
							err := executeTask(server, task, out)
							wg.Done()
							out.Flush()
							if err != nil {
								Redf("%v\n", err)
							}
						}(server, task, out)

					} else {
						err := executeTask(server, task, os.Stdout)
						if err != nil {
							FatalRedf("%v\n", err)
						}
					}
				}
				wg.Wait()
				return
			}

			for i, server := range serverConfiguration.Servers {
				commandWithServer := fmt.Sprintf("%v:%v:%v", serverConfiguration.Name, server.Address, task.Name)
				commandWithIndex := fmt.Sprintf("%v:%v:%v", serverConfiguration.Name, i, task.Name)
				if command == commandWithServer || command == commandWithIndex {
					fmt.Printf(colour.Blue("Executing %q on %q\n"), task.Script, server.Address)
					err := executeTask(server, task, os.Stdout)
					if err != nil {
						FatalRedf("%v\n", err)
					}
					return
				}
			}
		}
	}
	FatalRedf("I couldn't find the command %q\n", command)
}

func executeTask(server Server, task Task, out io.Writer) error {
	script, err := ioutil.ReadFile(task.Script)
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't read from your script %q:\n%v", task.Script, err))
	}

	port := "22"
	if server.Port != 0 {
		port = strconv.Itoa(server.Port)
	}
	cmd := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=no", "-p", port, fmt.Sprintf("%v@%v", server.User, server.Address))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't connect to stdin of ssh:\n%v", err))
	}
	for name, value := range task.Environment {
		fmt.Fprintf(stdin, "export %v=\"%v\"\n", name, value)
	}
	stdin.Write(script)
	stdin.Close()

	cmd.Stdout = out
	cmd.Stderr = out

	err = cmd.Start()
	if err != nil {
		return errors.New(fmt.Sprintf("I couldn't launch ssh:\n%v", err))
	}

	err = cmd.Wait()
	if err != nil {
		return errors.New(fmt.Sprintf("I had some problems running %q on %q:\n%v", task.Script, server.Address, err))
	}
	return nil
}

func Redf(format string, v ...interface{}) {
	fmt.Printf(colour.Red(format), v...)
}

func FatalRedf(format string, v ...interface{}) {
	fmt.Printf(colour.Red(format), v...)
	os.Exit(1)
}

func Bluef(format string, v ...interface{}) {
	fmt.Printf(colour.Blue(format), v...)
}

type DelayedStdWriter struct {
	Out    io.Writer
	buffer []byte
}

func (s *DelayedStdWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		s.buffer = append(s.buffer, b)
	}
	return len(p), nil
}

func (s *DelayedStdWriter) Flush() {
	s.Out.Write(s.buffer)
}
