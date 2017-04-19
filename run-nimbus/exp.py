#!/usr/bin/python

import subprocess

def runExp(name, bw, rtt, bufSizeBDPs, pulseSize, crossTrafficPattern, sendRate, measureTimescale, numflows, tcpflows, poissontfk):
    subprocess.call("go install github.mit.edu/hari/nimbus-cc/nimbus", shell=True)

    killAll()

    # TCP
    # start outside-mahimahi stuff
    subprocess.call("nimbus --mode receiver --port 42424 &", shell=True)
    subprocess.call("iperf -s -p 42425 &", shell=True)
    subprocess.call("trafficgen --mode receiver --port 42426 &", shell=True)
    subprocess.call("iperf -s -u -p 42427 &", shell=True)

    mmCmdTmp = 'mm-delay {0} mm-link --uplink-queue="droptail" --uplink-queue-args="packets={1}" --downlink-queue="droptail" --downlink-queue-args="packets={1}" ~/bw{2}.mahi ~/bw{2}.mahi ./start-{4}.sh {5} {2} {3} {6} {7} {8} {9} {10}'

    # start mahimahi
    bdp = bw * 1e6 * rtt * 1e-3 / 1500 / 8
    assert bdp % 100 == 0
    oneWay = rtt / 2

    outFile = '{}-pulse{}-buffer{}-bw{}-rtt{}-rate{}-nimbusflows{}-tcpflows{}-poissontfk{}'.format(
        name,
        int(pulseSize * 100),
        bufSizeBDPs,
        bw,
        rtt,
        sendRate,
        numflows,
        tcpflows,
        poissontfk,
    )

    mmCmd = mmCmdTmp.format(
        oneWay,
        bdp * bufSizeBDPs,
        bw,
        pulseSize,
        crossTrafficPattern,
        outFile,
        sendRate,
        measureTimescale,
        numflows,
        tcpflows,
        poissontfk,

    )
    print mmCmd

    subprocess.call(mmCmd, shell=True)

    killAll()

def killAll():
    subprocess.call("killall -9 nimbus", shell=True)
    subprocess.call("killall -9 client", shell=True)
    subprocess.call("killall -9 server", shell=True)
    subprocess.call("killall -9 iperf", shell=True)
    subprocess.call("killall -9 trafficgen", shell=True)
