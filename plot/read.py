#!/usr/bin/python

import sys
import re

def readTrace(fn):
    with open(fn, 'r') as f:
        for line in f:
            yield readLine(line)

def readLine(line):
    res = {}
    matches = re.findall(r'(.+?)=("(.+?)"|(.+?)) ', line)
    for m in matches:
        k, v = parseField(m)
        res[k] = v   
    return res

def parseField(match):
    k = match[0]
    v = match[3]
    if match[1][0] == '"':
        v = match[2]
        return k, v

    # try parsing as time
    t = parseTime(v)
    if t is not None:
        return k, parseTime(v)

    # try parsing as float
    try:
        return k, float(v)
    except:
        # just parse as string
        return k, v

def parseTime(t):
    matches = re.findall(r'([0-9]+m)?([0-9]+?\.?[0-9]*)s|([0-9]+?\.?[0-9]*)ms', t)
    if len(matches) == 0:
        #microseconds
        matches = re.findall(r'([0-9]+?\.?[0-9]+).+s', t)
        if len(matches) == 0:
            if t == '0s':
                return 0
            else:
                return None
        usecs_m = matches[0]
        return float(usecs_m) * 1e-6
    mnts_m, secs_m, mses_m = matches[0]

    if mses_m != '':
        return float(mses_m) * 1e-3
    mnts = 0
    secs = 0
    if mnts_m != '':
        mnts = float(mnts_m[:-1]) * 60
    if secs != '':
        secs = float(secs_m)
    return mnts + secs

if __name__ == '__main__':
    for l in readTrace(sys.argv[1]):
        if len(sys.argv) == 2:
            print l
        else:
            if all(a in l for a in sys.argv[2:]):
                m = map(lambda a: str(l[a]), sys.argv[2:])
                print ' '.join(m)
