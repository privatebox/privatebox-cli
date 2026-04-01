from __future__ import annotations

import json
import os
from typing import Any, Optional

import requests

DEFAULT_BASE_URL = "https://api.privatebox.co.nz/api"
BASE_URL_ENV = "PRIVATEBOX_BASE_URL"
API_KEY_ENV = "PRIVATEBOX_API_KEY"


class PrivateBoxAPIError(RuntimeError):
    def __init__(self, message: str, status_code: int = 0, payload: Optional[dict] = None):
        super().__init__(message)
        self.status_code = status_code
        self.payload = payload or {}


class PrivateBoxClient:
    def __init__(self, base_url: Optional[str] = None, token: Optional[str] = None):
        self.base_url = (base_url or os.getenv(BASE_URL_ENV) or DEFAULT_BASE_URL).rstrip("/")
        self.token = token

    def _headers(self) -> dict[str, str]:
        headers = {"Accept": "application/json", "Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        api_key = os.getenv(API_KEY_ENV)
        if api_key:
            headers["X-API-Key"] = api_key
        return headers

    def request(self, method: str, path: str, **kwargs: Any) -> dict[str, Any]:
        url = f"{self.base_url}/{path.lstrip('/')}"
        timeout = kwargs.pop("timeout", 30)
        extra_headers = kwargs.pop("headers", None) or {}
        headers = {**self._headers(), **extra_headers}
        try:
            response = requests.request(method.upper(), url, headers=headers, timeout=timeout, **kwargs)
        except requests.RequestException as exc:
            raise PrivateBoxAPIError(f"Network error calling {url}: {exc}") from exc

        try:
            data = response.json()
        except ValueError:
            data = {"raw": response.text}

        if response.status_code >= 400:
            msg = data.get("status_message") if isinstance(data, dict) else response.text
            raise PrivateBoxAPIError(str(msg), response.status_code, data if isinstance(data, dict) else {})

        return data if isinstance(data, dict) else {"data": data}

    def get(self, path: str, params: Optional[dict[str, Any]] = None) -> dict[str, Any]:
        return self.request("GET", path, params=params)

    def post(self, path: str, payload: Optional[dict[str, Any]] = None) -> dict[str, Any]:
        return self.request("POST", path, data=json.dumps(payload or {}))

    def put(self, path: str, payload: Optional[dict[str, Any]] = None) -> dict[str, Any]:
        return self.request("PUT", path, data=json.dumps(payload or {}))
