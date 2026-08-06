#!/usr/bin/env python3
import os
import sys
import json

def interesting(line):
    patterns = [
        "Resolving deltas:",
        "Receiving objects:",
        "remote: Compressing objects:",
        "remote: Counting objects:",
        "Updating files:",
        "Skip package: ",
    ]

    for p in patterns:
        if line.count(p):
            return False

    return True

def main(filename):
    with open(filename, 'r' ) as fp:
        data = json.load(fp)

    for entry in data:
        lines = []
        for m in entry['messages']:
            lines.extend(m.splitlines())
        if not entry['has_error']:
            entry['messages'] = []
            continue
            
        lines = [l for l in lines if interesting(l)]
        entry['messages'] = lines

    basename = os.path.basename(filename).split('.')[0]
    fmtbasename = '.'.join([basename, 'format', 'json'])
    newfile = os.path.join(os.path.dirname(filename), fmtbasename)

    with open(newfile, 'w') as fp:
        json.dump(data, fp, indent=4)

if __name__ == '__main__':
    main(sys.argv[1])