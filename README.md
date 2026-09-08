# privatebox-cli

A cross-platform command-line client for the PrivateBox API, written in Go
using only the standard library — no external dependencies, no `go.sum`.

## Commands

```
privatebox auth login --email you@example.com   Log in (prompts for password)
privatebox auth verify-code --code 123456        Submit the email verification code, if required
privatebox auth logout                           Clear the saved session

privatebox status                                Show login status (email / API URL)

privatebox items [--page N]                      List items in your inbox
privatebox items sent [--page N]                 List items you've sent
privatebox items scanned [--page N]              List scanned items (URLs are clickable links)

privatebox order scan --items 1001,1003 [--destroy]
                                                 Request a scan of the given items
privatebox order send --items 1001,1003 --receivers-name "Jane Doe" \
                       --service-id 25 --country-iso NZ --address "15 Beaumonts Way" \
                       [--address-detail --suburb --city --post-code] [--add-new]
                                                 Place a send order
                                                 (or use --search-id ID to use a saved address)
privatebox order send-cost --items 1001,1003 --country NZ --address "15 Beaumonts Way" \
                           [--address-detail --suburb --city --state --post-code]
                                                 Compare shipping services/estimates
privatebox order destroy --items 1001,1003      Queue items for destruction

privatebox meta countries                        List reference country codes
privatebox meta frequency                        List reference scan-frequency options

privatebox --json <command>                      Machine-readable JSON output for any command
privatebox -h / --help                           Show help
privatebox -v / --version                        Show version
```

`--items` takes a comma-separated list of item IDs, e.g. `--items 1001,1003`.
All list endpoints support `--page N`; when more than one page exists the
footer shows `page X of Y` and a hint to use `--page`.
Global flags can be placed before or after the command, for example
`privatebox --json items sent` and `privatebox items sent --json` are
equivalent.

### Login flow

1. `privatebox auth login --email you@example.com` — you'll be prompted for
   your password. The returned token is stored immediately.
2. If the account requires email verification, the CLI prints
   `A verification code was sent to your email.` and prompts for the code,
   submitting it to `/user/validate_code` automatically.
3. If you skipped step 2, run `privatebox auth verify-code --code 123456`
   later — the saved token is used for that call.

## Device identification

Every API request includes an `X-DeviceID` header. It is a SHA-256 hash of a
stable, per-machine hardware identifier (`IOPlatformUUID` on macOS,
`/etc/machine-id` on Linux, the `MachineGuid` registry value on Windows), so
the same computer always reports the same device ID across logins and
restarts, and different computers report different IDs.

## Session storage

The session token is stored in the OS-native credential store when one is
available:

| OS | Backend used |
|---|---|
| macOS | Keychain, via the built-in `security` CLI |
| Windows | Credential Manager, via the Win32 Cred* API (stdlib syscall) |
| Linux | Secret Service, via `secret-tool` (from `libsecret-tools` / `gnome-keyring`) — common on GNOME desktops, often missing on servers |

