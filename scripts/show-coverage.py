#!/usr/bin/env python3
import os
import subprocess
import sys
from collections import defaultdict

ERR = '\033[1;31m' # ]
OK = '\033[1;37m' # ]
END = '\033[0m' # ]

ignored_packages = [
    # /cmd/ sub-packages contain CLI implementation and so require some
    # additional infrastructure to be unit-tested.
    # TODO: implement unit tests for CLI commands
    'github.com/veraison/corim-store/cmd/corim-store/cmd',

    # Migrations are executed when constructed the test DB, and so are unit-tested
    # by tests outside the sub-package.
    'github.com/veraison/corim-store/pkg/migrations',

    # Test helpers only; does not contain production code.
    'github.com/veraison/corim-store/pkg/test',
]
ignored_packages.extend(os.getenv('IGNORE_COVERAGE', '').strip().split())

threshold = int(os.getenv('COVERAGE_THRESHOLD', '85').rstrip('%'))
print(f'COVERAGE_THRESHOLD: {OK}{threshold}%{END}')

result = subprocess.run(
    ['go', 'tool', 'cover', f'-func={sys.argv[1]}'],
    capture_output=True,
    text=True,
    check=True,
)

pkg_cov = defaultdict(float)
pkg_count = defaultdict(int)

for line in result.stdout.splitlines():
    if line.startswith('total:') or not line.strip():
        continue

    # Example line:
    # mypkg/foo/file.go:10:   MyFunc     85.7%
    parts = line.split()
    file_path = parts[0]
    coverage = float(parts[-1].strip('%'))

    file_path = file_path.split(':')[0]
    pkg = '/'.join(file_path.split('/')[:-1])

    pkg_cov[pkg] += coverage
    pkg_count[pkg] += 1

errors_found = False

for pkg in sorted(pkg_cov):

    if pkg in ignored_packages:
        coverage = 'ignored'
    else:
        start = OK
        avg = pkg_cov[pkg] / pkg_count[pkg]

        if avg < threshold:
            start = ERR
            errors_found = True

        coverage = f'{start}{avg:.1f}{END}%'
        coverage = f'{coverage:.>18}'

    print(f'{pkg:.<55}{coverage}')

if errors_found:
    sys.exit(1)
