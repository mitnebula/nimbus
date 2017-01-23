#!/usr/bin/python

import sys
import re

def parseTime(t):
    matches = re.findall(r'([0-9]+m)?([0-9]+?\.?[0-9]+)s|([0-9]+\.?[0-9]+)ms', t)
    if len(matches) == 0:
        #microseconds/
        matches = re.findall(r'([0-9]+?\.?[0-9]+).*s', t)
        if len(matches) == 0:
            if t == '0s':
                return 0
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

def parseTptOutput(sp):
    t, _, rin, rout, rtt, _, mode = sp
    ret = {
        'time': parseTime(t),
        'rin': float(rin),
        'rout': float(rout),
        'rtt': parseTime(rtt),
        'mode': mode,
    }
    return ret

def parseSwitchOutput(sp):
    if len(sp) > 5:
        sp = sp[:5]
    print sp
    t, _, fr, _, to = sp
    return {
        'time': parseTime(t),
        'from': fr,
        'to': to,
    }

def readNimbusLines(f):
    for line in f:
        sp = line.split()
        if sp[0] == 'Received:':
            yield {
                'rout': float(sp[2][:-1]),
            }

        if len(sp) < 2 or sp[1] != ':':
            continue

        if len(sp) == 7:
            yield parseTptOutput(sp)
        elif len(sp) == 5 and sp[3] == '->':
            yield parseSwitchOutput(sp)

def readIperfLines(f):
    ls = f.readlines()

    # first line is start time
    for l in ls[:-1]:
        matches = re.findall(r'-\s?([0-9]+\.0) sec .*\s([0-9]+\.[0-9]+) Mbits', l)
        if len(matches) != 1:
            continue
        t, bw = matches[0]

        yield {
            'time': float(t),
            'tpt': float(bw),
        }
