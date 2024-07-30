# Copyright 2024 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import dataclasses
import glob
from importlib import resources
import logging
import os
from pathlib import Path
import re
import shutil
import signal
import stat
import subprocess
import sys
import tempfile
import textwrap
import time
import unittest

EXII_RETURN_CODE = 0
INTERRUPTED_RETURN_CODE = 130


class RunToolWithLoggingTest(unittest.TestCase):

  @classmethod
  def setUpClass(cls):
    super().setUpClass()
    # Configure to print logging to stdout.
    logging.basicConfig(filename=None, level=logging.DEBUG)
    console = logging.StreamHandler(sys.stdout)
    logging.getLogger("").addHandler(console)

  def setUp(self):
    super().setUp()
    self.working_dir = tempfile.TemporaryDirectory()
    # Run all the tests from working_dir which is our temp Android build top.
    os.chdir(self.working_dir.name)

    self.logging_script_path = self._import_executable("run_tool_with_logging")

  def tearDown(self):
    self.working_dir.cleanup()
    super().tearDown()

  def test_does_not_log_when_logger_var_empty(self):
    test_tool = TestScript.create(self.working_dir)

    self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER=""
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    test_tool.assert_called_once_with_args("arg1 arg2")

  def test_does_not_log_with_logger_unset(self):
    test_tool = TestScript.create(self.working_dir)

    self._run_script_and_wait(f"""
      unset ANDROID_TOOL_LOGGER
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    test_tool.assert_called_once_with_args("arg1 arg2")

  def test_log_success_with_logger_enabled(self):
    test_tool = TestScript.create(self.working_dir)
    test_logger = TestScript.create(self.working_dir)

    self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    test_tool.assert_called_once_with_args("arg1 arg2")
    expected_logger_args = (
        "--tool_tag=FAKE_TOOL --start_timestamp=\d+\.\d+ --end_timestamp="
        "\d+\.\d+ --tool_args=arg1 arg2 --exit_code=0"
    )
    test_logger.assert_called_once_with_args(expected_logger_args)

  def test_run_tool_output_is_same_with_and_without_logging(self):
    test_tool = TestScript.create(self.working_dir, "echo 'tool called'")
    test_logger = TestScript.create(self.working_dir)

    run_tool_with_logging_stdout, run_tool_with_logging_stderr = (
        self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)
    )

    run_tool_without_logging_stdout, run_tool_without_logging_stderr = (
        self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {test_tool.executable} arg1 arg2
    """)
    )

    self.assertEqual(
        run_tool_with_logging_stdout, run_tool_without_logging_stdout
    )
    self.assertEqual(
        run_tool_with_logging_stderr, run_tool_without_logging_stderr
    )

  def test_logger_output_is_suppressed(self):
    test_tool = TestScript.create(self.working_dir)
    test_logger = TestScript.create(self.working_dir, "echo 'logger called'")

    run_tool_with_logging_output, _ = self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    self.assertNotIn("logger called", run_tool_with_logging_output)

  def test_logger_error_is_suppressed(self):
    test_tool = TestScript.create(self.working_dir)
    test_logger = TestScript.create(
        self.working_dir, "echo 'logger failed' > /dev/stderr; exit 1"
    )

    _, err = self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    self.assertNotIn("logger failed", err)

  def test_log_success_when_tool_interrupted(self):
    test_tool = TestScript.create(self.working_dir, script_body="sleep 100")
    test_logger = TestScript.create(self.working_dir)

    process = self._run_script(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    pgid = os.getpgid(process.pid)
    # Give sometime for the subprocess to start.
    time.sleep(1)
    # Kill the subprocess and any processes created in the same group.
    os.killpg(pgid, signal.SIGINT)

    returncode, _, _ = self._wait_for_process(process)
    self.assertEqual(returncode, INTERRUPTED_RETURN_CODE)

    expected_logger_args = (
        "--tool_tag=FAKE_TOOL --start_timestamp=\d+\.\d+ --end_timestamp="
        "\d+\.\d+ --tool_args=arg1 arg2 --exit_code=130"
    )
    test_logger.assert_called_once_with_args(expected_logger_args)

  def test_logger_can_be_toggled_on(self):
    test_tool = TestScript.create(self.working_dir)
    test_logger = TestScript.create(self.working_dir)

    self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER=""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    test_logger.assert_called_with_times(1)

  def test_logger_can_be_toggled_off(self):
    test_tool = TestScript.create(self.working_dir)
    test_logger = TestScript.create(self.working_dir)

    self._run_script_and_wait(f"""
      export ANDROID_TOOL_LOGGER="{test_logger.executable}"
      export ANDROID_TOOL_LOGGER=""
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    test_logger.assert_not_called()

  def test_integration_tool_event_logger_dry_run(self):
    test_tool = TestScript.create(self.working_dir)
    logger_path = self._import_executable("tool_event_logger")

    self._run_script_and_wait(f"""
      export TMPDIR="{self.working_dir.name}"
      export ANDROID_TOOL_LOGGER="{logger_path}"
      export ANDROID_TOOL_LOGGER_EXTRA_ARGS="--dry_run"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} arg1 arg2
    """)

    self._assert_logger_dry_run()

  def test_tool_args_do_not_fail_logger(self):
    test_tool = TestScript.create(self.working_dir)
    logger_path = self._import_executable("tool_event_logger")

    self._run_script_and_wait(f"""
      export TMPDIR="{self.working_dir.name}"
      export ANDROID_TOOL_LOGGER="{logger_path}"
      export ANDROID_TOOL_LOGGER_EXTRA_ARGS="--dry_run"
      {self.logging_script_path} "FAKE_TOOL" {test_tool.executable} --tool-arg1
    """)

    self._assert_logger_dry_run()

  def _import_executable(self, executable_name: str) -> Path:
    # logger = "tool_event_logger"
    executable_path = Path(self.working_dir.name).joinpath(executable_name)
    with resources.as_file(
        resources.files("testdata").joinpath(executable_name)
    ) as p:
      shutil.copy(p, executable_path)
    Path.chmod(executable_path, 0o755)
    return executable_path

  def _assert_logger_dry_run(self):
    log_files = glob.glob(self.working_dir.name + "/tool_event_logger_*/*.log")
    self.assertEqual(len(log_files), 1)

    with open(log_files[0], "r") as f:
      lines = f.readlines()
      self.assertEqual(len(lines), 1)
      self.assertIn("dry run", lines[0])

  def _run_script_and_wait(self, test_script: str) -> tuple[str, str]:
    process = self._run_script(test_script)
    returncode, out, err = self._wait_for_process(process)
    logging.debug("script stdout: %s", out)
    logging.debug("script stderr: %s", err)
    self.assertEqual(returncode, EXII_RETURN_CODE)
    return out, err

  def _run_script(self, test_script: str) -> subprocess.Popen:
    return subprocess.Popen(
        test_script,
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        start_new_session=True,
        executable="/bin/bash",
    )

  def _wait_for_process(
      self, process: subprocess.Popen
  ) -> tuple[int, str, str]:
    pgid = os.getpgid(process.pid)
    out, err = process.communicate()
    # Wait for all process in the same group to complete since the logger runs
    # as a separate detached process.
    self._wait_for_process_group(pgid)
    return (process.returncode, out, err)

  def _wait_for_process_group(self, pgid: int, timeout: int = 5):
    """Waits for all subprocesses within the process group to complete."""
    start_time = time.time()
    while True:
      if time.time() - start_time > timeout:
        raise TimeoutError(
            f"Process group did not complete after {timeout} seconds"
        )
      for pid in os.listdir("/proc"):
        if pid.isdigit():
          try:
            if os.getpgid(int(pid)) == pgid:
              time.sleep(0.1)
              break
          except (FileNotFoundError, PermissionError, ProcessLookupError):
            pass
      else:
        # All processes have completed.
        break


@dataclasses.dataclass
class TestScript:
  executable: Path
  output_file: Path

  def create(temp_dir: Path, script_body: str = ""):
    with tempfile.NamedTemporaryFile(dir=temp_dir.name, delete=False) as f:
      output_file = f.name

    with tempfile.NamedTemporaryFile(dir=temp_dir.name, delete=False) as f:
      executable = f.name
      executable_contents = textwrap.dedent(f"""
      #!/bin/bash

      echo "${{@}}" >> {output_file}
      {script_body}
      """)
      f.write(executable_contents.encode("utf-8"))

    Path.chmod(f.name, os.stat(f.name).st_mode | stat.S_IEXEC)

    return TestScript(executable, output_file)

  def assert_called_with_times(self, expected_call_times: int):
    lines = self._read_contents_from_output_file()
    assert len(lines) == expected_call_times, (
        f"Expect to call {expected_call_times} times, but actually called"
        f" {len(lines)} times."
    )

  def assert_called_with_args(self, expected_args: str):
    lines = self._read_contents_from_output_file()
    assert len(lines) > 0
    assert re.search(expected_args, lines[0]), (
        f"Expect to call with args {expected_args}, but actually called with"
        f" args {lines[0]}."
    )

  def assert_not_called(self):
    self.assert_called_with_times(0)

  def assert_called_once_with_args(self, expected_args: str):
    self.assert_called_with_times(1)
    self.assert_called_with_args(expected_args)

  def _read_contents_from_output_file(self) -> list[str]:
    with open(self.output_file, "r") as f:
      return f.readlines()


if __name__ == "__main__":
  unittest.main()
