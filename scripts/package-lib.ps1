function Assert-ValidVersion {
    param([string]$Value)

    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -match '[\s/\\]') {
        throw "Version must be non-empty and must not contain whitespace or path separators"
    }
}

function Assert-BinaryVersion {
    param(
        [string]$BinaryPath,
        [string]$ExpectedVersion
    )

    $versionOutput = @(& $BinaryPath version)
    $versionExit = $LASTEXITCODE
    if ($versionExit -ne 0) {
        throw "Version command failed for $BinaryPath with exit $versionExit"
    }
    $actualVersion = ($versionOutput -join "`n").Trim()
    $expectedOutput = "done-canary $ExpectedVersion"
    if ($actualVersion -ne $expectedOutput) {
        throw "Version mismatch for ${BinaryPath}: got '$actualVersion', want '$expectedOutput'"
    }
}
