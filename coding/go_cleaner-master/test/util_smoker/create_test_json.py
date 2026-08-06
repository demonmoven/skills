#!/usr/bin/env python3
import os
import sys
import json

txtpath = os.path.abspath(sys.argv[1])
basename = os.path.basename(txtpath).split('.')[0]

with open(txtpath, 'r') as f:
    data = f.readlines()

tests = {}
results = []
for line in data:
    if '\t' not in line:
        continue

    version, repo = line.split('\t')
    if version in ['go', 'Go']:
        continue
    
    version = version[2:]
    repo = repo.strip()

    if repo in tests and tests[repo]['compiler_version'] > version:
        continue

    business = os.path.basename(os.path.dirname(txtpath))
    business = business[0].upper() + business[1:]

    tests[repo] = {
        "repo": repo,
        "compiler_version": version,
        "goals": [
            "Fix Crash"
        ],
        "head": "master",
        "flags": "",
        "build_command": "",
        "business": business
    }

for v in tests.values():
    results.append(v)

output = os.path.join(os.path.dirname(txtpath), basename + 'Tests.json')
with open(output, 'w') as f:
    json.dump(results, f, indent=4)