#!/bin/bash

scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-cbr.out . 
scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-tcp.out . 
scp akshay@bespin.akshayn.xyz:~/run-nimbus/$1-poisson.out . 

python ./elasticity_all.py $1-cbr.out &
python ./elasticity_all.py $1-tcp.out &
python ./elasticity_all.py $1-poisson.out &
