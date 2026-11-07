$ErrorActionPreference = 'Stop'

# Run locally after closing CraftBox. Install the verified build before cleanup.
$projectRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..')).Path
$buildRoot = (Resolve-Path -LiteralPath (Join-Path $projectRoot 'builds')).Path
$installRoot = (Resolve-Path -LiteralPath 'D:\bedrock-tools').Path
if ([IO.Path]::GetDirectoryName($buildRoot) -ine $projectRoot) { throw 'Unexpected build directory' }
if ($installRoot.TrimEnd('\') -ine 'D:\bedrock-tools') { throw 'Unexpected installation directory' }
$buildPath = Join-Path $buildRoot 'Craft-Box.exe'
$installPath = Join-Path $installRoot 'Craft-Box.exe'
$expectedHash = 'B99915BA7E35A77622785D50DEAADA145CB30645752A89981363F610D26FFF63'
if ((Get-FileHash -LiteralPath $buildPath -Algorithm SHA256).Hash -ne $expectedHash) {
    throw 'Source EXE differs from the verified CraftBox build. No files have been changed.'
}

function Get-ObsoleteExecutables([string] $root) {
    if ((Get-Item -LiteralPath $root).Attributes -band [IO.FileAttributes]::ReparsePoint) {
        throw "Linked cleanup directory is not allowed: $root"
    }
    foreach ($file in Get-ChildItem -LiteralPath $root -File -Filter '*.exe') {
        if ($file.Name -ieq 'Craft-Box.exe' -or $file.Name -inotmatch '^(?:craft-?box|bedrocktool)(?:[-_.].*)?\.exe$') { continue }
        $absolute = (Resolve-Path -LiteralPath $file.FullName).Path
        if ([IO.Path]::GetDirectoryName($absolute).TrimEnd('\') -ine $root.TrimEnd('\') -or
            ($file.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw "Unexpected cleanup target: $absolute"
        }
        $file
    }
}

$obsolete = @((Get-ObsoleteExecutables $installRoot); (Get-ObsoleteExecutables $buildRoot))
try {
    Copy-Item -LiteralPath $buildPath -Destination $installPath -Force
} catch {
    throw "Close CraftBox and ensure D:\bedrock-tools is writable. $($_.Exception.Message)"
}
if ((Get-FileHash -LiteralPath $installPath -Algorithm SHA256).Hash -ne $expectedHash) {
    throw 'Installed executable hash mismatch. Old EXEs have been retained.'
}
$removed = 0
$freedBytes = 0L
foreach ($file in $obsolete) {
    Remove-Item -LiteralPath $file.FullName -Force
    $removed++
    $freedBytes += $file.Length
}
[PSCustomObject]@{
    Installed = $installPath
    Version = '1.63.1-url-pack-handshake-dev'
    SHA256 = $expectedHash
    RemovedExecutables = $removed
    FreedBytes = $freedBytes
} | Format-List
