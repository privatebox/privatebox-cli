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

The API base URL is **not** persisted. It always comes from the
`PRIVATEBOX_API_URL` environment variable when set, otherwise from the
`defaultAPIBaseURL` constant in `internal/config/config.go` — so changing
that constant (or the env var) takes effect immediately without needing to
log out or delete a config file.

## Installing

Each release provides AMD64 (`amd64`) and ARM64 (`arm64`) builds for Linux,
macOS and Windows. Replace `1.0.0` below with the release version you want.

### Linux packages

The `.deb` and `.rpm` packages install the executable as
`/usr/bin/privatebox`.

Debian or Ubuntu, on AMD64 or ARM64:

```bash
VERSION=1.0.0
ARCH=amd64 # Use arm64 on ARM64 systems.
curl -LO "https://github.com/privatebox/privatebox-cli/releases/download/v${VERSION}/privatebox_${VERSION}_${ARCH}.deb"
sudo dpkg -i "privatebox_${VERSION}_${ARCH}.deb"
privatebox --version
```

Fedora, RHEL or another RPM-based distribution, on AMD64 or ARM64:

```bash
VERSION=1.0.0
ARCH=amd64 # Use arm64 on ARM64 systems.
curl -LO "https://github.com/privatebox/privatebox-cli/releases/download/v${VERSION}/privatebox_${VERSION}_${ARCH}.rpm"
sudo rpm -Uvh "privatebox_${VERSION}_${ARCH}.rpm"
privatebox --version
```

For distributions without `.deb` or `.rpm` support, download
`privatebox_<version>_linux_amd64.tar.gz` or
`privatebox_<version>_linux_arm64.tar.gz`, extract it, then install the binary
somewhere on your `PATH`:

```bash
sudo install -m 0755 privatebox /usr/local/bin/privatebox
```

### macOS

Homebrew automatically selects the correct build for Intel (`amd64`) or Apple
Silicon (`arm64`):

```bash
brew tap privatebox/privatebox
brew install --cask privatebox
privatebox --version
```

Alternatively, download `privatebox_<version>_darwin_amd64.tar.gz` for an
Intel Mac or `privatebox_<version>_darwin_arm64.tar.gz` for Apple Silicon,
extract it, then run:

```bash
sudo install -m 0755 privatebox /usr/local/bin/privatebox
```

### Windows

Download `privatebox_<version>_windows_amd64.zip` for an Intel/AMD 64-bit PC,
or `privatebox_<version>_windows_arm64.zip` for a Windows on ARM PC. Extract
`privatebox.exe` and put it in a directory on your `PATH`, such as
`C:\Tools`. Windows SmartScreen may warn about an unsigned executable the
first time it runs.

### Verify a download

Every release includes `checksums.txt` containing SHA-256 checksums. On Linux,
download it beside the selected package or archive and run:

```bash
sha256sum --ignore-missing -c checksums.txt
```

On macOS, compare the checksum printed by `shasum -a 256 <filename>` with the
matching line in `checksums.txt`.

On Windows, compare the relevant line in `checksums.txt` with:

```powershell
Get-FileHash .\privatebox_1.0.0_windows_amd64.zip -Algorithm SHA256
```

## Building and releasing

Building locally requires Go 1.22 or newer:

```bash
go build -o privatebox .
./privatebox --version # Reports "dev" for an untagged local build.
```

Maintainers can validate every release target locally with GoReleaser:

```bash
goreleaser release --snapshot --clean
```

Pushing a tag whose name starts with `v` runs the release workflow. GoReleaser
builds all six OS/architecture combinations, injects the tag into
`privatebox --version`, creates the archives and Linux packages, writes
`checksums.txt`, publishes everything to the matching GitHub Release, and
updates the PrivateBox Homebrew tap.

The release workflow uses a short-lived GitHub App token for the tap update.
Configure the `HOMEBREW_TAP_APP_ID` Actions variable and the
`HOMEBREW_TAP_APP_PRIVATE_KEY` Actions secret in this repository. The app
should be installed only on `privatebox/homebrew-privatebox` with repository
contents read/write access.

## Example session

```
$ PRIVATEBOX_API_URL=https://api.example.com/v1 privatebox auth login --email jane@example.com
Password:
Logged in as jane@example.com

$ privatebox status
Logged in as Jane Doe <jane@example.com>
API: https://api.example.com/v1

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
