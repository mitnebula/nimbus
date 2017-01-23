#!/bin/bash

scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-$2.out . 

python ./zt-rtt.py $1-$2.out &
