#!/usr/bin/python

import subprocess
from exp import runExp

name = 't10'
bw = 48
buf = [1]

pulseSizes = [0.25]#, 0.5, 0.75]
rtts = [100]
switching = ['t']
initRates = [24]#, 12, 20]
measures = [1]
crossTraffic = ['tcppoisson']
#trafficParams = [[4 , 2, 24, 'XTCP'], [4, 3, 20, 'XTCP'], [4, 4, 16, 'XTCP'], [4, 5, 12, 'XTCP'], [4, 6, 8, 'XTCP'], [4, 7, 4, 'XTCP']]
#trafficParams = [[4, 0, 12, 'DELAY'], [4, 0, 18, 'DELAY'], [4, 0, 24, 'DELAY'], [4, 0, 30, 'DELAY'], [4,0,36, 'DELAY'], [4,0,42, 'DELAY']]
trafficParams = [[4, 1, 0, 'XTCP'], [4, 2, 0, 'XTCP'], [4, 4, 0, 'XTCP'], [4, 8, 0, 'XTCP'], [4 , 12, 0, 'XTCP']]


def run(r, p, c, i, m, b, t0, t1, t2, t3, s):
    runExp(name, bw, r, b, p, c, i, m, t0, t1, t2, t3, s)

if __name__ == '__main__':
    for r in rtts:
        for p in pulseSizes:
            for c in crossTraffic:
                for i in initRates:
                    for m in measures:
                        for b in buf:
                            for t in trafficParams:
                            	for s in switching:
                           	    	run(r, p, c, i, m, b, t[0], t[1], t[2], t[3], s)


