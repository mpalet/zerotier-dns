<h1 align="center">
  ztdns
</h1>

<h4 align="center">
  A DNS server for ZeroTier virtual networks.
</h4>

<p align="center">
  <a href="https://github.com/mje-nz/ztdns">
    <img src="https://github.com/mje-nz/ztdns/workflows/Check/badge.svg"
         alt="Github Actions">
  </a>
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
  -p 53:53/udp \
  --volume $(pwd)/ztdns.yml:/app/ztdns.yml \
  mjenz/ztdns server --api-key API_KEY --network NETWORK_ID
```

where `API_KEY` is a ZeroTier API token and `NETWORK_ID` is a ZeroTier network ID.
If your ZeroTier network interface is not called `zt0`, then you should add `--interface=<your ZeroTier interface>` (or `--interface=` to listen on all interfaces).
It is recommended to use a configuration file instead of these command-line arguments (see ["Configuration"](#configuration) section below).

Once the server is running you will be able to resolve ZeroTier member names by querying it directly:

```bash
dig @<server address> <member name>.<ztdns domain>

;; QUESTION SECTION:
;matthews-mbp.zt.   IN  A

;; ANSWER SECTION:
matthews-mbp.zt.  3600  IN  A 192.168.192.120
```

Note that the DNS name is based on the ZeroTier member name (as shown in [ZeroTier Central](https://my.zerotier.com/network)), not the hostname of the member.

In order to resolve names normally, you need to get the server into the DNS lookup chain on all of your machines.
In practise this means either configuring the system resolver on each machine to use your `ztdns` instance for your chosen domain (see instructions for [Linux](https://learn.hashicorp.com/consul/security-networking/forwarding#systemd-resolved-setup) or [macOS](https://learn.hashicorp.com/consul/security-networking/forwarding#macos-setup)), or configuring the DNS server each machine uses to delegate to your `ztdns` instance for your chosen domain (see instructions for [dnsmasq](https://learn.hashicorp.com/consul/security-networking/forwarding#dnsmasq-setup) or [bind](https://learn.hashicorp.com/consul/security-networking/forwarding#bind-setup)).



## Building from source
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

If you are running on Linux, run `sudo setcap cap_net_bind_service=+ep /go/bin/ztdns` to enable non-root users to bind privileged ports.
On other operating systems, `ztdns` may need to be run as an administrator.
This does not apply to the Docker image.



## Installation as a service
For both of these methods it is assumed your configuration is stored in a configuration file, although this is not compulsory.


### Docker
To install the server as a service using the Docker image:

```bash
docker run --detach \
  -p 53:53/udp \
  --volume $(pwd)/ztdns.yml:/app/ztdns.yml \
  --restart=unless-stopped \
  --name=ztdns mjenz/ztdns
```


### Systemd
To install the server as a `systemd` service, create a file `/etc/systemd/system/ztdns.service` containing:

```ini
[Unit]
Description=Zerotier DNS Server
After=network-online.target
Wants=network-online.target

[Service]
WorkingDirectory=<path containing config file>
ExecStart=ztdns server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Make sure you set `WorkingDirectory` to the directory containing your configuration file.

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
There is a `ztdns.example.yml` example config file with all supported options and their default values:

```yaml
# Network interface to bind to (or "" to bind to all interfaces).  By default, only
# respond on the ZeroTier interface.
interface: "zt0"

# Port to listen on.
port: 53

# Base domain.  Could be a top-level domain for internal use only (e.g., zt) or
# a domain name with one or more subdomains (e.g., internal.yourdomain.com).
# By default, map members to "<member name>.zt".
origin: "zt"

# How often to poll the ZeroTier controller in minutes.
refresh: 30

# Include members that are currently offline.
include-offline: true

# Enable debug messages.
debug: false

# An API key for your ZeroTier account (required).
api-key: ""

# The base API URL for the ZeroTier controller.
api-url: "https://my.zerotier.com/api"

# ID of the ZeroTier network.  Only one of "network" and "networks" can be
# specified.  E.g., if there is a network with ID "123abc" then this would map
# its members to "<member name>.zt":
#   network: "123abc"
network:

# Mappings between subdomains and ZeroTier network IDs.  Only one of "network"
# and "networks" can be specified.
networks:
  # E.g., if origin="zt" and there is a network with ID "123abc" then this would
  # map its members to "<member name>.home.zt":
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

Configuration options can be overridden using environment variables, where the variable name is "ZTDNS_" followed by the option's name in upper-case with hyphens changed to underscores (e.g., the `api-url` option is overridden by the `ZTDNS_API_URL` environment variable).
Command-line options override both.



## Contributing

Thanks for considering contributing to the project.
We welcome contributions, issues or requests from anyone, and are grateful for any help.
Problems or questions?
Feel free to open an issue on GitHub.

Please make sure your contributions adhere to the following guidelines:

* Code must adhere to the [official Go formatting guidelines](https://golang.org/doc/effective_go.html#formatting) (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
* Pull requests need to be based on and opened against the `master` branch.
