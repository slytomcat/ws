# ws

ws is a simple command line websocket client designed for exploring and debugging websocket servers. ws includes readline-style keyboard shortcuts, persistent history, and colorization.

![Example usage recording](https://hashrocket-production.s3.amazonaws.com/uploads/blog/misc/ws/ws.gif)

## Installation

```
go install github.com/slytomcat/ws@latest
```

Or download compiled binary (ELF 64-bit LSB executable, x86-64) from assets of [releases](https://github.com/slytomcat/ws/releases)

## Usage

```
ws URL [flags]
```

Simply run ws with the destination URL. For security some sites check the origin header. ws will automatically send the destination URL as the origin. If this doesn't work you can specify it directly with the `--origin` option.

Example of usage with echo server (see below):
```
$ ws ws://localhost:8080/ws
> {"type": "echo", "payload": "Hello, world"}
< {"type":"echo","payload":"Hello, world"}
> {"type": "broadcast", "payload": "Hello, world"}
< {"type":"broadcast","payload":"Hello, world"}
< {"type":"broadcastResult","payload":"Hello, world","listenerCount":1}
> ^D
```

## Other possible options

```
Flags:
  -a, --auth string          auth header value, like 'Bearer $TOKEN'
  -b, --bin2text             print binary message as text
  -c, --compression          enable compression
  -f, --filter string        only messages that match regexp will be printed
  -h, --help                 help for ws
  -m, --init string          connection init message
  -k, --insecure             skip ssl certificate check
  -i, --interval duration    send ping each interval (ex: 20s)
  -o, --origin string        websocket origin (default value is formed from URL)
  -p, --pingPong             print out ping/pong messages
  -s, --subprotocal string   sec-websocket-protocal field
  -t, --timestamp            print timestamps for sent and received messages
  -v, --version              print version  
```

# Echo server

Folder `echo-server` contains a very simple echo server. It allows to establish ws connection and just replay with received messages or send the message to all active connection. Server accept messages in JSON format (like `{"type": "echo", "payload": "Hello, world"}`). 

Only wto types allowed:
  - `echo` - the message replayed to sender only
  - `broadcast` - the message is sent to all active connection and the result of broadcasting is sent to sender. 

## build

```
cd echo-server
./build.sh
```

## start

```
./echo-server ws://localhost:8080/ws
```

## test

```
ws ws://localhost:8080/ws
> {"type": "echo", "payload": "Hello, world"}
< {"type":"echo","payload":"Hello, world"}
> {"type": "broadcast", "payload": "Hello, world"}
< {"type":"broadcast","payload":"Hello, world"}
< {"type":"broadcastResult","payload":"Hello, world","listenerCount":1}
> ^D
```