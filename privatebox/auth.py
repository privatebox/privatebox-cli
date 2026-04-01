from __future__ import annotations

import json
import os
from pathlib import Path
from typing import Optional

try:
    import keyring  # type: ignore
except Exception:  # pragma: no cover
    keyring = None

SERVICE_NAME = "privatebox-cli"
TOKEN_ENV = "PRIVATEBOX_TOKEN"
TOKEN_FILE_ENV = "PRIVATEBOX_TOKEN_FILE"
TOKEN_STORE_ENV = "PRIVATEBOX_TOKEN_STORE"


class TokenStore:
    """Store auth token in keyring or a local file."""

    def __init__(self, backend: str = "auto", profile: str = "default") -> None:
        self.profile = profile
        self.backend = backend.lower()
        self.file_path = Path(
            os.getenv(TOKEN_FILE_ENV, os.path.expanduser("~/.privatebox/token.json"))
        )

    def _use_keyring(self) -> bool:
        if self.backend == "keyring":
            return keyring is not None
        if self.backend == "file":
            return False
        return keyring is not None

    def save(self, token: str) -> None:
        if self._use_keyring():
            keyring.set_password(SERVICE_NAME, self.profile, token)
            return
        self.file_path.parent.mkdir(parents=True, exist_ok=True)
        self.file_path.write_text(
            json.dumps({"profile": self.profile, "token": token}) + "\n",
            encoding="utf-8",
        )
        os.chmod(self.file_path, 0o600)

    def get(self) -> Optional[str]:
        env_token = os.getenv(TOKEN_ENV)
        if env_token:
            return env_token

        if self._use_keyring():
            return keyring.get_password(SERVICE_NAME, self.profile)

        if not self.file_path.exists():
            return None
        try:
            data = json.loads(self.file_path.read_text(encoding="utf-8"))
            return data.get("token")
        except Exception:
            return self.file_path.read_text(encoding="utf-8").strip() or None

    def clear(self) -> None:
        if self._use_keyring():
            try:
                keyring.delete_password(SERVICE_NAME, self.profile)
            except Exception:
                pass
            return
        if self.file_path.exists():
            self.file_path.unlink()
