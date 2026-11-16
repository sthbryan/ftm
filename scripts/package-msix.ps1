#Requires -Version 5.1
<#
.SYNOPSIS
  Package a Wails Windows .exe into an MSIX for sideload or Microsoft Store upload.

.DESCRIPTION
  Layout:
    <staging>/
      AppxManifest.xml
      ftm-desktop.exe   (or -ExeName)
      Assets/*.png

  Needs Windows + Windows SDK (makeappx.exe). Optional: signtool for local sideload.

  Store notes:
  - Set -Publisher and -IdentityName EXACTLY as Partner Center shows
    (Package/Identity → Name and Publisher).
  - Version must be four parts: Major.Minor.Build.Revision (e.g. 0.10.0.0).
  - For Store upload you usually do NOT sign the MSIX yourself; Microsoft re-signs.
  - For local install (sideload), pass -Sign and install the cert once.

.EXAMPLE
  # After: wails build -s -nopackage  (from cmd/ftm-desktop/)
  .\scripts\package-msix.ps1 `
    -ExePath .\cmd\ftm-desktop\build\bin\ftm-desktop.exe `
    -Version 0.10.0.0 `
    -IdentityName "YourPublisher.ftm" `
    -Publisher "CN=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"

.EXAMPLE
  # Sideload test (self-signed)
  .\scripts\package-msix.ps1 -ExePath .\ftm-desktop.exe -Version 0.10.0.0 -Sign
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string] $ExePath,

    # MSIX needs four parts (e.g. 0.10.0.0). "0.10.0" or "v0.10.0" is padded automatically.
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v?\d+(\.\d+){1,3}$')]
    [string] $Version,

    # Partner Center → Product identity → Package/Identity Name
    [string] $IdentityName = "com.justcallmebryan.ftm",

    # Partner Center → Package/Identity Publisher (must match exactly for Store).
    # For local sideload with -Sign, leave default or match your test cert subject.
    [string] $Publisher = "CN=ftm-dev",

    [string] $DisplayName = "Foundry Tunnel Manager",
    [string] $PublisherDisplayName = "sthbryan",
    [string] $Description = "Share your Foundry VTT world with your players. No port forwarding.",
    [string] $ExeName = "ftm-desktop.exe",

    # Architecture inside the package (Wails release builds are typically x64).
    [ValidateSet("x64", "x86", "arm64", "neutral")]
    [string] $Architecture = "x64",

    # Optional source PNG (any size); resized to Store asset sizes.
    [string] $IconPath = "",

    [string] $OutputDir = "",
    [string] $OutputName = "ftm-desktop",

    # Create a self-signed cert (once) and sign for local sideload testing.
    [switch] $Sign,

    # Reuse an existing .pfx instead of generating one.
    [string] $PfxPath = "",
    [securestring] $PfxPassword,

    # Skip makeappx and only write the staging folder (debug).
    [switch] $StagingOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-Windows {
    if (-not ($IsWindows -or $env:OS -match "Windows")) {
        throw "package-msix.ps1 must run on Windows (makeappx / Windows SDK)."
    }
}

function Find-SdkTool {
    param([Parameter(Mandatory = $true)][string] $Name)

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $roots = @(
        "${env:ProgramFiles(x86)}\Windows Kits\10\bin",
        "${env:ProgramFiles}\Windows Kits\10\bin"
    ) | Where-Object { $_ -and (Test-Path $_) }

    foreach ($root in $roots) {
        $hit = Get-ChildItem -Path $root -Recurse -Filter $Name -ErrorAction SilentlyContinue |
            Where-Object { $_.FullName -match '\\x64\\' -or $_.FullName -match '\\arm64\\' } |
            Sort-Object FullName -Descending |
            Select-Object -First 1
        if ($hit) { return $hit.FullName }
    }

    throw "Could not find $Name. Install the Windows 10/11 SDK (makeappx + signtool)."
}

function ConvertTo-FourPartVersion {
    param([string] $V)
    $clean = $V.Trim() -replace '^v', ''
    $parts = @($clean -split '\.')
    while ($parts.Count -lt 4) { $parts += '0' }
    if ($parts.Count -gt 4) { $parts = $parts[0..3] }
    if ($parts[3] -ne '0') {
        Write-Warning "Revision $($parts[3]) is rejected by the Store; forcing 0."
        $parts[3] = '0'
    }
    return ($parts -join '.')
}

function New-ResizedPng {
    param(
        [string] $SourcePath,
        [string] $DestPath,
        [int] $Width,
        [int] $Height
    )

    Add-Type -AssemblyName System.Drawing
    $src = [System.Drawing.Image]::FromFile((Resolve-Path $SourcePath))
    try {
        $bmp = New-Object System.Drawing.Bitmap $Width, $Height
        try {
            $g = [System.Drawing.Graphics]::FromImage($bmp)
            try {
                $g.Clear([System.Drawing.Color]::Transparent)
                $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
                $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                $g.DrawImage($src, 0, 0, $Width, $Height)
                $dir = Split-Path -Parent $DestPath
                if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir | Out-Null }
                $bmp.Save($DestPath, [System.Drawing.Imaging.ImageFormat]::Png)
            }
            finally { $g.Dispose() }
        }
        finally { $bmp.Dispose() }
    }
    finally { $src.Dispose() }
}

function Write-AppxManifest {
    param(
        [string] $Path,
        [string] $IdentityName,
        [string] $Publisher,
        [string] $Version,
        [string] $Architecture,
        [string] $DisplayName,
        [string] $PublisherDisplayName,
        [string] $Description,
        [string] $ExeName
    )

    # XML: Publisher may contain characters that need escaping in attributes.
    $xmlPublisher = [System.Security.SecurityElement]::Escape($Publisher)
    $xmlDisplay = [System.Security.SecurityElement]::Escape($DisplayName)
    $xmlPubDisplay = [System.Security.SecurityElement]::Escape($PublisherDisplayName)
    $xmlDesc = [System.Security.SecurityElement]::Escape($Description)
    $xmlId = [System.Security.SecurityElement]::Escape($IdentityName)
    $xmlExe = [System.Security.SecurityElement]::Escape($ExeName)

    $manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:uap5="http://schemas.microsoft.com/appx/manifest/uap/windows10/5"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  IgnorableNamespaces="uap uap5 rescap">

  <Identity
    Name="$xmlId"
    Publisher="$xmlPublisher"
    Version="$Version"
    ProcessorArchitecture="$Architecture" />

  <Properties>
    <DisplayName>$xmlDisplay</DisplayName>
    <PublisherDisplayName>$xmlPubDisplay</PublisherDisplayName>
    <Description>$xmlDesc</Description>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>

  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.17763.0" MaxVersionTested="10.0.22621.0" />
  </Dependencies>

  <Resources>
    <Resource Language="en-us" />
  </Resources>

  <Applications>
    <Application Id="App" Executable="$xmlExe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements
        DisplayName="$xmlDisplay"
        Description="$xmlDesc"
        BackgroundColor="transparent"
        Square150x150Logo="Assets\Square150x150Logo.png"
        Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile
          Wide310x150Logo="Assets\Wide310x150Logo.png"
          Square71x71Logo="Assets\Square71x71Logo.png" />
        <uap:SplashScreen Image="Assets\SplashScreen.png" />
      </uap:VisualElements>
      <Extensions>
        <uap5:Extension
          Category="windows.startupTask"
          Executable="$xmlExe"
          EntryPoint="Windows.FullTrustApplication">
          <uap5:StartupTask
            TaskId="FtmAutostart"
            Enabled="false"
            DisplayName="$xmlDisplay" />
        </uap5:Extension>
      </Extensions>
    </Application>
  </Applications>

  <Capabilities>
    <Capability Name="internetClient" />
    <Capability Name="internetClientServer" />
    <Capability Name="privateNetworkClientServer" />
    <rescap:Capability Name="runFullTrust" />
  </Capabilities>
</Package>
"@

    # UTF-8 without BOM (makeappx is picky about BOM sometimes).
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($Path, $manifest, $utf8NoBom)
}

function New-DevCertificate {
    param(
        [string] $Subject,
        [string] $PfxOut,
        [securestring] $Password
    )

    if (Test-Path $PfxOut) {
        Write-Host "Reusing existing cert: $PfxOut"
        return
    }

    Write-Host "Creating self-signed cert for sideload: $Subject"
    $cert = New-SelfSignedCertificate `
        -Type Custom `
        -Subject $Subject `
        -KeyUsage DigitalSignature `
        -FriendlyName "ftm MSIX dev" `
        -CertStoreLocation "Cert:\CurrentUser\My" `
        -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")

    if (-not $Password) {
        $Password = ConvertTo-SecureString -String "ftm-dev" -Force -AsPlainText
    }
    Export-PfxCertificate -Cert $cert -FilePath $PfxOut -Password $Password | Out-Null
    Write-Host "Exported $PfxOut (install this cert as Trusted People / Trusted Root for sideload)."
}

# --- main ---

Assert-Windows

$ExePath = (Resolve-Path $ExePath).Path
if (-not (Test-Path $ExePath)) { throw "Exe not found: $ExePath" }

$Version = ConvertTo-FourPartVersion $Version

if (-not $OutputDir) {
    $OutputDir = Join-Path (Split-Path -Parent $ExePath) "msix"
}
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$OutputDir = (Resolve-Path $OutputDir).Path

$staging = Join-Path $OutputDir "staging"
if (Test-Path $staging) { Remove-Item -Recurse -Force $staging }
New-Item -ItemType Directory -Path $staging | Out-Null
New-Item -ItemType Directory -Path (Join-Path $staging "Assets") | Out-Null

# Binary
$destExe = Join-Path $staging $ExeName
Copy-Item -Force $ExePath $destExe
Write-Host "Staged binary: $destExe"

# Icons
if (-not $IconPath) {
    $candidates = @(
        (Join-Path $PSScriptRoot "..\cmd\ftm-desktop\icon.png"),
        (Join-Path $PSScriptRoot "..\cmd\ftm-desktop\build\appicon.png"),
        (Join-Path (Split-Path $ExePath) "icon.png")
    )
    foreach ($c in $candidates) {
        if (Test-Path $c) { $IconPath = $c; break }
    }
}
if (-not $IconPath -or -not (Test-Path $IconPath)) {
    throw "No icon PNG found. Pass -IconPath path\to\icon.png"
}
$IconPath = (Resolve-Path $IconPath).Path
Write-Host "Icon source: $IconPath"

$assets = Join-Path $staging "Assets"
New-ResizedPng $IconPath (Join-Path $assets "StoreLogo.png") 50 50
New-ResizedPng $IconPath (Join-Path $assets "Square44x44Logo.png") 44 44
New-ResizedPng $IconPath (Join-Path $assets "Square71x71Logo.png") 71 71
New-ResizedPng $IconPath (Join-Path $assets "Square150x150Logo.png") 150 150
New-ResizedPng $IconPath (Join-Path $assets "Wide310x150Logo.png") 310 150
New-ResizedPng $IconPath (Join-Path $assets "SplashScreen.png") 620 300

$manifestPath = Join-Path $staging "AppxManifest.xml"
Write-AppxManifest `
    -Path $manifestPath `
    -IdentityName $IdentityName `
    -Publisher $Publisher `
    -Version $Version `
    -Architecture $Architecture `
    -DisplayName $DisplayName `
    -PublisherDisplayName $PublisherDisplayName `
    -Description $Description `
    -ExeName $ExeName

Write-Host "Wrote $manifestPath"

if ($StagingOnly) {
    Write-Host "StagingOnly: package left at $staging"
    exit 0
}

$makeappx = Find-SdkTool "makeappx.exe"
$msixPath = Join-Path $OutputDir "$OutputName-$Architecture.msix"
if (Test-Path $msixPath) { Remove-Item -Force $msixPath }

Write-Host "Packing with makeappx..."
& $makeappx pack /d $staging /p $msixPath /o
if ($LASTEXITCODE -ne 0) { throw "makeappx failed with exit $LASTEXITCODE" }
Write-Host "Created $msixPath"

if ($Sign) {
    $signtool = Find-SdkTool "signtool.exe"
    if (-not $PfxPath) {
        $PfxPath = Join-Path $OutputDir "ftm-msix-dev.pfx"
    }
    if (-not $PfxPassword) {
        $PfxPassword = ConvertTo-SecureString -String "ftm-dev" -Force -AsPlainText
    }
    # Publisher on the cert must match Identity Publisher for install.
    New-DevCertificate -Subject $Publisher -PfxOut $PfxPath -Password $PfxPassword

    # signtool wants plain password on CLI for pfx in many setups
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($PfxPassword)
    try {
        $plain = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }

    Write-Host "Signing $msixPath ..."
    & $signtool sign /fd SHA256 /a /f $PfxPath /p $plain /tr http://timestamp.digicert.com /td SHA256 $msixPath
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "Timestamp failed or sign error; retrying without timestamp..."
        & $signtool sign /fd SHA256 /a /f $PfxPath /p $plain $msixPath
        if ($LASTEXITCODE -ne 0) { throw "signtool failed with exit $LASTEXITCODE" }
    }
    Write-Host "Signed. Sideload: Add-AppxPackage `"$msixPath`""
    Write-Host "If install fails on cert trust, import $PfxPath into Local Machine → Trusted People (and/or Trusted Root for testing)."
}
else {
    Write-Host @"

Unsigned MSIX ready for Microsoft Store upload (Partner Center re-signs).
  File: $msixPath

Before Store submission, set:
  -IdentityName  = Package identity Name from Partner Center
  -Publisher     = Package identity Publisher (CN=...) from Partner Center
  -Version       = four-part version matching this release

"@
}

Write-Host "Done."
