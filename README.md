# multiserver
Minetest reverse proxy supporting multiple servers and media multiplexing

## Credits
This project is based on and was made possible by [anon55555's RUDP package](https://github.com/anon55555/mt).

## Installation
Go 1.15 or higher is required

`go get github.com/HimbeerserverDE/multiserver`

`cd ~/go/src/github.com/HimbeerserverDE/multiserver/multiserver`

`go build`
### Updating
`go get -u github.com/HimbeerserverDE/multiserver`

`cd ~/go/src/github.com/HimbeerserverDE/multiserver/multiserver`

`go build`

## How to use
**Note: This shouldn't be used with existing minetest servers without moving authentication data (not the database files) to the proxy and then deleting the auth databases on the minetest servers! Not doing so can cause proxy <-> mt_server authentication to fail.**

### Running
The `go build` command will create an executable file in `~/go/src/github.com/HimbeerserverDE/multiserver/multiserver/multiserver`.
This file should always be run from the same working directory. If you don't do this, the program will be unable to read the old data and will create
the default files in the new working directory.

### Configuration
The configuration file is located in `WORKING_DIR/config/multiserver.yml`

- Default config file
```yml
host: "0.0.0.0:33000"
player_limit: -1
servers:
  lobby:
    address: "127.0.0.1:30000"
default_server: lobby
force_default_server: true
```

> `host` 
```
Type: String
Description: The IP address and port the proxy will be running on
```
> `player_limit`
```
Type: Integer
Description: Maximum number of concurrent connections, unlimited if the value is negative
```
> `servers`
```
Type: Dictionary
Description: List of all servers
```
> `servers.*`
```
Type: Dictionary
Description: Contains server details
```
> `servers.*.address`
```
Type: String
Description: The IP address and port this server is running on
```
> `servers.*.priv`
```
Type: String
Description: If this is set, players need this privilege (on the proxy) to connect to this server.
Leave this empty to require no special privilege. Note that this only affects the #server command. This means that
players can still be connected to this server if it is the default server, a minetest server requests it
or if a different command is used.
```
> `default_server`
```
Type: String
Description: Name of the minetest server new players are sent to
```
> `force_default_server`
```
Type: Boolean
Description: If this is false known players are sent to the last server they were connected to, not to the default server
```
> `admin`
```
Type: String
Description: The "privs" privilege is granted to the player with this name on startup
```
