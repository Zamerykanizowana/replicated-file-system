We needed something simple for p2p communication, here are the condisderations:
- [ZeroMQ](https://zeromq.org/) has a couple of golang libraries avaialble:
  - [pebbe/zmq4](https://github.com/pebbe/zmq4) seems to the one with the best
    maintenance, it serves bindings for the libzmq (C++)
  - [goczmq](https://github.com/zeromq/goczmq) interface for CZMQ v4.2 (yeah,
    there are two versions of ZeroMQ, one written in C++ and C)
  - [gomq](https://github.com/zeromq/gomq) is actually a pure GO ZeroMQ
    implementation, so no dependencies, but It seems they dropped it 2 years
    ago...
  - [go-zeromq/zmq4](https://github.com/go-zeromq/zmq4) - yeah, another zmq4
    library, this on however is a pure go implementation, just like `gomq`,
    seems to have to commits from this year, so not bad.
- I've stumbled upon something called Scalability Protocols (aka nanomsg), It 
  seems former zeromq bois have dropped from that train and wanted to do sth 
  similar but different, here comes [Mangos](https://github.com/nanomsg/mangos)
  which is a pure Go implementation of these `Scalability Protocols`, they also
  maintain and develop a pure C impl callen [nng](https://github.com/nanomsg/nng)
  which translates to `nanomsg-next-gen`. Basically It does similar things to 
  ZeroMQ with less bloat, and since we don't need anything complex here, we
  might just want to go with `Mangos`. Another advantage I see here is that
  we'll be able to debug the library itself!

Here are a few links I've gathered from:
- [pres on nng](https://staysail.github.io/nng_presentation/nng_presentation.html)
- [nanomsg vs ZeroMQ](https://nanomsg.org/documentation-zeromq.html)
- [nng site](https://nng.nanomsg.org/)
- [ZeroMQ opinionated course :D](https://www.youtube.com/watch?v=UrwtQfSbrOs)
- [why nanomsg was made after ZeroMQ](https://www.youtube.com/watch?v=8HLOgzvndAA)
- [whole podcast on nanomsg](https://www.youtube.com/watch?v=JzCHr2HgLEw)
