#!/bin/bash

scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-$2.out . 

python ./elasticity.py $1-$2.out &
