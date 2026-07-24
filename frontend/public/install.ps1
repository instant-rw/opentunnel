$ErrorActionPreference = "Stop"

$Repository = if ($env:OPENTUNNEL_REPOSITORY) { $env:OPENTUNNEL_REPOSITORY } else { "opentunnel/opentunnel" }
$InstallDir = if ($env:OPENTUNNEL_INSTALL_DIR) { $env:OPENTUNNEL_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "OpenTunnel\bin" }
$Version = if ($env:OPENTUNNEL_VERSION) { $env:OPENTUNNEL_VERSION } else { "latest" }

switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
  "X64" { $Architecture = "amd64" }
  "Arm64" { $Architecture = "arm64" }
  default { throw "Unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
}

if ($Version -eq "latest") {
  $Release = Invoke-RestMethod "https://api.github.com/repos/$Repository/releases/latest"
  $Version = $Release.tag_name
}

$Number = $Version.TrimStart("v")
$Archive = "opentunnel_${Number}_windows_${Architecture}.zip"
$BaseUrl = "https://github.com/$Repository/releases/download/$Version"
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) "opentunnel-$([Guid]::NewGuid())"

try {
  New-Item -ItemType Directory -Path $TempDir | Out-Null
  Invoke-WebRequest "$BaseUrl/$Archive" -OutFile (Join-Path $TempDir $Archive)
  Invoke-WebRequest "$BaseUrl/checksums.txt" -OutFile (Join-Path $TempDir "checksums.txt")

  $ChecksumLine = Get-Content (Join-Path $TempDir "checksums.txt") | Where-Object { $_ -match "\s$([regex]::Escape($Archive))$" }
  if (-not $ChecksumLine) { throw "$Archive is missing from checksums.txt" }
  $Expected = ($ChecksumLine -split "\s+")[0].ToLowerInvariant()
  $Actual = (Get-FileHash (Join-Path $TempDir $Archive) -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($Actual -ne $Expected) { throw "Checksum verification failed" }

  Expand-Archive (Join-Path $TempDir $Archive) -DestinationPath $TempDir
  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Copy-Item (Join-Path $TempDir "opentunnel.exe") (Join-Path $InstallDir "opentunnel.exe") -Force

  $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if (($UserPath -split ";") -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to your user PATH. Open a new terminal to use it."
  }
  Write-Host "Installed OpenTunnel $Version to $InstallDir\opentunnel.exe"
}
finally {
  Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
}
