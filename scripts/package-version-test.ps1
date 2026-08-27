$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "package-lib.ps1")

foreach ($invalid in @("", "bad`nversion", "bad/version", "bad\version")) {
    $rejected = $false
    try {
        Assert-ValidVersion $invalid
    }
    catch {
        $rejected = $true
    }
    if (-not $rejected) {
        throw "Invalid version was accepted: '$invalid'"
    }
}

$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$testRoot = Join-Path $projectRoot "bin\package-version-test"
New-Item -ItemType Directory -Force -Path $testRoot | Out-Null
$go = (Get-Command go -ErrorAction Stop).Source
$binaryName = if ((& $go env GOOS).Trim() -eq "windows") { "done-canary.exe" } else { "done-canary" }

$developmentBinary = Join-Path $testRoot ("development-" + $binaryName)
& $go build -trimpath -o $developmentBinary ./cmd/done-canary
if ($LASTEXITCODE -ne 0) { throw "Development build failed" }
Assert-BinaryVersion -BinaryPath $developmentBinary -ExpectedVersion "0.1.0-dev"

$mismatchRejected = $false
try {
    Assert-BinaryVersion -BinaryPath $developmentBinary -ExpectedVersion "0.1.0-rc.3"
}
catch {
    $mismatchRejected = $true
}
if (-not $mismatchRejected) {
    throw "Version mismatch was not rejected"
}

$releaseBinary = Join-Path $testRoot ("release-" + $binaryName)
$ldflags = "-X github.com/superdoccimo/done-canary/internal/model.Version=0.1.0-rc.3"
& $go build -trimpath -ldflags $ldflags -o $releaseBinary ./cmd/done-canary
if ($LASTEXITCODE -ne 0) { throw "Release build failed" }
Assert-BinaryVersion -BinaryPath $releaseBinary -ExpectedVersion "0.1.0-rc.3"

Write-Output "PACKAGE_VERSION_TEST PASS"
