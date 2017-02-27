#!/usr/bin/zsh

echo "starting nimbus"
nimbus --log debug --mode sender --ip $MAHIMAHI_BASE --port 42424 --t 3m --useSwitching=f --initMode DELAY --estBandwidth $2e6 --initRate $4e6 --numFlows 1 --pulseSize $3 --measurementInterval 10ms --measurementTimescale $5 > "$1-poisson.out" &

sleep 2
#~/empirical-traffic-gen/bin/client -c trafficConfig > trafficGen.out &
echo "tgen 1"
trafficgen --mode sender --ip $MAHIMAHI_BASE --port 42426 --initRate 12e6 --t 3m --msgSizeBytes 10220 > $1-t1.out &
#sleep 5
#echo "tgen 2"
#trafficgen --mode sender --ip $MAHIMAHI_BASE --port 24243 --initRate 4e6 --t 45s --msgSizeBytes 10220 > $1-t2.out &
#sleep 5
#echo "tgen 3"
#trafficgen --mode sender --ip $MAHIMAHI_BASE --port 24244 --initRate 4e6 --t 30s --msgSizeBytes 10220 > $1-t3.out &
#sleep 5
#echo "tgen 4"
#trafficgen --mode sender --ip $MAHIMAHI_BASE --port 24245 --initRate 4e6 --t 25s --msgSizeBytes 10220 > $1-t4.out &
#sleep 5
#echo "tgen 5"
#trafficgen --mode sender --ip $MAHIMAHI_BASE --port 24246 --initRate 4e6 --t 15s --msgSizeBytes 10220 > $1-t5.out &

sleep 180
