#!/bin/bash -eu

set -o pipefail

# Test that bp2build and Bazel can play nicely together

source "$(dirname "$0")/lib.sh"

readonly GENERATED_BUILD_FILE_NAME="BUILD.bazel"

function test_bp2build_null_build {
  setup
  run_soong bp2build
  local -r output_mtime1=$(stat -c "%y" out/soong/bp2build_workspace_marker)

  run_soong bp2build
  local -r output_mtime2=$(stat -c "%y" out/soong/bp2build_workspace_marker)

  if [[ "$output_mtime1" != "$output_mtime2" ]]; then
    fail "Output bp2build marker file changed on null build"
  fi
}

function test_bp2build_null_build_with_globs {
  setup

  mkdir -p foo/bar
  cat > foo/bar/Android.bp <<'EOF'
filegroup {
    name: "globs",
    srcs: ["*.txt"],
  }
EOF
  touch foo/bar/a.txt foo/bar/b.txt

  run_soong bp2build
  local -r output_mtime1=$(stat -c "%y" out/soong/bp2build_workspace_marker)

  run_soong bp2build
  local -r output_mtime2=$(stat -c "%y" out/soong/bp2build_workspace_marker)

  if [[ "$output_mtime1" != "$output_mtime2" ]]; then
    fail "Output bp2build marker file changed on null build"
  fi
}

