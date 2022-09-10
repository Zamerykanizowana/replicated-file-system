# Replicated File System

As the glorious name suggests, it's a replicated file system.
But it's not just replicated! It's actually a peer-2-peer
(as in decentralized, and all participants are equal) FS where all the
peers work on the same FS which is replicated on their local machines.

## Core concepts

1. Decentralization is a myth, it only exists in theory.
   Our centralized entity is the config file in json format, which tells
   each peer where other peers are located.
2. Peer is identified by it's name, which has to be unique.
3. Each peers' TLS certificate has to be signed by the same authority
   and must contain Common Name (CN) equal to it's name.
4. QUIC protocol is used at the transport layer.
5. Custom, simple protocol is used at the application layer (no HTTP).
6. Peers talk only through "broadcast", there is no peer-2-peer here, rather peer-2-peers.
7. FUSE is used to capture FS operations.
8. FUSE serves as rich mans `inotify`,
   we're injecting peers replication agreement process before executing any operation.
9. Each replication request is treated as a transaction.
   All peers must respond the the request and their responses must be public.
10. Each file changing operation is consulted between all peers.
    All peers must agree before the operation is executed.
11. Each peer decides on it's own if it should replicate (mirror) the requested change
    by gathering responses of other peers and his own.

## Project structure

| Name        | Description                                                             |
|-------------|-------------------------------------------------------------------------|
| `adr`       | Architecture decision records. Why we did what we did.                  |
| `config`    | Reading and parsing config files along with the default config.json.    |
| `mirror`    | Applying and consulting (do I agree to replicate) replication requests. |
| `protobuf`  | Protbuf definitions and auto generate `go` code along with our helpers. |
| `logging`   | [Zerolog](https://github.com/rs/zerolog) configuration.                 |
| `rfs`       | FUSE server and bindings.                                               |
| `p2p`       | Peer-2-Peer replication mechanisms.                                     |
| `connetion` | The underlying network connection.                                      |
| `test`      | End to end tests.                                                       |

## Running

We provide both `Makefile` and `Dockerfile` and strongly suggest running it with either.
Before running any commands you should make sure the Certificate Authority is first generated:

```shell
make cert/generate-ca
```

The below command spins a single peer with the name `Gimli` using config under the `var/config-gimli.json` path.

```shell
PEER=Gimli CONFIGPATH=var/config-gimli.json make
```

For more details run either `make help` or simply `make`.

## Configuring

```text
NAME:
   rfs - Replicated file system using FUSE bindings and peer-2-peer architecture

USAGE:
   rfs [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config value, -c value  Load configuration from 'FILE' (default: By default embedded config.json is loaded)
   --help, -h                show help (default: false)
   --name value, -n value    Provide peer name, It must be also present in the config, linked to an address
```

Aside from `-c/--config` which provides  and `-n/--name` flags all configuration is done through a json config file.
You can view the schema with constraints and detailed information on what's what
inside [config.go](./config/config.go).

Here's an example:

```json
{
  "connection": {
    "tls_version": "1.3",
    "message_buffer_size": 10000,
    "dial_backoff": {
      "initial": 100000000,
      "max": 30000000000,
      "factor": 1.3,
      "max_factor_jitter": 0.2
    },
    "send_recv_timeout": 60000000000,
    "handshake_timeout": 5000000000,
    "network": "tcp",
    "compression": "DefaultCompression"
  },
  "peers": [
    {
      "address": "localhost:9001",
      "name": "Aragorn"
    },
    {
      "address": "localhost:9005",
      "name": "Gimli"
    }
  ],
  "paths": {
    "fuse_dir": "~/other/rfs/virtual",
    "mirror_dir": "~/other/rfs/contents"
  },
  "logging": {
    "level": "info"
  },
  "replication_timeout":"300000000000"
}
```

## Troubleshooting

A common issue occurs when we kill the `rfs` process when the virtual (mounted) directory is still busy.
You will get an error log explaining what's wrong and what manual steps should be taken to fix the issue.
All you need to do is manually unmount the directory yourself by calling `fusermount`:

```shell
fusermount -u ${VIRTUAL_MOUNT_PATH}
```

The system is not 100% fail proof, the greatest danger happens when the `rfs` process is abruptly
(this means no graceful shutdown was achieved) closed during one of the transactions, then we might run
into a dirty state if the peer has not mirrored (or finished via loopback FS) the transaction yet.
It will be missing one or more FS changes and it will require manual fixing with `rfs` unplugged.

At the moment we haven't yet figured out how to run `rfs` in Docker:

```text
unable to mount fuse filesystem: exec: \"/bin/fusermount\": stat /bin/fusermount: no such file or directory
```