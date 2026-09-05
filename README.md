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
privatebox --help / --version
```

`--items` takes a comma-separated list of item IDs, e.g. `--items 1001,1003`.
All list endpoints support `--page N`; when more than one page exists the
footer shows `page X of Y` and a hint to use `--page`.

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

## Building (developer machine)

Requires Go 1.22+ (https://go.dev/dl/) — only needed to *build* the CLI,
not to run it.

```bash
# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o build/privatebox .
tar czf privatebox_darwin_amd64.tar.gz -C build privatebox
shasum -a 256 privatebox_darwin_amd64.tar.gz

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o build/privatebox .
tar czf privatebox_darwin_arm64.tar.gz -C build privatebox
shasum -a 256 privatebox_darwin_arm64.tar.gz

# Linux
Change the x to the current version
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
GOOS=linux GOARCH=amd64 go build -o build/privatebox .
nfpm package --packager deb --config nfpm.yaml --target privatebox_1.x.x_amd64.deb
```

Save these shasum you'll paste them into the Ruby formula in https://github.com/privatebox/homebrew-privatebox.

#Upload these to a GitHub Releases page.
```bash
gh release create v1.x.x \
  privatebox_darwin_arm64.tar.gz \
  privatebox_darwin_amd64.tar.gz \
  privatebox_1.x.x_amd64.deb \
  --title "v1.x.x" --notes "Release v1.x.x"

git tag v1.x.x
git push origin v1.x.x
```
Additionaly: 
`build.sh` produces:

```
dist/privatebox-linux-amd64
dist/privatebox-linux-arm64
dist/privatebox-darwin-amd64      (Intel Mac)
dist/privatebox-darwin-arm64      (Apple Silicon Mac)
dist/privatebox-windows-amd64.exe
dist/privatebox-windows-arm64.exe
```

## Installing (end users)

No extra software is required — each binary is a single self-contained
executable.

**macOS:**
```bash
brew tap privatebox/privatebox
brew trust privatebox/privatebox
brew install privatebox
privatebox
```

**Linux:**
```bash
wget https://github.com/privatebox/privatebox-cli/releases/download/v1.0.0/privatebox_1.0.0_amd64.deb
sudo dpkg -i privatebox_1.0.0_amd64.deb
privatebox
```

**Windows:**
1. Download `https://github.com/privatebox/privatebox-cli/releases/download/v1.0.0/privatebox_windows_amd64.zip`
2. Unzip and Rename it to `privatebox.exe` and place it somewhere on your `PATH`
   (e.g. `C:\Tools\`), or run it directly from any folder.
3. Windows SmartScreen may warn about an unsigned `.exe` the first time —
   click "More info" → "Run anyway", or code-sign the binary to remove
   the warning.

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

$ privatebox --json items sent
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