function test_different_relative_outdir {
  setup

  mkdir -p a
  touch a/g.txt
  cat > a/Android.bp <<'EOF'
filegroup {
    name: "g",
    srcs: ["g.txt"],
    bazel_module: {bp2build_available: true},
  }
EOF

  # A directory under $MOCK_TOP
  outdir=out2
  trap "rm -rf $outdir" EXIT
  # Modify OUT_DIR in a subshell so it doesn't affect the top level one.
  (export OUT_DIR=$outdir; run_soong bp2build && run_bazel build --config=bp2build --config=ci //a:g)
}

function test_different_absolute_outdir {
  setup

  mkdir -p a
  touch a/g.txt
  cat > a/Android.bp <<'EOF'
filegroup {
    name: "g",
    srcs: ["g.txt"],
    bazel_module: {bp2build_available: true},
  }
EOF

  # A directory under /tmp/...
  outdir=$(mktemp -t -d st.XXXXX)
  trap 'rm -rf $outdir' EXIT
  # Modify OUT_DIR in a subshell so it doesn't affect the top level one.
  (export OUT_DIR=$outdir; run_soong bp2build && run_bazel build --config=bp2build --config=ci //a:g)
}

function _bp2build_generates_all_buildfiles {
  setup

  mkdir -p foo/convertible_soong_module
  cat > foo/convertible_soong_module/Android.bp <<'EOF'
genrule {
    name: "the_answer",
    cmd: "echo '42' > $(out)",
    out: [
        "the_answer.txt",
    ],
    bazel_module: {
        bp2build_available: true,
    },
  }
EOF

  mkdir -p foo/unconvertible_soong_module
  cat > foo/unconvertible_soong_module/Android.bp <<'EOF'
genrule {
    name: "not_the_answer",
    cmd: "echo '43' > $(out)",
    out: [
        "not_the_answer.txt",
    ],
    bazel_module: {
        bp2build_available: false,
    },
  }
EOF

  run_soong bp2build

  if [[ ! -f "./out/soong/workspace/foo/convertible_soong_module/${GENERATED_BUILD_FILE_NAME}" ]]; then
    fail "./out/soong/workspace/foo/convertible_soong_module/${GENERATED_BUILD_FILE_NAME} was not generated"
  fi

  if [[ ! -f "./out/soong/workspace/foo/unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME}" ]]; then
    fail "./out/soong/workspace/foo/unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME} was not generated"
  fi

  if ! grep "the_answer" "./out/soong/workspace/foo/convertible_soong_module/${GENERATED_BUILD_FILE_NAME}"; then
    fail "missing BUILD target the_answer in convertible_soong_module/${GENERATED_BUILD_FILE_NAME}"
  fi

  if grep "not_the_answer" "./out/soong/workspace/foo/unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME}"; then
    fail "found unexpected BUILD target not_the_answer in unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME}"
  fi

  if ! grep "filegroup" "./out/soong/workspace/foo/unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME}"; then
    fail "missing filegroup in unconvertible_soong_module/${GENERATED_BUILD_FILE_NAME}"
  fi

  # NOTE: We don't actually use the extra BUILD file for anything here
  run_bazel build --config=android --config=bp2build --config=ci //foo/...

  local the_answer_file="$(find -L bazel-out -name the_answer.txt)"
  if [[ ! -f "${the_answer_file}" ]]; then
    fail "Expected the_answer.txt to be generated, but was missing"
  fi
  if ! grep 42 "${the_answer_file}"; then
    fail "Expected to find 42 in '${the_answer_file}'"
  fi
}

function test_bp2build_generates_all_buildfiles {
  _save_trap=$(trap -p EXIT)
  trap '[[ $? -ne 0 ]] && echo Are you running this locally? Try changing --sandbox_tmpfs_path to something other than /tmp/ in build/bazel/linux.bazelrc.' EXIT
  _bp2build_generates_all_buildfiles
  eval "${_save_trap}"
}

function test_cc_correctness {
  setup

  mkdir -p a
  cat > a/Android.bp <<EOF
cc_object {
  name: "qq",
  srcs: ["qq.cc"],
  bazel_module: {
    bp2build_available: true,
  },
  stl: "none",
  system_shared_libs: [],
}
EOF

  cat > a/qq.cc <<EOF
#include "qq.h"
int qq() {
  return QQ;
}
EOF

  cat > a/qq.h <<EOF
#define QQ 1
EOF

  run_soong bp2build

  run_bazel build --config=android --config=bp2build --config=ci //a:qq
  local -r output_mtime1=$(stat -c "%y" bazel-bin/a/_objs/qq/qq.o)

  run_bazel build --config=android --config=bp2build --config=ci //a:qq
  local -r output_mtime2=$(stat -c "%y" bazel-bin/a/_objs/qq/qq.o)

  if [[ "$output_mtime1" != "$output_mtime2" ]]; then
    fail "output changed on null build"
  fi

  cat > a/qq.h <<EOF
#define QQ 2
EOF

  run_bazel build --config=android --config=bp2build --config=ci //a:qq
  local -r output_mtime3=$(stat -c "%y" bazel-bin/a/_objs/qq/qq.o)

  if [[ "$output_mtime1" == "$output_mtime3" ]]; then
    fail "output not changed when included header changed"
  fi
}

# Regression test for the following failure during symlink forest creation:
#
#   Cannot stat '/tmp/st.rr054/foo/bar/unresolved_symlink': stat /tmp/st.rr054/foo/bar/unresolved_symlink: no such file or directory
#
function test_bp2build_null_build_with_unresolved_symlink_in_source() {
  setup

  mkdir -p foo/bar
  ln -s /tmp/non-existent foo/bar/unresolved_symlink
  cat > foo/bar/Android.bp <<'EOF'
filegroup {
    name: "fg",
    srcs: ["unresolved_symlink/non-existent-file.txt"],
  }
EOF

  run_soong bp2build

  dest=$(readlink -f out/soong/workspace/foo/bar/unresolved_symlink)
  if [[ "$dest" != "/tmp/non-existent" ]]; then
    fail "expected to plant an unresolved symlink out/soong/workspace/foo/bar/unresolved_symlink that resolves to /tmp/non-existent"
  fi
}

# Smoke test to verify api_bp2build worksapce does not contain any errors
function test_api_bp2build_empty_build() {
  setup
  run_soong api_bp2build
  run_bazel build --config=android --config=api_bp2build //:empty
}

scan_and_run_tests
