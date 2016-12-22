#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import *

plt.cla()
plt.clf()

plt.xlabel('Time (s)')
plt.ylabel('RTT (us)')

def rtt(lines):
    for l in lines:
        yield (l['t'], l['rtt'])

def minrtt(lines):
    for l in lines:
        yield (l['t'], l['minrtt'])

if __name__ == '__main__':
    els = list(readLines())
    xa, rtts = zip(*rtt(els))
    _, minrtts = zip(*minrtt(els))

    start = min(xa)
    xaxis = np.array(xa) - start

    print xaxis[:10]
    print rtts[:10]
    print minrtts[:10]

    plt.ylim(0, 1e5)

    plt.plot(xaxis, rtts, label='rtt')
    plt.plot(xaxis, minrtts, label='minrtt')

    plt.legend()
    plt.show()

