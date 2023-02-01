#!/bin/bash -eu
#
# Copyright 2017 Google Inc. All rights reserved.
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

#
# This file is used in our continous build infrastructure to run a variety of
# tests related to the build system.
#
# Currently, it's used to build and run multiproduct_kati, so it'll attempt
# to build ninja files for every product in the tree. I expect this to
# evolve as we find interesting things to test or track performance for.
#

# Products that are broken or otherwise don't work with multiproduct_kati
SKIPPED_PRODUCTS=(
    # These products are for soong-only builds, and will fail the kati stage.
    linux_bionic
    mainline_sdk
    ndk

    # New architecture bringup, fails without ALLOW_MISSING_DEPENDENCIES=true
    aosp_riscv64
)

# To track how long we took to startup. %N isn't supported on Darwin, but
# that's detected in the Go code, which skips calculating the startup time.
export TRACE_BEGIN_SOONG=$(date +%s%N)

# Remove BUILD_NUMBER so that incremental builds on build servers don't
# re-read makefiles every time.
unset BUILD_NUMBER

export TOP=$(cd $(dirname ${BASH_SOURCE[0]})/../..; PWD= /bin/pwd)
cd "${TOP}"
source "${TOP}/build/soong/scripts/microfactory.bash"

case $(uname) in
  Linux)
    if [[ -f /lib/x86_64-linux-gnu/libSegFault.so ]]; then
      export LD_PRELOAD=/lib/x86_64-linux-gnu/libSegFault.so
      export SEGFAULT_USE_ALTSTACK=1
    fi
    ulimit -a
    ;;
esac

echo
echo "Free disk space:"
# Ignore df errors because it errors out on gvfsd file systems
# but still displays most of the useful info we need
df -h || true

echo
echo "Running Bazel smoke test..."
STANDALONE_BAZEL=true "${TOP}/build/bazel/bin/bazel" --batch --max_idle_secs=1 help

echo
echo "Running Soong test..."
soong_build_go multiproduct_kati android/soong/cmd/multiproduct_kati
exec "$(getoutdir)/multiproduct_kati" --skip-products "$(echo "${SKIPPED_PRODUCTS[@]-}" | tr ' ' ',')" "$@"
