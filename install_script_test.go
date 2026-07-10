package andesai_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScriptSyntax(t *testing.T) {
	cmd := exec.Command("bash", "-n", "install.sh")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bash -n install.sh failed: %v\n%s", err, out)
	}
}

func TestInstallScriptHelp(t *testing.T) {
	cmd := exec.Command("bash", "install.sh", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --help failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "usage: install.sh [--version vX.Y.Z]") {
		t.Fatalf("help output = %q, want usage", out)
	}
}

func TestInstallScriptUnknownOptionFails(t *testing.T) {
	cmd := exec.Command("bash", "install.sh", "--wat")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("install.sh --wat succeeded, want failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown option: --wat") {
		t.Fatalf("unknown option output = %q", out)
	}
}

func TestInstallScriptDownloadsLatestViaGh(t *testing.T) {
	home := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "gh.args")
	fakeBin := fakeGh(t, argsFile)

	cmd := exec.Command("bash", "install.sh")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"GH_ARGS_FILE="+argsFile,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh with fake gh failed: %v\n%s", err, out)
	}

	installed := filepath.Join(home, ".local", "bin", "andes")
	info, err := os.Stat(installed)
	if err != nil {
		t.Fatalf("expected installed binary at %s: %v\n%s", installed, err, out)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed binary mode = %04o, want executable", info.Mode().Perm())
	}
	if !strings.Contains(string(out), "installed release binary (latest) via gh") {
		t.Fatalf("output = %q, want gh install success", out)
	}

	gotArgs := readFile(t, argsFile)
	if strings.Contains(gotArgs, " v") {
		t.Fatalf("latest download should not pass a version tag, got args %q", gotArgs)
	}
	if !strings.Contains(gotArgs, "--repo Papoosky/Andes-AI") {
		t.Fatalf("gh args = %q, want repo", gotArgs)
	}
	if !strings.Contains(gotArgs, "--pattern "+expectedAssetName()) {
		t.Fatalf("gh args = %q, want asset pattern %q", gotArgs, expectedAssetName())
	}
	if !strings.Contains(gotArgs, "--output "+installed) {
		t.Fatalf("gh args = %q, want output path %q", gotArgs, installed)
	}
}

func TestInstallScriptDownloadsPinnedVersionViaGh(t *testing.T) {
	home := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "gh.args")
	fakeBin := fakeGh(t, argsFile)

	cmd := exec.Command("bash", "install.sh", "--version", "v1.2.3")
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"GH_ARGS_FILE="+argsFile,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --version with fake gh failed: %v\n%s", err, out)
	}

	gotArgs := readFile(t, argsFile)
	if !strings.Contains(gotArgs, "release download v1.2.3") {
		t.Fatalf("gh args = %q, want pinned version", gotArgs)
	}
	if !strings.Contains(string(out), "installed release binary (v1.2.3) via gh") {
		t.Fatalf("output = %q, want pinned install success", out)
	}
}

func fakeGh(t *testing.T, argsFile string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gh")
	script := `#!/bin/sh
printf '%s\n' "$*" > "$GH_ARGS_FILE"
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output" ]; then
    shift
    out="$1"
  fi
  shift
done
if [ -z "$out" ]; then
  echo "missing --output" >&2
  exit 1
fi
mkdir -p "$(dirname "$out")"
printf '#!/bin/sh\n' > "$out"
exit 0
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func expectedAssetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "386" {
		arch = "386"
	}
	return "andes-" + osName + "-" + arch
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}
