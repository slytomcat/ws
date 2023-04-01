# ws

ws is a simple command line websocket client designed for exploring and debugging websocket servers. ws includes readline-style keyboard shortcuts, persistent history, and colorization.

![Example usage recording](https://hashrocket-production.s3.amazonaws.com/uploads/blog/misc/ws/ws.gif)

## Installation

```
go install -u github.com/slytomcat/ws
```

## Usage

Simply run ws with the destination URL. For security some sites check the origin header. ws will automatically send the destination URL as the origin. If this doesn't work you can specify it directly with the `--origin` parameter.

```
$ ws ws://localhost:3000/ws
> {"type": "echo", "payload": "Hello, world"}
< {"type":"echo","payload":"Hello, world"}
> {"type": "broadcast", "payload": "Hello, world"}
< {"type":"broadcast","payload":"Hello, world"}
< {"type":"broadcastResult","payload":"Hello, world","listenerCount":1}
> ^D
EOF
```

## Help message

```websocket tool v. 0.2.2

Usage:
  ws URL [flags]

Flags:
  -b, --bin2text             print binary message as text
  -h, --help                 help for ws
  -k, --insecure             skip ssl certificate check
  -o, --origin string        websocket origin
  -s, --subprotocal string   sec-websocket-protocal field
  -t, --timestamp            print timestamps for sent and incoming messages
  -v, --version              print version
  
  ```