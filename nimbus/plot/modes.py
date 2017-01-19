#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readNimbusLines, readShortIperfs

plt.cla()
plt.clf()

plt.xlabel('Time (s)')
plt.ylabel('Mode (0 = delay, 0.5 = test, 1 = xtcp)')

plt.ylim(0,1)

def mkMds(lines):
    for l in lines:
        if 'time' in l and 'to' in l:
            m = 0.1
            if l['to'] == 'XTCP':
                m = 0.9
            elif 'TEST' in l['to']:
                m = 0.5
            yield float(l['time']), m
        elif 'time' in l and 'mode' in l:
            m = 0.1
            if l['mode'] == 'XTCP':
                m = 0.9
            elif 'TEST' in l['mode']:
                m = 0.5
            yield float(l['time']), m

def enumIperf(lines):
    for l in lines:
        yield l['time'], 0.5

if __name__ == '__main__':
    with open(sys.argv[1], 'r') as f:
        nimbus = list(readNimbusLines(f))
    with open(sys.argv[2], 'r') as f:
        iperf = list(readShortIperfs(f))

    nxa, mds = zip(*mkMds(nimbus))
    plt.plot(nxa, mds)

    for ip in iperf:
        ixa, ons = zip(*enumIperf(ip))
        plt.plot(ixa, ons, color='black')

    plt.show()

