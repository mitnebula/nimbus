#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readNimbusLines, readIperfLines

plt.cla()
plt.clf()

plt.xlabel('Time (s)')
plt.ylabel('Throughput (Rout) (bps)')

def tpt(lines):
    for l in lines:
        if 'rout' in l and 'time' in l:
            yield float(l['time']), l['rout']

def itpt(lines):
    for l in lines:
        if 'time' in l and 'tpt' in l:
            yield l['time'], l['tpt']*1e6

if __name__ == '__main__':
    with open(sys.argv[1], 'r') as f:
        nimbus = list(readNimbusLines(f))
    with open(sys.argv[2], 'r') as f:
        iperf = list(readIperfLines(f))

    nxa, tpt = zip(*tpt(nimbus))
    ixa, itpt = zip(*itpt(iperf))

    plt.plot(nxa, tpt, label='nimbus')
    plt.plot(ixa, itpt, label='iperf')

    plt.legend(loc='lower right')
    plt.show()

