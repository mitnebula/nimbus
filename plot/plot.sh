#!/bin/bash

scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-$2.out . 

python ./elasticity_all.py $1-$2.out &
