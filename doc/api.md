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
Description: Registers a callback function for disconnect packets and timeouts
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
## Structs
### Peer
#### Methods
> `Addr() net.Addr`
```
Description: Returns the address of the peer
```
> `Disco() <-chan struct{}`
```
Description: Returns a channel that is closed when the peer is closed
```
> `ID() PeerID`
```
Description: Returns the ID of the peer
```
> `IsSrv() bool`
```
Description: Reports whether the peer is a server
```
> `TimedOut() bool`
```
Description: Reports whether the peer has timed out
```
> `Username() string`
```
Description: Returns the username of the peer
```
> `Server() *Peer`
```
Description: Returns the peer this peer is connected to if it isn't a server
```
> `ServerName() string`
```
Description: Returns the name of the server this peer is connected to if this peer is not a server
```
> `Close() error`
```
Description: Closes the peer but does not send a disconnect packet
```
> `SendChatMsg(msg string)`
```
msg: Message to send
Description: Sends msg to the peer
```

### Listener
#### Methods
> `GetPeerByName(name string) *Peer`
```
name: The username of the peer
Description: Returns the peer with username name
```
> `GetPeers() []*Peer`
```
Description: Returns an array containing all connected client peers
```
