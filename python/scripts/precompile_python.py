#!/usr/bin/env python3
# Copyright 2023 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import py_compile
import os
import shutil
import tempfile
import zipfile

# This file needs to support both python 2 and 3.


def process_one_file(name, inf, outzip):
    if not name.endswith('.py'):
        outzip.writestr(name, inf.read())
        return

    # Unfortunately py_compile requires the input/output files to be written
    # out to disk.
    with tempfile.NamedTemporaryFile(prefix="Soong_precompile_", delete=False) as tmp:
        shutil.copyfileobj(inf, tmp)
        in_name = tmp.name
    with tempfile.NamedTemporaryFile(prefix="Soong_precompile_", delete=False) as tmp:
        out_name = tmp.name
    try:
        py_compile.compile(in_name, out_name, name, doraise=True)
        with open(out_name, 'rb') as f:
            outzip.writestr(name + 'c', f.read())
    finally:
        os.remove(in_name)
        os.remove(out_name)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('src_zip')
    parser.add_argument('dst_zip')
    args = parser.parse_args()

    with open(args.dst_zip, 'wb') as outf, open(args.src_zip, 'rb') as inf:
        with zipfile.ZipFile(outf, mode='w') as outzip, zipfile.ZipFile(inf, mode='r') as inzip:
            for name in inzip.namelist():
                with inzip.open(name, mode='r') as inzipf:
                    process_one_file(name, inzipf, outzip)


if __name__ == "__main__":
    main()
