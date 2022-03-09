# MyDynDNS

A client for the mydyndns dynamic DNS service, providing an application library, CLI tools, and agent process for
keeping DNS entries up-to-date.


## Features


### Command Line Interface

An extensive CLI exists for interacting with the MyDynDNS API:


#### Example Usage

```cli
# Show the current public IP for this host:
$ mydyndns api my-ip --config-file mydyndns.toml
1.2.3.4

# Request an update to the DNS alias for the dynamic DNS host:
$ mydyndns api update-alias --config-file mydyndns.toml
1.2.3.4
```


#### Configuration

The CLI contains built-in support for generating and validating config files:

```cli
# Generate mydyndns.toml with validated custom options:
$ mydyndns config write toml \
    --validate \
    --api-url=https://example.com --api-key=secret \
    --interval=1h --log-verbosity=1

# Generate mydyndns.json populated default values:
$ mydyndns config write json --defaults

# Generate /var/mydyndns/conf.yml populated with default values:
$ mydyndns config write /var/mydyndns/conf.yml --defaults
$ mydyndns config write conf.yml --directory /var/mydyndns --defaults
```

##### Configuration sources

In addition to configuration files, environment variables may be provided in place of file-based 
configuration directives, as well as command-line flags. Environment variables must be prefixed with `MYDYNDNS_`, 
be upper-cased, and use underscores instead of dashes. 

As an example, the following directives are equivalent means of providing a base URL for a remote MyDynDNS
service API:
1. As a command-line flag: `--api-url=https://example.com`
2. As an environment variable: `MYDYNDNS_API_URL=https://example.com`
3. As a line in a (`toml`) configuration file: `api-url = https://example.com`

Note that the above list is in order of precedence, e.g. a configuration directive provided as a command-line flag
will take precedence over a conflicting environment variable, etc.


##### Notes:

- By default, the CLI looks for a configuration file called `mydyndns.ext` in the current working directory,
where `.ext` is any supported config file extension. The source directory and/or filename can be customized
by providing the `--config-path` and/or `--config-file` CLI flags, respectively.
- Configuration files generated with the `--defaults` CLI flag are not inherently valid and require customizations
before they may be used successfully.
- See `mydyndns help config` for more information.


#### Completion

