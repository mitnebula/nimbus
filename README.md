Nimbus Userspace Implementation
===============================

Installation
------------

This is a go project. Its location as a subdirectory within this repository makes running it slightly complicated.

1. Install golang and set up a ```$GOPATH``` directory (usually ```~/go-work```).

2. Add $GOPATH/bin to your PATH.

3. Symlink the packages in this directory into ```$GOPATH/src/github.mit.edu/hari/nimbus-cc```

4. ```go install nimbus``` and ```go install trafficgen```

Running
-------

```nimbus``` will start a Nimbus sender, receiver, client, or server according to ```--mode```. A sender/receiver pair will send traffic from the sender to the receiver, with the sender initiating the connection. A client/server pair will send traffic from the server to the client, with the client initiating the connection. This can be useful if one end is behind a NAT.

There are various options in ```nimbus/nimbus.go``` which set algorithm parameters.

```trafficgen``` has the same modes as ```nimbus```, and will send Poisson traffic.
