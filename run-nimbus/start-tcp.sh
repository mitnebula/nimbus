#!/usr/bin/zsh

echo "starting nimbus"
nimbus --log debug --mode sender --ip $MAHIMAHI_BASE --port 42424 --t 3m --useSwitching=f --initMode XTCP --estBandwidth $2e6 --initRate $4e6 --numFlows 1 --pulseSize $3 --measurementInterval 10ms --measurementTimescale $5 > "$1-tcp.out" &
#
sleep 2
iperf -c $MAHIMAHI_BASE -p 42425 -t 180 -i 1 -P 1 -Z reno > "$1-iperf-tcp.out" &
sleep 180
