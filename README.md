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
