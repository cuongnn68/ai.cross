param()

$ErrorActionPreference = 'Stop'

if ($env:OS -ne 'Windows_NT') {
    throw 'This installer requires Windows.'
}

$architecture = $env:PROCESSOR_ARCHITEW6432
if (-not $architecture) {
    $architecture = $env:PROCESSOR_ARCHITECTURE
}
$arch = switch ($architecture) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "Unsupported Windows architecture: $architecture" }
}

[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$release = Invoke-RestMethod -Uri 'https://api.github.com/repos/cuongnn68/ai.cross/releases/latest'
$assetName = "ai-cross-windows-$arch.exe"
$asset = $release.assets | Where-Object { $_.name -eq $assetName }
$checksum = $release.assets | Where-Object { $_.name -eq "$assetName.sha256" }
if (-not $asset -or -not $checksum) {
    throw "Release $($release.tag_name) does not contain $assetName and its checksum."
}

$installDir = Join-Path $env:LOCALAPPDATA 'ai-cross\bin'
$tempDir = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir | Out-Null
try {
    $download = Join-Path $tempDir $assetName
    $checksumFile = Join-Path $tempDir 'checksum.sha256'
    Invoke-WebRequest -UseBasicParsing -Uri $asset.browser_download_url -OutFile $download
    Invoke-WebRequest -UseBasicParsing -Uri $checksum.browser_download_url -OutFile $checksumFile
    $expectedHash = (Get-Content -Raw $checksumFile).Trim()
    if ($expectedHash -notmatch '^[a-fA-F0-9]{64}$' -or
        (Get-FileHash $download -Algorithm SHA256).Hash -ne $expectedHash) {
        throw 'Downloaded exe failed SHA-256 verification.'
    }

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Move-Item -Force -Path $download -Destination (Join-Path $installDir 'ai-cross.exe')

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($installDir -notin ($userPath -split ';')) {
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $installDir } else { "$userPath;$installDir" }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    }
    if ($installDir -notin ($env:Path -split ';')) {
        $env:Path = "$env:Path;$installDir"
    }

    Write-Host "Installed $($release.tag_name) to $installDir"
    Write-Host 'Run: ai-cross help'
    Write-Host 'Restart other terminals to pick up the updated user PATH.'
} finally {
    Remove-Item -Recurse -Force $tempDir
}