If no keyring backend is found (e.g. a headless Linux box without
`secret-tool` installed), the CLI automatically falls back to storing the
token in `~/.privatebox/config.json` with owner-only permissions (`0600`)
and prints a one-line note when this happens. Either way, the flow ("stores
token in keyring when available") is handled transparently — callers don't
need to know or care which backend was used.

The CLI connects to the Private Box production API by default. No API URL
configuration is needed.

## Installing

Each release provides AMD64 (`amd64`) and ARM64 (`arm64`) builds for Linux,
macOS and Windows. The commands below resolve the latest published release
automatically, so this README does not need a version change for each release.

### Homebrew on macOS or Linux

Homebrew automatically selects the correct build for the operating system and
architecture:

```bash
brew tap privatebox/privatebox
brew install --cask privatebox
privatebox --version
```

Upgrade an existing Homebrew installation with:

```bash
brew update
brew upgrade --cask privatebox
privatebox --version
```

### Linux and WSL: install or upgrade

Run these commands from a writable directory (requires Bash and curl):

```bash
curl -fsSL https://raw.githubusercontent.com/privatebox/privatebox-cli/main/install.sh -o privatebox-install.sh &&
  bash privatebox-install.sh
privatebox --version
```

Use the same commands for upgrades. The [installer](install.sh) detects AMD64
or ARM64, downloads the latest stable release, verifies its SHA-256 checksum,
then installs it. It prompts for sudo access only when needed to install:

- Debian/Ubuntu: a `.deb` package through APT at `/usr/bin/privatebox`.
- Fedora/RHEL with DNF: an `.rpm` package at `/usr/bin/privatebox`.
- Other Linux distributions: the archive binary at `/usr/local/bin/privatebox`.

If you installed with Homebrew, use the Homebrew upgrade commands above
instead of mixing installation methods. No Git checkout, Go or GitHub CLI
is required. Manual packages and archives remain available on the
[releases page](https://github.com/privatebox/privatebox-cli/releases/latest).

### macOS without Homebrew

Homebrew is recommended. For a manual installation instead:

```bash
RELEASE_TAG="$(
  curl -fsSL https://api.github.com/repos/privatebox/privatebox-cli/releases/latest |
    sed -n 's/.*"tag_name": "\(v[^"]*\)".*/\1/p'
)"
VERSION="${RELEASE_TAG#v}"

case "$(uname -m)" in
  x86_64) ARCH=amd64 ;;
  arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

ASSET="privatebox_${VERSION}_darwin_${ARCH}.tar.gz"
curl -fLO "https://github.com/privatebox/privatebox-cli/releases/download/${RELEASE_TAG}/${ASSET}"
curl -fLO "https://github.com/privatebox/privatebox-cli/releases/download/${RELEASE_TAG}/checksums.txt"
EXPECTED="$(grep "  ${ASSET}$" checksums.txt | cut -d ' ' -f 1)"
ACTUAL="$(shasum -a 256 "$ASSET" | cut -d ' ' -f 1)"
test -n "$EXPECTED" && test "$ACTUAL" = "$EXPECTED"
tar -xzf "$ASSET"
sudo install -m 0755 privatebox /usr/local/bin/privatebox
privatebox --version
```

Repeat the commands to upgrade; `install` replaces the existing binary.

### Windows

In PowerShell, download the latest AMD64 or ARM64 archive, verify it and copy
the executable to `C:\Tools`:

```powershell
$Release = Invoke-RestMethod "https://api.github.com/repos/privatebox/privatebox-cli/releases/latest"
$Version = $Release.tag_name.TrimStart("v")

switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    "X64"   { $Arch = "amd64" }
    "Arm64" { $Arch = "arm64" }
    default { throw "Unsupported architecture" }
}

$AssetName = "privatebox_${Version}_windows_${Arch}.zip"
$Asset = $Release.assets | Where-Object name -eq $AssetName
$Checksums = $Release.assets | Where-Object name -eq "checksums.txt"

if (-not $Asset -or -not $Checksums) { throw "Release assets not found" }

Invoke-WebRequest $Asset.browser_download_url -OutFile $AssetName
Invoke-WebRequest $Checksums.browser_download_url -OutFile "checksums.txt"

$ExpectedLine = Get-Content .\checksums.txt | Where-Object { $_ -match [regex]::Escape($AssetName) }
if (-not $ExpectedLine) { throw "Checksum not found" }

$Expected = ($ExpectedLine -split '\s+')[0].ToLowerInvariant()
$Actual = (Get-FileHash ".\$AssetName" -Algorithm SHA256).Hash.ToLowerInvariant()
if ($Actual -ne $Expected) { throw "Checksum verification failed" }

Expand-Archive ".\$AssetName" -DestinationPath ".\privatebox" -Force
New-Item -ItemType Directory -Path "C:\Tools" -Force | Out-Null
Copy-Item ".\privatebox\privatebox.exe" "C:\Tools\privatebox.exe" -Force
& "C:\Tools\privatebox.exe" --version
```

Run the same commands again to upgrade. Ensure `C:\Tools` is on `PATH` if you
want to invoke `privatebox` without its full path. Windows SmartScreen may warn
about an unsigned executable the first time it runs.

## Building locally

Building locally requires Go 1.22 or newer:

```bash
go build -o privatebox .
./privatebox --version # Reports "dev" for an untagged local build.
```

## Example session

```
$ privatebox auth login --email jane@example.com
Password:
Logged in as jane@example.com

$ privatebox status
Logged in as Jane Doe <jane@example.com>

$ privatebox items
ID     RECEIVED        WEIGHT  TYPE      STATUS    SCAN   FROM     TO
204568 13th Jan 2016   20g     Letter    Arrived          AMAZON   Jane Doe
204788 10th Jul 2016   10g     Letter    Arrived          TEST     Jane Doe

21 items | page 1 of 3 | Mail items fetched ok
Tip: view other pages with --page N (e.g. --page 2)

$ privatebox items sent
ID        SENT          TYPE    WEIGHT  STATUS  FROM  TO              DESTINATION
43424324  2nd Aug 2025  Letter  2g      Sent    AA    Jane Doe        123M Bell Road, ...

$ privatebox order scan --items 1001,1002 --destroy
Scan requested for 2 item(s). Order ID: 42
Items will be destroyed automatically after scanning.

$ privatebox order send-cost --items 1001,1002 --country NZ \
    --address "15 Beaumonts Way" --suburb Manurewa --city Auckland --post-code 2102
CARRIER      SERVICE                     ESTIMATE  BEFORE DISC  FREE SENDING  MESSAGE
CourierPost  Courier Parcel              0.95      10.95        true          Qualifies for free sending discount of $10.00 NZD
NZ Post      Economy (NZ)                4.60      4.60         false

$ privatebox order send --items 1001 --receivers-name "Jane Doe" \
    --service-id 25 --country-iso NZ --address "15 Beaumonts Way" \
    --suburb Manurewa --city Auckland --post-code 2102 --add-new
Send order created for 1 item(s).
Service: Economy (NZ)
Address verified: yes
Destination: 15 Beaumonts Way, Manurewa, Auckland 2102
Estimated cost: $4.60 (range $4.60 - $4.60)

$ privatebox items sent --json
{
  "items": [ ... ],
  "pagination": { ... },
  "meta": { ... }
}
```

## Where local (non-secret) session data is stored

`~/.privatebox/config.json`, permissions `0600` (owner read/write only).
On Windows this resolves to `%USERPROFILE%\.privatebox\config.json`. It
holds the logged-in name and email; the token itself lives in the OS
keyring when available (see above), and only appears in this file as a
fallback.
