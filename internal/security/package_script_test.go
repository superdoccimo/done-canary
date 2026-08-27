package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageScriptInjectsAndVerifiesRequestedVersion(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "package.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)
	libraryData, err := os.ReadFile(filepath.Join(root, "scripts", "package-lib.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	combined := script + "\n" + string(libraryData)
	required := []string{
		`[string]$Version = "0.1.0-rc.3"`,
		`[string]$TargetOS = ""`,
		`[string]$TargetArch = ""`,
		`. (Join-Path $PSScriptRoot "package-lib.ps1")`,
		`[string]::IsNullOrWhiteSpace($Value)`,
		`$Value -match '[\s/\\]'`,
		`Release archives must be created on a matching operating system`,
		`$ldflags = "-X github.com/superdoccimo/done-canary/internal/model.Version=$Version"`,
		`& $go build -trimpath -ldflags $ldflags`,
		`$packageName = "done-canary-v$Version-$TargetOS-$TargetArch"`,
		`Assert-BinaryVersion -BinaryPath $probePath -ExpectedVersion $Version`,
		`& chmod 0755 $binaryPath`,
		`& tar -czf $archivePath`,
		`Compress-Archive`,
		`if ($actualVersion -ne $expectedOutput)`,
		`throw "Version mismatch`,
	}
	for _, fragment := range required {
		if !strings.Contains(combined, fragment) {
			t.Errorf("package.ps1 does not contain required packaging guard %q", fragment)
		}
	}
}

func TestReleaseManifestUsesNativeArchiveFormats(t *testing.T) {
	root := projectRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "release", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := string(data)
	required := []string{
		`"file": "done-canary-v0.1.0-rc.3-windows-amd64.zip"`,
		`"file": "done-canary-v0.1.0-rc.3-linux-amd64.tar.gz"`,
		`"file": "done-canary-v0.1.0-rc.3-darwin-amd64.tar.gz"`,
	}
	for _, fragment := range required {
		if !strings.Contains(manifest, fragment) {
			t.Errorf("release manifest is missing %q", fragment)
		}
	}
	if got := strings.Count(manifest, `"package_format": "tar.gz"`); got != 2 {
		t.Errorf("release manifest has %d tar.gz targets, want 2", got)
	}
}

func TestRC3VersionMetadataIsAligned(t *testing.T) {
	root := projectRoot(t)
	retiredVersion := "0.1.0-" + "rc.1"
	for _, relative := range []string{
		"CITATION.cff",
		filepath.Join("release", "manifest.json"),
		filepath.Join("examples", "result-7-of-7.json"),
	} {
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), retiredVersion) || !strings.Contains(string(data), "0.1.0-rc.3") {
			t.Errorf("%s does not consistently identify rc.3", relative)
		}
	}
}
