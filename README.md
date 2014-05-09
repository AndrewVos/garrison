# garrison

Garrison is a simple tool I use for deploying docker containers to large amounts
of servers, mostly on EC2.

The idea behind it is that most people just need a few bash scripts to be run on
a lot of servers, and don't want to mess around with larger scale Cloud Auto Scaling solutions.

# Installation

	go get github.com/AndrewVos/garrison

# Usage

Typically, you would create a ```garrison.json``` file in your project root.
This describes what your infrastructure looks like.
You have configuration for each server, and tasks that can be run on each server.

# Example with Elasticsearch

Lets say we want to deploy ```andrewvos/docker-elasticsearch```. We need a task
to build (docker pull) the elasticsearch image on each server, and we need a task
to launch the container on each server.

This is what our ```garrison.json``` file looks like:

	[
		{
			"name": "portiere",
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

Notice that ```parallel``` is specified for the ```build``` task. This is because we want this task to run
as fast as possible. Garrison will execute the build script on all servers at the same time.
For the ```launch``` task we want to be sure that at least one elasticsearch server is up at a time, so we
don't execute this in parallel.

And we have a bash script for ```build```:

	#!/bin/bash -e
	docker pull andrewvos/docker-elasticsearch

And one for ```launch```:

	#!/bin/bash
	export CONTAINER=elasticsearch
	docker stop $CONTAINER || :
	docker rm $CONTAINER || :
	docker run -d --name $CONTAINER -p 9200:9200 -p 9300:9300 -v /some/path:/var/lib/elasticsearch andrewvos/docker-elasticsearch
