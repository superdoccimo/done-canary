param(
    [string]$Version = "0.1.0-rc.3",
    [string]$TargetOS = "",
    [string]$TargetArch = ""
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "package-lib.ps1")

Assert-ValidVersion $Version

$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$distRoot = [System.IO.Path]::GetFullPath((Join-Path $projectRoot "dist"))

if (-not $distRoot.StartsWith($projectRoot + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing package output outside the project"
}

$go = (Get-Command go -ErrorAction Stop).Source
$hostGOOS = (& $go env GOOS).Trim()
if ($LASTEXITCODE -ne 0) { throw "Unable to determine host GOOS" }
$hostGOARCH = (& $go env GOARCH).Trim()
if ($LASTEXITCODE -ne 0) { throw "Unable to determine host GOARCH" }

if ([string]::IsNullOrWhiteSpace($TargetOS)) { $TargetOS = $hostGOOS }
if ([string]::IsNullOrWhiteSpace($TargetArch)) { $TargetArch = $hostGOARCH }
if (@("windows", "linux", "darwin") -notcontains $TargetOS) {
    throw "Unsupported target OS '$TargetOS'"
}
if ($TargetArch -ne "amd64") {
    throw "Unsupported target architecture '$TargetArch'"
}
if ($TargetOS -ne $hostGOOS) {
    throw "Release archives must be created on a matching operating system so executable metadata is preserved"
}

$binaryName = if ($TargetOS -eq "windows") { "done-canary.exe" } else { "done-canary" }
$packageName = "done-canary-v$Version-$TargetOS-$TargetArch"
$releaseRoot = Join-Path $distRoot ("done-canary-v" + $Version)
$packageDir = Join-Path $releaseRoot $packageName
$ldflags = "-X github.com/superdoccimo/done-canary/internal/model.Version=$Version"

if (Test-Path -LiteralPath $releaseRoot) {
    Remove-Item -LiteralPath $releaseRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $packageDir | Out-Null

$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO = $env:CGO_ENABLED

try {
    $env:CGO_ENABLED = "0"
    $env:GOOS = $TargetOS
    $env:GOARCH = $TargetArch
    $binaryPath = Join-Path $packageDir $binaryName
    & $go build -trimpath -ldflags $ldflags -o $binaryPath ./cmd/done-canary
    if ($LASTEXITCODE -ne 0) { throw "Go build failed for $TargetOS/$TargetArch" }

    # Validate the embedded version using a native probe even when the release
    # target architecture differs from the runner architecture.
    $probeName = if ($hostGOOS -eq "windows") { "version-probe.exe" } else { "version-probe" }
    $probePath = Join-Path $releaseRoot $probeName
    $env:GOOS = $hostGOOS
    $env:GOARCH = $hostGOARCH
    & $go build -trimpath -ldflags $ldflags -o $probePath ./cmd/done-canary
    if ($LASTEXITCODE -ne 0) { throw "Native version-probe build failed" }
    Assert-BinaryVersion -BinaryPath $probePath -ExpectedVersion $Version
    Remove-Item -LiteralPath $probePath -Force

    Copy-Item (Join-Path $projectRoot "README.md") $packageDir -Force
    Copy-Item (Join-Path $projectRoot "LICENSE") $packageDir -Force
    Copy-Item (Join-Path $projectRoot "SECURITY.md") $packageDir -Force

    if ($TargetOS -eq "windows") {
        $archivePath = Join-Path $releaseRoot ($packageName + ".zip")
        Compress-Archive -Path (Join-Path $packageDir "*") -DestinationPath $archivePath -Force
    }
    else {
        & chmod 0755 $binaryPath
        if ($LASTEXITCODE -ne 0) { throw "Unable to mark $binaryName executable" }
        $archivePath = Join-Path $releaseRoot ($packageName + ".tar.gz")
        Push-Location $packageDir
        try {
            & tar -czf $archivePath $binaryName "README.md" "LICENSE" "SECURITY.md"
            if ($LASTEXITCODE -ne 0) { throw "tar failed for $TargetOS/$TargetArch" }
        }
        finally {
            Pop-Location
        }
    }

    if (-not (Test-Path -LiteralPath $archivePath)) {
        throw "Package archive was not created: $archivePath"
    }
    $hash = Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath
    $checksumPath = Join-Path $releaseRoot ($packageName + ".sha256")
    Set-Content -LiteralPath $checksumPath -Value ("$($hash.Hash.ToLowerInvariant())  $([System.IO.Path]::GetFileName($archivePath))") -Encoding utf8NoBOM
    Remove-Item -LiteralPath $packageDir -Recurse -Force
}
finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
}

Write-Output $archivePath
Write-Output $checksumPath
