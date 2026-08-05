$ErrorActionPreference = 'Stop'

$ScriptDirectory = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = (Resolve-Path (Join-Path $ScriptDirectory '../..')).Path
$ServerRoot = Join-Path $ProjectRoot 'src/server'
$WebDist = Join-Path $ProjectRoot 'src/web/dist'
$EmbedRoot = Join-Path $ServerRoot 'webdist'
$RuntimeRoot = Join-Path $ProjectRoot 'private/runtime'
$TemporaryBinary = Join-Path $RuntimeRoot 'jobdeck.tmp.exe'
$RootBinary = Join-Path $ProjectRoot 'jobdeck.exe'

if (-not (Test-Path (Join-Path $WebDist 'index.html'))) {
  throw 'Job Deck web output is missing. Build the UI only because dist is unavailable.'
}

New-Item -ItemType Directory -Force -Path $RuntimeRoot | Out-Null
if (Test-Path $EmbedRoot) {
  Remove-Item -Recurse -Force $EmbedRoot
}
New-Item -ItemType Directory -Force -Path $EmbedRoot | Out-Null
Copy-Item -Recurse -Force (Join-Path $WebDist '*') $EmbedRoot
Set-Content -Path (Join-Path $EmbedRoot 'placeholder.txt') -Value 'Generated embed directory.'

Push-Location $ServerRoot
try {
  go test ./...
  go build -trimpath -o $TemporaryBinary .
} finally {
  Pop-Location
}

Move-Item -Force $TemporaryBinary $RootBinary
& $RootBinary record-build
& $RootBinary verify
