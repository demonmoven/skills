import os
import subprocess
import tempfile
import lit.formats

config.name = "GoCleaner"
config.test_format = lit.formats.ShTest(True)

config.suffixes = ['.go']
config.excludes = []
parent = os.path.dirname(__file__)

for _, _, filenames in os.walk(parent):
    config.excludes.extend([fn for fn in filenames if not fn.startswith('check')])

config.substitutions.append(('%cleaner', os.path.join(tempfile.gettempdir(), 'cleaner')))
config.substitutions.append(('%filecheck', 'filecheck'))

# This code will update the PATH environment variable at runtime to 
# ensure that the system uses correct Go version set by Cloud IDE
goroot = subprocess.check_output(['go', 'env', 'GOROOT']).decode().strip()
config.environment['PATH'] = f"{os.path.join(goroot, 'bin')}:{os.environ['PATH']}"
config.environment['HOME'] = os.environ.get('HOME', '/tmp')
config.environment['FOR_CLEANER_INTEGRATION_TEST'] = '1'