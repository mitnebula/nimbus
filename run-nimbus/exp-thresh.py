#!/usr/bin/python

import subprocess
from exp import runExp

name = 't10'
bw = 48
buf = [1, 2]

pulseSizes = [0.25]#, 0.5, 0.75]
rtts = [50, 100]
initRates = [24]#, 12, 20]
measures = [1]
crossTraffic = ['tcp', 'poisson', 'tcppoisson']

def run(r, p, c, i, m, b):
    runExp(name, bw, r, b, p, c, i, m)

if __name__ == '__main__':
    for r in rtts:
        for p in pulseSizes:
            for c in crossTraffic:
                for i in initRates:
                    for m in measures:
                        for b in buf:
                            run(r, p, c, i, m, b)


