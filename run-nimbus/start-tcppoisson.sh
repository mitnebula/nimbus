#!/usr/bin/zsh

echo "starting nimbus $6 $7 $8e6"
nimbus --log debug --mode sender --ip $MAHIMAHI_BASE --port 42424 --t 2m --useSwitching=$10 --initMode $9 --estBandwidth $2e6 --initRate $4e6 --numFlows $6 --pulseSize $3 --measurementInterval 10ms --measurementTimescale $5 > "$1-tcppoisson.out" &
#
sleep 2
if [[ $8 -ge 1 ]]; then
	trafficgen --mode sender --ip $MAHIMAHI_BASE --port 42426 --initRate $8e6 --t 2m --msgSizeBytes 10220 > $1-t1.out &
fi
if [[ $7 -ge 1 ]]; then
	iperf -c $MAHIMAHI_BASE -p 42425 -t 120 -i 1 -P $7 -Z reno > "$1-iperf-tcppoisson.out" &
fi
sleep 120
