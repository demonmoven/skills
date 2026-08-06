#!/usr/bin/env python3
import os
import sys
import shutil
import subprocess
import tempfile
from lit.main import main

def rootdir():
    directory = os.path.dirname(os.path.abspath(__file__))
    return os.path.dirname(os.path.dirname(directory))

def testdir():
    return os.path.join(rootdir(), 'test')

def exampledir():
    return os.path.join(testdir(), 'examples')

def build():
    root = os.path.abspath(rootdir())
    cleaner = os.path.join(root, 'cmd', 'cleaner')

    env = os.environ.copy()
    env.update({
        'GOROOT': subprocess.check_output(['go', 'env', 'GOROOT']).decode().strip(),
    })
    target = os.path.join(tempfile.gettempdir(), 'cleaner')
    proc = subprocess.run(f"go build -o {target}", shell=True, cwd=cleaner, env=env)
    return not proc.returncode

def create_new_test_dir():
    new_test_dir = os.path.join(tempfile.gettempdir(), 'cleaner-integration-tests')
    if os.path.exists(new_test_dir):
        shutil.rmtree(new_test_dir)
    shutil.copytree(testdir(), new_test_dir)
    shutil.copy(os.path.join(testdir(), 'util_lit', 'lit.cfg.py'), new_test_dir)
    return new_test_dir

if __name__ == '__main__':
    if build():
        print(create_new_test_dir())
    else:
        raise Exception("build cleaner failed")