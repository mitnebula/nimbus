Nimbus Userspace Implementation
===============================

Installation
------------

This is a go project. Its location as a subdirectory within this repository makes running it slightly complicated.

1. Install golang and set up a go work directory (usually ```~/go-work```). Make a "src" directory inside this work directory.

2. Add $GOPATH/bin to your PATH.

3. In this directory, make a symlink in the go-work src: ```ln -s . $GOPATH/src/nimbus```

4. run ```go install nimbus```

5. run the program nimbus (it should be in your path)

Running
-------

No arguments - ```nimbus``` - will start a sender and send packets to localhost on port 42424. The ```--ip``` and ```--port``` options change this behavior.

To run a receiver, use ```nimbus --mode receiver --port 42424```.
