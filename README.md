# garrison

Garrison is a simple tool I use for deploying docker containers to large amounts
of servers, mostly on EC2.

The idea behind it is that most people just need a few bash scripts to be run on
a lot of servers, and don't want to mess around with larger scale "Cloud" solutions.

If you're using auto-scaling or something like that, then this probably isn't for you.
Garrison uses a push model, where you probably need a pull model. Consider just setting up
your build scripts as "User Data"?

# Requirements

* golang
* ssh

# Installation

	go get github.com/AndrewVos/garrison

# Usage

Typically, you would create a ```garrison.json``` file in your project root.
This describes what your infrastructure looks like.
You have configuration for each server, and tasks that can be run on each server.

# Tasks
Tasks can only be scripts at the moment. They can be run in parallel, and can have
environment variables.

When a set of parallel tasks are run, you won't see any output
until the first tasks complete. Output from tasks will only show when each task completes.

	{
		"name": "your-task-name",
		"script": "/some-local-file",
		"parallel": false,
		"environment": {
			"ENV_VAR1": "value1",
			"ENV_VAR2": "value2"
		}
	}

## Task parameters

Sometimes you might need to pass an environment variable through to some task and you don't want
to have to change your configuration every time. For example, you may be telling your build script
which git branch you want to build.

You can specify parameters in your task like this:

```json
{
  "name": "build",
  "parameters": ["BRANCH"],
  "script": "build"
}
```

And then execute garrison as usual and it will use the `BRANCH` environment variable:

```
BRANCH=some-branch garrison group:task
```

# Example with Elasticsearch

Lets say we want to deploy ```andrewvos/docker-elasticsearch```. We need a task
to build (docker pull) the elasticsearch image on each server, and we need a task
to launch the container on each server.

This is what our ```garrison.json``` file looks like:

```json
[
	{
		"name": "elastic",
		"servers": [
			{ "address": "11.11.11.11", "user": "ubuntu" },
			{ "address": "22.22.22.22", "user": "ubuntu" }
		],
		"tasks": [
			{ "name": "build", "script": "deployment-scripts/build", "parallel": true },
			{ "name": "launch", "script": "deployment-scripts/launch" }
		]
	}
]
```

Notice that ```parallel``` is specified for the ```build``` task. This is because we want this task to run
as fast as possible. Garrison will execute the build script on all servers at the same time.
For the ```launch``` task we want to be sure that at least one elasticsearch server is up at a time, so we
don't execute this in parallel.

And we have a bash script for ```build```:

```bash
#!/bin/bash -e
docker pull andrewvos/docker-elasticsearch
```

And one for ```launch```:

```bash
#!/bin/bash
export CONTAINER=elasticsearch
docker stop $CONTAINER || :
docker rm $CONTAINER || :
docker run -d --name $CONTAINER -p 9200:9200 -p 9300:9300 -v /some/path:/var/lib/elasticsearch andrewvos/docker-elasticsearch
```

To launch the ```build``` task just run ```garrison elastic:build```. Note that you can also single out a specific server by using
the address or the index in the task name:

```bash
garrison elastic:0:build
garrison elastic:11.11.11.11:build
```

## ZSH Completion

Add the following to your ```.zprofile```.

    fpath=($GOPATH/src/github.com/AndrewVos/garrison/zsh-completion $fpath)
