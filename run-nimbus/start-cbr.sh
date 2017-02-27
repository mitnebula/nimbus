#!/usr/bin/zsh

echo "starting nimbus"
nimbus --log debug --mode sender --ip $MAHIMAHI_BASE --port 42424 --t 3m --useSwitching=f --initMode DELAY  --estBandwidth $2e6 --initRate $4e6 --numFlows 1 --pulseSize $3 --measurementInterval 10ms --measurementTimescale $5 > "$1-cbr.out" &
#
sleep 2
iperf -c $MAHIMAHI_BASE -u -b 12M -p 42427 -t 180 -i 1 > "$1-iperf-cbr.out" &
sleep 180
