#!/usr/bin/env python3
import os
import sys
import argparse
import subprocess
from util_smoker.testcase import TestCase
from util_smoker.benchmark import Benchmark

SmokeTestPath = os.path.join(os.path.dirname(__file__), 'repositories')

def main(argv):
    get_bench_path = lambda bench: os.path.join(SmokeTestPath, f'{bench}Tests.json')
    defaultTestBench = Benchmark(get_bench_path('Smoke'))

    parser = argparse.ArgumentParser()
    allversions = ','.join(defaultTestBench.get_all_version())
    parser.add_argument('-v', '--versions', help=f'Go versions to test, default is {allversions}', default=allversions)
    parser.add_argument('--debug', action='store_true', help='Debug mode, just print the commands without executing them')
    parser.add_argument('-b', '--benchmark', help=f'Select benchmark, default is "Smoke"', default='Smoke')
    parser.add_argument('-p', '--parallel', action='store_true', help='Run benchmark in parallel')
    parser.add_argument('-o', '--output', help=f'Output summary result into file', default='')
    parser.add_argument('-t', '--targets', help=f'Interesting cases to run in this benchmark', type=lambda s: s.split(','), default=[])

    args, extra_flags = parser.parse_known_args(argv)
    
    if args.benchmark != 'Smoke':
        defaultTestBench = Benchmark(get_bench_path(args.benchmark))
        allversions = ','.join(defaultTestBench.get_all_version())
    
    if args.targets:
        defaultTestBench.set_target_cases(args.targets)
        allversions = ','.join(defaultTestBench.get_all_version())
    
    versions = [v for v in args.versions.split(',') if v in allversions]
    if not args.parallel: 
        returncode = defaultTestBench.run_bench(versions, args.debug, extra_flags[1:])
    else:
        returncode = defaultTestBench.run_bench_parallel(versions, args.debug, extra_flags[1:])
    
    if args.output:
        defaultTestBench.write_summary(args.output)

    return returncode

if __name__ == '__main__':
    main(sys.argv)