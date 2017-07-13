Nimbus Userspace Implementation
===============================

Installation
------------

This is a go project.

1. Set up go: https://golang.org/doc/install
2. ```go get github.com/mitnebula/nimbus/...```
3. ```go install github.com/mitnebula/nimbus/nimbus``` and ```go install github.com/mitnebula/nimbus/trafficgen```

Running
-------

```nimbus``` will start a Nimbus sender, receiver, client, or server according to ```--mode```. A sender/receiver pair will send traffic from the sender to the receiver, with the sender initiating the connection. A client/server pair will send traffic from the server to the client, with the client initiating the connection. This can be useful if one end is behind a NAT.

There are various options in ```nimbus/nimbus.go``` which set algorithm parameters.

```trafficgen``` has the same modes as ```nimbus```, and will send Poisson traffic.
