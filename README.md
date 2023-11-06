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

```
$ ws ws://localhost:3000/ws
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