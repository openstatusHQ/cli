#!/usr/bin/env pwsh
# Copyright 2019-2025 the Deno authors. All rights reserved. MIT license.
# Copyright 2025 OpenStatus. All rights reserved. MIT license.
# Adopted from https://github.com/denoland/deno_install

$ErrorActionPreference = 'Stop'

if ($v) {
  $Version = "v${v}"
}
if ($Args.Length -eq 1) {
  $Version = $Args.Get(0)
}

$OpenstatusInstall = $env:OPENSTATUS_INSTALL
$BinDir = if ($OpenstatusInstall) {
  "${OpenstatusInstall}\bin"
} else {
  "${Home}\.openstatus\bin"
}

$OpenstatusExe = "$BinDir\openstatus.exe"
$Target = 'win-x64'

$Version = if (!$Version) {
  curl.exe --ssl-revoke-best-effort -s "https://api.github.com/repos/openstatusHQ/cli/releases/latest" |
  ConvertFrom-Json |
  Select-Object -ExpandProperty tag_name
} else {
  $Version
}

Write-Output "Installing openstatus ${Version} for ${Target}"

$DownloadUrl = "https://github.com/openstatusHQ/cli/releases/download/${Version}/openstatus-${Target}.exe"

if (!(Test-Path $BinDir)) {
  New-Item $BinDir -ItemType Directory | Out-Null
}

curl.exe --ssl-revoke-best-effort -Lo $OpenstatusExe $DownloadUrl

$User = [System.EnvironmentVariableTarget]::User
$Path = [System.Environment]::GetEnvironmentVariable('Path', $User)
if (!(";${Path};".ToLower() -like "*;${BinDir};*".ToLower())) {
  [System.Environment]::SetEnvironmentVariable('Path', "${Path};${BinDir}", $User)
  $Env:Path += ";${BinDir}"
}

Write-Output "openstatus was installed successfully to ${OpenstatusExe}"
Write-Output "Run 'openstatus --help' to get started"
Write-Output "Stuck? Join our Discord https://openstatus.dev/discord"
