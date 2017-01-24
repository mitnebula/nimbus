#!/bin/bash

scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-cbr.out . 
scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-tcp.out . 
scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-poisson.out . 

python ./elasticity.py $1-cbr.out &
python ./elasticity.py $1-tcp.out &
python ./elasticity.py $1-poisson.out &
