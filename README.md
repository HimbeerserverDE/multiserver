# multiserver
Minetest reverse proxy supporting multiple servers and media multiplexing

## Credits
This project was made possible by [anon55555's Minetest RUDP package](https://github.com/anon55555/mt/tree/master/rudp).

## Modchannel RPC API
There is a modchannel-based RPC API for minetest servers: [click here](https://github.com/HimbeerserverDE/multiserver_api).

## Installation
Go 1.16 or higher is required

`go get github.com/HimbeerserverDE/multiserver`

### Updating
`go get -u github.com/HimbeerserverDE/multiserver`

## How to use
**Note: This shouldn't be used with existing minetest servers without moving authentication data (not the database files) to the proxy and then deleting the auth databases on the minetest servers! Not doing so can cause proxy <-> mt_server authentication to fail.**

### Running
The `go get` command will create an executable file in `~/go/bin/multiserver`.
This file should always be run from the same working directory. If you don't do this, the program will be unable to read the old data and will create
the default files in the new working directory.

### Configuration
The configuration file is located in `WORKING_DIR/config/multiserver.yml`

- Default config file
```yml
servers:
  lobby:
    address: "127.0.0.1:30000"
default_server: lobby
force_default_server: true
```

> `host` 
```
Type: String
Description: The IP address and port the proxy will be running on,
default is 0.0.0.0:33000
```
> `player_limit`
```
Type: Integer
Description: Maximum number of concurrent connections, unlimited if not set
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
> `csm_restriction_flags`
```
Type: Integer
Description: The CSM restriction flags, default is none
* 0: No restrictions
* 1: No CSM loading
* 2: No chat message sending
* 4: No itemdef reading
* 8: No nodedef reading
* 16: Limit node lookup
* 32: No player info lookup
* 63: All restrictions
To set multiple flags at the same time add the corresponding numbers
```
> `server_reintergration_interval`
```
Type: Integer
Description: Number of seconds between server reintegrations, default is 600.
A server reintegration is reconnecting the RPC user and refetching the media
from one or more minetest servers.
```
> `disable_builtin`
```
Type: Boolean
Description: If this is true the builtin chatcommands and the redirect error message are deactivated, default is false
```
> `command_prefix`
```
Type: String
Description: Prefix of proxy chat commands, default is #
```
> `do_fallback`
```
Type: Boolean
Description: If this is true players are automatically redirected to
the default server if the server they are on shuts down or crashes,
default is true
```
> `disallow_empty_passwords`
```
Type: Boolean
Description: Whether to deny access if the password is empty,
default is false
```
> `serverlist_url`
```
Type: String
Description: The URL of the list server to announce to,
announcements are disabled if this is unset
```
> `serverlist_address`
```
Type: String
Description: The server address to be displayed on the server list,
can be omitted
```
> `serverlist_name`
```
Type: String
Description: The name to be displayed on the server list, can be omitted
```
> `serverlist_desc`
```
Type: String
Description: The description to be displayed on the server list,
can be omitted
```
> `serverlist_display_url`
```
Type: String
Description: The URL of the website of the server, can be omitted
```
> `serverlist_creative`
```
Type: Boolean
Description: Whether one of the minetest servers has creative enabled,
default is false
```
> `serverlist_damage`
```
Type: Boolean
Description: Whether one of the minetest servers has damage enabled,
default is false
```
> `serverlist_pvp`
```
Type: Boolean
Description: Whether one of the minetest servers has PvP enabled,
default is false
```
> `serverlist_game`
```
Type: String
Description: The subgame that is used on the minetest servers,
can be omitted
```
> `serverlist_can_see_far_names`
```
Type: Boolean
Description: Whether unlimited player view range is enabled,
default is false
```
> `serverlist_mods`
```
Type: List
Description: All mods that have been installed on any of the
minetest servers, can be omitted
```
> `serverlist_announce_interval`
```
Type: Integer
Description: Number of seconds between serverlist announcement updates,
default is 300
```
