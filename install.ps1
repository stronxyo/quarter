# Installs Quarter on Windows:
#
#   irm https://raw.githubusercontent.com/stronxyo/quarter/main/install.ps1 | iex
#
# It downloads the latest release for this PC from GitHub, checks it against
# the release's checksums, puts quarter.exe in your user folder
# (%LOCALAPPDATA%\Programs\quarter) and adds that folder to your PATH. No
# administrator rights needed. Optional settings, as environment variables:
#   QUARTER_VERSION      a version to install instead of the latest (v0.1.0)
#   QUARTER_INSTALL_DIR  where to put the program
#   QUARTER_BASE_URL     where release files come from (for testing)
# Works in Windows PowerShell 5.1 (every Windows 10 and 11) and PowerShell 7.

# All in a block of its own: run through iex, nothing it sets stays behind
# in your PowerShell window.
& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue' # the progress bar slows downloads a lot in 5.1
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    $repo = 'stronxyo/quarter'
    $dir = if ($env:QUARTER_INSTALL_DIR) { $env:QUARTER_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\quarter' }

    # The processor: the real one, also when this PowerShell runs emulated.
    $cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    switch ($cpu) {
        'AMD64' { $arch = 'amd64' }
        'ARM64' { $arch = 'arm64' }
        default { throw "quarter install: no build for this processor ($cpu); see https://github.com/$repo/releases" }
    }

    $version = $env:QUARTER_VERSION
    if (-not $version) {
        # The latest release's page redirects to its tag: no API call, so no
        # rate limit on a shared connection (install.sh does the same).
        # PowerShell 5.1 keeps the final address in ResponseUri, 7 in RequestMessage.
        $r = Invoke-WebRequest "https://github.com/$repo/releases/latest" -Method Head -UseBasicParsing
        $final = if ($r.BaseResponse.ResponseUri) { $r.BaseResponse.ResponseUri } else { $r.BaseResponse.RequestMessage.RequestUri }
        $version = ([string]$final).Split('/')[-1]
    }
    if ($version -notmatch '^v\d') { throw "quarter install: couldn't find the latest release (got '$version')" }
    $base = if ($env:QUARTER_BASE_URL) { $env:QUARTER_BASE_URL } else { "https://github.com/$repo/releases/download/$version" }
    $archive = "quarter_$($version.Substring(1))_windows_$arch.zip"

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("quarter-" + [guid]::NewGuid())
    New-Item -ItemType Directory $tmp | Out-Null
    try {
        Write-Host "Downloading Quarter $version for windows/$arch..."
        Invoke-WebRequest "$base/$archive" -OutFile (Join-Path $tmp $archive) -UseBasicParsing
        Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt') -UseBasicParsing

        $want = $null
        foreach ($line in Get-Content (Join-Path $tmp 'checksums.txt')) {
            $parts = $line -split '\s+'
            if ($parts.Count -ge 2 -and $parts[1] -eq $archive) { $want = $parts[0] }
        }
        if (-not $want) { throw "quarter install: $archive isn't in the release's checksums" }
        $got = (Get-FileHash (Join-Path $tmp $archive) -Algorithm SHA256).Hash
        if ($got -ne $want) { throw "quarter install: the download doesn't match its checksum; nothing was installed" }

        Expand-Archive (Join-Path $tmp $archive) -DestinationPath (Join-Path $tmp 'unpacked') -Force
        New-Item -ItemType Directory -Force $dir | Out-Null
        Copy-Item (Join-Path $tmp 'unpacked\quarter.exe') (Join-Path $dir 'quarter.exe') -Force
        Write-Host "Installed $(Join-Path $dir 'quarter.exe')"
    }
    finally {
        Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }

    # Typing quarter has to just work: on the user's PATH for new terminals,
    # and in this one right away.
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $parts = @($userPath -split ';' | Where-Object { $_ })
    if ($parts -notcontains $dir) {
        [Environment]::SetEnvironmentVariable('Path', (($parts + $dir) -join ';'), 'User')
        Write-Host "Added $dir to your PATH. Terminals that were already open find quarter once you reopen them."
    }
    if (($env:Path -split ';') -notcontains $dir) { $env:Path = "$env:Path;$dir" }
    $first = Get-Command quarter -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($first -and $first.Source -ne (Join-Path $dir 'quarter.exe')) {
        Write-Host "Note: typing quarter starts $($first.Source), which comes first on your PATH. Remove that one to use this install."
    }
    Write-Host 'Type quarter to start.'
}
