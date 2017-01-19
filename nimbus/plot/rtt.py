#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import readNimbusLines

plt.cla()
plt.clf()

plt.xlabel('Time (s)')
plt.ylabel('RTT (s)')

plt.ylim(0, 0.5)

def rtt(lines):
    for l in lines:
        if 'time' in l and 'rtt' in l:
            yield (l['time'], l['rtt'])

if __name__ == '__main__':
    with open(sys.argv[1], 'r') as f:
        nimbus = list(readNimbusLines(f))

    xa, rtts = zip(*rtt(nimbus))

    plt.plot(xa, rtts, label='rtt')

    plt.show()

