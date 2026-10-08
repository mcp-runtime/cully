"""Exercise the installer without network access or real agent configuration."""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


INSTALLER = Path(__file__).resolve().parents[1] / "install.sh"


class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.cli = self.root / "cully"
        self.cli.write_text('''#!/bin/sh
case "$1" in
  version) echo v-test ;;
  _internal) printf '%s\\n' "$*" > "$TEST_ROOT/daemon-call" ;;
  setup) printf '%s\\n' "$*" > "$TEST_ROOT/setup"; printf '%s\\n' "$PATH" > "$TEST_ROOT/setup-path" ;;
esac
''')
        self.cli.chmod(0o755)
        with tarfile.open(self.root / "archive.tar.gz", "w:gz") as archive:
            content = self.cli.read_bytes()
            entry = tarfile.TarInfo("cully")
            entry.size = len(content)
            entry.mode = 0o755
            archive.addfile(entry, io.BytesIO(content))
        digest = hashlib.sha256((self.root / "archive.tar.gz").read_bytes()).hexdigest()
        (self.root / "checksums.txt").write_text(f"{digest}  cully_linux_amd64.tar.gz\n")
        self.stub("uname", 'case "$1" in -s) echo Linux ;; -m) echo x86_64 ;; esac')
        self.stub("curl", '''
printf '%s\\n' "$*" >> "$TEST_ROOT/curl-calls"
[ "${DOWNLOAD_FAIL:-0}" = 0 ] || exit 28
source_file=archive.tar.gz
while [ "$#" -gt 0 ]; do
  case "$1" in
    */checksums.txt) source_file=checksums.txt ;;
    -o) shift; destination="$1" ;;
  esac
  shift
done
cp "$TEST_ROOT/$source_file" "$destination"
''')
        self.stub("go", '''
printf '%s\\n' "$*" > "$TEST_ROOT/go-call"
cp "$TEST_ROOT/cully" "$GOBIN/cully"
''')
        self.env = dict(os.environ, HOME=str(self.root), TEST_ROOT=str(self.root),
                        CLAUDE_CONFIG_DIR=str(self.root / "claude"),
                        CODEX_HOME=str(self.root / "codex"),
                        CURSOR_CONFIG_DIR=str(self.root / "cursor"),
                        PATH=str(self.bin) + os.pathsep + os.environ["PATH"])

    def stub(self, name, body):
        path = self.bin / name
        path.write_text("#!/bin/sh\nset -eu\n" + body + "\n")
        path.chmod(0o755)

    def run_installer(self, *args):
        return subprocess.run(["sh", str(INSTALLER), *args], env=self.env,
                              capture_output=True, text=True, timeout=10)

    def test_binary_install_forwards_agent_and_mcp_options(self):
        result = self.run_installer("--agent", "codex", "--mcp-url",
                                    "https://example.com/mcp", "--oauth")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.root / "setup").read_text().strip(),
                         "setup --mcp-url https://example.com/mcp --agent codex --oauth")
        self.assertFalse((self.root / "go-call").exists())
        calls = (self.root / "curl-calls").read_text()
        self.assertIn("--progress-bar --connect-timeout 10 --max-time 120", calls)
        self.assertIn("--speed-limit 1024 --speed-time 30", calls)
        self.assertIn("--max-time 30", calls)

    def test_failed_download_does_not_compile_or_install(self):
        self.env["DOWNLOAD_FAIL"] = "1"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("--from-source", result.stderr)
        self.assertFalse((self.root / "go-call").exists())
        self.assertFalse((self.root / ".local" / "bin" / "cully").exists())

    def test_explicit_source_install_uses_selected_ref(self):
        for version, ref in [("latest", "main"), ("v0.3.0", "v0.3.0")]:
            with self.subTest(version=version):
                self.env["CULLY_VERSION"] = version
                result = self.run_installer("--from-source", "--agent", "cursor")
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn("several minutes", result.stdout)
                self.assertEqual((self.root / "go-call").read_text().strip(),
                                 f"install -v github.com/mcp-runtime/cully/cmd/cully@{ref}")
                self.assertFalse((self.root / "curl-calls").exists())

    def test_checksum_mismatch_aborts_install(self):
        (self.root / "checksums.txt").write_text("bad  cully_linux_amd64.tar.gz\n")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum mismatch", result.stderr)
        self.assertFalse((self.root / ".local" / "bin" / "cully").exists())

    def test_agent_binary_and_path_follow_selected_agent(self):
        for agent, directory in [("claude", ".claude"),
                                 ("codex", ".codex"),
                                 ("cursor", ".cursor"),
                                 ("all", ".local"),
                                 (None, ".local")]:
            with self.subTest(agent=agent):
                self.env.pop("CLAUDE_CONFIG_DIR", None)
                self.env.pop("CODEX_HOME", None)
                self.env.pop("CURSOR_CONFIG_DIR", None)
                self.env["SHELL"] = "/bin/zsh"
                args = ("--agent", agent) if agent else ()
                result = self.run_installer(*args)
                self.assertEqual(result.returncode, 0, result.stderr)
                installed = self.root / directory / "bin" / "cully"
                self.assertTrue(installed.is_file())
                self.assertEqual((self.root / "daemon-call").read_text().strip(),
                                 "_internal stop-daemon")
                self.assertFalse((self.root / "setup").exists())
                self.assertIn("Start Docker, then run: cully setup", result.stdout)
                if agent:
                    self.assertIn(f"cully setup --agent {agent}", result.stdout)
                path_line = f'export PATH="$HOME/{directory}/bin:$PATH"'
                self.assertEqual((self.root / ".zshrc").read_text().count(path_line), 1)

    def test_path_line_is_idempotent_and_preserves_shell_config(self):
        self.env.pop("CODEX_HOME")
        self.env["SHELL"] = "/bin/zsh"
        profile = self.root / ".zshrc"
        profile.write_text("# my own settings\n")
        for _ in range(2):
            result = self.run_installer("--agent", "codex")
            self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(profile.read_text().count('export PATH="$HOME/.codex/bin:$PATH"'), 1)
        self.assertTrue(profile.read_text().startswith("# my own settings\n"))

    def test_selected_agent_binary_overrides_older_claude_install(self):
        self.env.pop("CODEX_HOME")
        self.env["SHELL"] = "/bin/zsh"
        old_bin = self.root / ".claude" / "bin"
        selected_bin = self.root / ".codex" / "bin"
        old_bin.mkdir(parents=True)
        selected_bin.mkdir(parents=True)
        old_cli = old_bin / "cully"
        old_cli.write_text("#!/bin/sh\necho old\n")
        old_cli.chmod(0o755)
        self.env["PATH"] = os.pathsep.join((str(old_bin), str(selected_bin), self.env["PATH"]))
        result = self.run_installer("--agent", "codex", "--mcp-url", "https://example.com/mcp")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.root / "setup-path").read_text().split(os.pathsep)[0],
                         str(selected_bin))
        self.assertIn('export PATH="$HOME/.codex/bin:$PATH"',
                      (self.root / ".zshrc").read_text())


if __name__ == "__main__":
    unittest.main()
