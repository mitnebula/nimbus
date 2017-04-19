#!/usr/bin/python

import subprocess
from exp import runExp

name = 't10'
bw = 48
buf = [1]

pulseSizes = [0.25]#, 0.5, 0.75]
rtts = [100]
initRates = [24]#, 12, 20]
measures = [1]
crossTraffic = ['tcppoisson']
#trafficParams = [[5, 2, 20], [5, 3, 16], [5, 4, 12], [5, 5, 8], [5, 6, 4], [5,7,0]]
#trafficParams = [[5, 0, 12], [5, 0, 18], [5, 0, 24], [5, 0, 30], [5,0,36], [5,0,42]]
#trafficParams = [[4, 1, 0], [3, 2, 0], [2, 3, 0], [1, 4, 0]]


def run(r, p, c, i, m, b, t0, t1, t2):
    runExp(name, bw, r, b, p, c, i, m, t0, t1, t2)

if __name__ == '__main__':
    for r in rtts:
        for p in pulseSizes:
            for c in crossTraffic:
                for i in initRates:
                    for m in measures:
                        for b in buf:
                            for t in trafficParams:
                           	    run(r, p, c, i, m, b, t[0], t[1], t[2])