The CLI fully supports tab completion through [Cobra](https://github.com/spf13/cobra).
You can run `mydyndns help completion` for instructions on how to generate and enable completions.


#### Additional Help

Help for every command is available by appending the `-h / --help` flag to any command.


#### CLI Agent Process

The CLI provides support for a long-running (daemon) process that can be used as a local service for monitoring
changes to the host's (dynamic) IP address and requesting the MyDynDNS service to update DNS records accordingly.

To run in a terminal:
```cli
# Assume a valid and discoverable config file exists in the current working directory 
$ mydyndns agent start
level=info msg="Initialized with IP address after DNS update" ip=1.2.3.4 ts=2022-01-02T15:04:05.552333-07:00
^Clevel=warn msg="Agent stopped" ts=2022-01-02T15:06:07.744942-07:00
task: signal received: interrupt
```


##### Notes:
- The amount of information logged by the agent can be controlled via the `-v / --log-verbosity` flag
or by adjusting the `log-verbosity` config file directive.
- The `SIGINT` signal ([`ctrl-c`](https://en.wikipedia.org/wiki/Control-C)) requests a graceful shutdown 
of the agent process.


### Client SDK

A package is made available for interacting with the API in other applications:

```go
package main

import (
	"fmt"
	"github.com/TylerHendrickson/mydyndns/pkg/sdk"
	"net"
	"os"
)

func main() {
	var currentIP net.IP

	c := sdk.NewClient("https://example.com/mydyndns-service", os.Getenv("MYDYNDNS_API_KEY"))
	fmt.Println("Fetching my IP address...")
	if ip, err := c.MyIP(); err != nil {
		panic(err)
	} else {
		fmt.Println("My IP address is:", ip)
		currentIP = ip
	}

	if ips, err := net.LookupIP(os.Getenv("MYDYNDNS_MANAGED_HOSTNAME")); err != nil {
		panic(err)
	} else if len(ips) != 1 {
		panic("there should be exactly 1 aliased IP address!")
	} else {
		aIP := ips[0]
		if !aIP.Equal(currentIP) {
			fmt.Println("Requesting DNS update...")
			if ip, err := c.UpdateAlias(); err != nil {
				panic(fmt.Errorf("error requesting DNS update: %w", err))
			} else {
				fmt.Println("DNS alias will point to:", ip)
			}
		}
	}
}
```

### Agent Library

The Agent behavior is available as an importable package that can be configured and executed from other libraries 
and applications.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TylerHendrickson/mydyndns/pkg/agent"
	"github.com/TylerHendrickson/mydyndns/pkg/sdk"
	"github.com/go-kit/log"
)

func main() {
	c := sdk.NewClient("https://example.com/mydyndns-service", os.Getenv("MYDYNDNS_API_KEY"))
	logger := log.NewJSONLogger(os.Stderr)

	// SIGINT cancels context
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, os.Interrupt)
	defer cancel()

	// Start an agent that checks syncs DNS with the IP every 1 hour 
	// until CTRL+C sends SIGINT for graceful shutdown.
	// Note: this function call is safe for concurrent use and may be wrapped in a goroutine.
	err := agent.Run(ctx, logger, c, time.Hour)
	if err != nil {
		fmt.Printf("Failed to run agent: %s\n", err)
	}
}
```


## Using


### go install

The easiest way to install this program is to clone the repository, navigate to `mydyndns`, 
and run `go install ./cmd/mydyndns` from within the cloned repository. 
This will compile the proper binary for your operating system and place it in your Go binaries path 
(e.g. `$GOBIN`, `$GOPATH/bin`, etc.).

Alternatively, run `go install github.com/TylerHendrickson/mydyndns/cmd/mydyndns@latest` to install without
cloning.


### go get

Use `go get -u github.com/TylerHendrickson/mydyndns/pkg/sdk` 
and/or `go get -u github.com/TylerHendrickson/mydyndns/pkg/agent`
to make this available for use in your own Go application.


### Precompiled Binary

See the [releases page](https://github.com/TylerHendrickson/mydyndns/releases) to download the appropriate archive 
for your platform. You can find the latest release [here](https://github.com/TylerHendrickson/mydyndns/releases/latest).

Example:

```cli
$ wget https://github.com/TylerHendrickson/mydyndns/releases/download/0.1.8/mydyndns_0.1.8_Linux_arm64.tar.gz
$ tar xvfz mydyndns_*.tar.gz mydyndns
$ mv ./mydyndns /usr/bin/mydyndns
```

### Docker

The CLI application is available as a Docker image, hosted by GitHub's container registry. 
View released images [here](https://github.com/TylerHendrickson/mydyndns/pkgs/container/mydyndns).


#### Using docker-compose

Using `docker-compose` to run the agent is relatively straightforward:

```yaml
version: "3.6"
services:
  mydyndns:
    image: ghcr.io/tylerhendrickson/mydyndns:latest
    container_name: mydyndns-agent
    command: agent start
    environment:
      - TZ
      - PUID
      - PGID
      - MYDYNDNS_CONFIG_PATH=/config
    volumes:
      - ./mydyndns:/config:ro
    restart: unless-stopped
```

In this example, configuration is sourced from file within a volume mounted at `/config` in the container, 
e.g. `/config/mydyndns.toml`. However, all usual means of configuration are still supported; 
parameters may be provided via the `environment` section, as well as by passing additional flags 
in the `command` directive. Note that all environment variables must be prefixed with `MYDYNDNS_`.


## Developing

This project uses [`Taskfile.yml`](https://taskfile.dev/) to manage common development tasks (instead of using `make`).
[Install Task](https://taskfile.dev/#/installation), then run `task --list` to see a list of available tasks
for this project, or `task --help` for more information about using this system.
