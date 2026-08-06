#!/usr/bin/env python3

import os
import sys
import json

with open(sys.argv[1], 'r' ) as fp:
    data = json.load(fp)

for result in data:
    if result['has_error'] == 0:
        print(result['repo'])