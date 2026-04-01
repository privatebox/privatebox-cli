# privatebox-cli

## Install

```bash
pip install .
# or
pipx install .
```

## Quick start

```bash
# Optional: override API base URL
export PRIVATEBOX_BASE_URL="https://api.privatebox.co.nz/api"

# Login (stores token in keyring when available)
privatebox auth login --email you@example.com

# Validate one-time code if prompted
privatebox auth verify-code --code 123456

# Check session
privatebox status

# List inbox items
privatebox items inbox

# Request scan for items
privatebox order scan --items 1001 --items 1002 --destroy

# Create a send order (address payload from JSON file)
privatebox order send --items 1001 --receivers-name "Jane Doe" --service-id 7 --address-json address.json

# Queue item destruction
privatebox order destroy --items 1003 --items 1004

# Reference inputs
privatebox meta countries
privatebox meta frequency

# Script-friendly output
privatebox --json items sent
```

## Tests

```bash
pip install -e . pytest
pytest -q
```

## Notes

- Token storage backend: `--token-store auto|keyring|file` or `PRIVATEBOX_TOKEN_STORE`
- Token file path (file backend): `PRIVATEBOX_TOKEN_FILE` (default `~/.privatebox/token.json`)
- For automation, you can inject a token with `PRIVATEBOX_TOKEN`
