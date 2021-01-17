# API
## Functions
> `RegisterChatCommand(name string, privs map[string]bool, function func(*Peer, string))`
```
name: Name of the command
privs: Required privileges
function: Callback function, arguments: client peer + parameters
Description: Registers a chat command for clients
```
> `RegisterOnChatMessage(function func(*Peer, string))`
```
function: Callback function, arguments: client peer + chat message
Description: Registers a callback for chat messages sent by clients
```
> `RegisterServerChatCommand(name string, function(*Peer, string))`
```
name: Name of the command
function: Callback function, arguments: client peer + parameters
Description: Registers a chat command for servers
```
> `RegisterOnServerChatMessage(function func(*Peer, string))`
```
function: Callback function, arguments: client peer + chat message
Description: Registers a callback for chat messages sent by servers
```
> `ChatSendAll(msg string)`
```
msg: Message to send
Description: Sends msg to all connected clients
```
> `GetConfKey(key string) interface{}`
```
key: Key to read
Description: Returns a configuration setting (type assertion needed)
```
> `End(crash, reconnect bool)`
```
crash: Whether the shutdown reason is a crash or not
reconnect: Whether to show a reconnect button on the client
Description: Kick all clients and stop the program
```
> `GetListener() *Listener`
```
Description: Returns the listener the proxy is using
```
> `GetPeerCount() int`
```
Description: Returns the number of connected client peers
```
> `RegisterOnJoinPlayer(function func(*Peer))`
```
function: Callback function for join events, arguments: client peer
Description: Registers a callback for TOCLIENT_READY packets
```
> `RegisterOnLeavePlayer(function func(*Peer))`
```
function: Callback function for client disconnect events, arguments: client peer
Description: Registers a callback function for Disco packets and timeouts
```
> `IsOnline(name string) bool`
```
name: The player's name
Description: Reports whether a player named name is connected to the proxy
```
> `RegisterOnRedirectDone(function func(*Peer, string, bool))`
```
function: Callback function for successful server redirects, arguments: client peer + server name passed to the
Peer.Redirect method + whether the redirect succeeded
Descriptions: Registers a callback function for successful redirects (server hopping)
```
> `InitStorageDB() (*sql.DB, error)`
```
Description: Initialises the storage database and returns a database handle for it
```
> `ModOrAddStorageItem(db *sql.DB, key, value string) error`
```
db: Storage database handle
key: Key to modify
value: New value
Description: Modifies an entry in the storage database
```
> `DeleteStorageItem(db *sql.DB, key string) error`
```
db: Storage database handle
key: Key to delete
Description: Deletes an entry in the storage database
```
