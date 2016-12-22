#!/usr/bin/python

import sys
from matplotlib import pyplot as plt
import numpy as np

from read import *

plt.cla()
plt.clf()

plt.xlabel('Time (ns)')
plt.ylabel('Throughput (Rin) (bps)')

def tpt(lines):
    for l in lines:
        yield (l['t'], l['rate'])

if __name__ == '__main__':
    els = list(readLines())
    xa, tpt = zip(*tpt(els))

    start = min(xa)
    xaxis = np.array(xa) - start

    print xaxis[:10], len(xaxis)
    print tpt[:10], len(tpt)

    plt.plot(xaxis, tpt, label='Rin')

    #plt.legend()
    plt.show()

