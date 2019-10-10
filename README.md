<h1 align="center">
  ztdns
</h1>

<h4 align="center">
  A DNS server for ZeroTier virtual networks.
</h4>

<p align="center">
    <img src="https://github.com/mje-nz/ztdns/workflows/Check%2C%20build/badge.svg"
         alt="Github Actions">
</p>

This is a fork of [uxbh/ztdns](https://github.com/uxbh/ztdns).



## Features

* Address members of ZeroTier networks by name
* IPv4 and IPv6 support (A and AAAA records)
* Create round-robin DNS names
* Tiny Docker image for easy setup



## Usage
First, add a new API Access Token to your [ZeroTier account](https://my.zerotier.com/).

To start the server using the Docker image:

```bash
docker run --rm \
  --volume $(pwd)/ztdns.yml:/app/ztdns.yml \
  mjenz/ztdns server --api-key API_KEY --network NETWORK_ID
```

where `API_KEY` is a ZeroTier API Access Token and `NETWORK_ID` is a ZeroTier network ID.
If your ZeroTier network interface is not called `zt0`, then you should add `--interface=<your ZeroTier interface` (or `--interface=` to operate on all interfaces).
It is recommended to use a configuration file instead of these command-line arguments (see "Configuration" section below).

To build from source:

``` bash
go get -u github.com/mje-nz/ztdns/
# or
git clone https://github.com/mje-nz/ztdns.git
cd ztdns
go install
# then
ztdns server --api-key API_KEY --network NETWORK_ID
```

If you are running on Linux, run `sudo setcap cap_net_bind_service=+ep ./ztdns` to enable non-root users to bind privileged ports.
On other operating systems, `ztdns` may need to be run as an administrator.
The Docker image should work as-is.

TODO: pre-built releases


Once the server is running you will be able to resolve ZeroTier network members by querying it directly:

```bash
dig @serveraddress member.zt
```

In order to resolve names normally, you need to get the server into the DNS lookup chain on all of your machines.
The easiest way to do this is to delegate a zone from a public domain you control.
For example, to use the name `<membername>.zt.yourdomain.com`:

* Create an `A` record `ztns.yourdomain.com` with the ZeroTier IP address of your server.
* Create an `NS` record delegating `zt.yourdomain.com` to `ztns.yourdomain.com`.

Now all your machines can resolve ZeroTier names without any extra configuration.
In fact anyone can resolve names (if they know them), but only machines on your network can actually route to them.
Note that the DNS name is based on the ZeroTier member name (as in [ZeroTier Central](https://my.zerotier.com/network)), not the hostname of the member.


TODO: split up install and usage?



## Installation as a service

### Docker
To install the server as a service using the Docker image:

```bash
docker run --detach \
  --volume $(pwd)/ztdns.yml:/app/ztdns.yml \
  --restart=unless-stopped \
  --name=ztdns mjenz/ztdns
```


### Systemd
To install the server as a `systemd` service, create a file `/etc/systemd/system/ztdns.service` containing:

```ini
[Unit]
Description=Zerotier DNS Server

[Service]
WorkingDirectory=<path containing config file>
ExecStart=ztdns server
TimeoutStopSec=10
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Make sure whatever you set `WorkingDirectory` to the directory containing your configuration file.

Then, to install the service:

```bash
# Reload service files
sudo systemctl daemon-reload
# Set ztdns service to start on boot
sudo systemctl enable ztdns.service
# Also start ztdns service right now
sudo systemctl start ztdns.service
```

To uninstall the service:

```bash
# Stop ztdns service
sudo systemctl stop ztdns.service
# Stop ztdns service from starting on boot
sudo systemctl disable ztdns.service
```



## Configuration


### Command-line options:
```bash
$ ztdns server --help
Server (ztdns server) will start the DNS server.

  Example: ztdns server

Usage:
  ztdns server [flags]

Flags:
      --api-key string     ZeroTier API key
      --api-url string     ZeroTier API URL (default "https://my.zerotier.com/api")
      --domain string      domain to serve names under (default "zt")
  -h, --help               help for server
      --include-offline    include offline members (default true)
      --interface string   network interface to bind to (default "zt0")
      --network string     ZeroTier Network ID
      --port int           port to listen on (default 53)
      --refresh int        how often to poll the ZeroTier controller in minutes (default 30)

Global Flags:
      --config string   config file (default is ztdns.yml)
      --debug           enable debug messages
```


### Config file
`ztdns` looks for `ztdns.yml` in the current working directory or `$HOME`.
There is a `ztdns.example.yml` example config file with all supported options, their description and default value:

```yaml
# This file contains all available configuration options
# with their default values.

# Network interface to bind to (or "" to bind to).  By default, only respond on
# the ZeroTier network.
interface: "zt0"

# Port to listen on.
port: 53

# Base domain.  Could be a top-level domain for internal use only (e.g., zt) or
# a domain name with one or more subdomains (e.g., internal.yourdomain.com).
# By default, map members to "<member name>.zt".
origin: "zt"

# How often to poll the ZeroTier controller in minutes.
refresh: 30

# Whether to include offline members
include-offline: true

# Enable debug messages.
debug: false

# An API key for your ZeroTier account (required).
api-key: ""

# The base API URL for the ZeroTier controller.
api-url: "https://my.zerotier.com/api"

# ID of the ZeroTier network.  Only one of "network" and "networks" can be
# specified.  E.g., if there is a network with ID "123abc" this would map
# its members to "<member name>.zt":
#   network: "123abc"
network:

# Mappings between subdomains and ZeroTier network IDs.  Only one of "network"
# and "networks" can be specified.
networks:
  # E.g., if there is a network with ID "123abc" this would map its members to
  # "<member name>.home.zt":
  #   home: "123abc"

# Mappings between round-robin names and regexps to match members.  Names are
# matched within each network (i.e., if there are members matching a mapping in
# multiple networks then the name will be defined separately in each).
round-robin:
  # E.g., if the "home" network defined above had members "k8s-node-23refw" and
  # "k8s-node-09sf8g" this would create a name "k8s-nodes.home.zt" returning one
  # of them at random:
  #   k8s-nodes: "k8s-node-\w"
```

Configuration options can be overridden using environment variables, where the variable name is "ZTDNS_" followed by the option's name in uppercase with hyphens change to underscores (e.g. the `api-url` option is overridden by the `ZTDNS_API_URL` environment variable).
Command-line options override both.



## Contributing

Thanks for considering contributing to the project.
We welcome contributions, issues or requests from anyone, and are grateful for any help.
Problems or questions?
Feel free to open an issue on GitHub.

Please make sure your contributions adhere to the following guidelines:

* Code must adhere to the official Go [formating](https://golang.org/doc/effective_go.html#formatting) guidelines  (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Pull requests need to be based on and opened against the `master` branch.
