from __future__ import annotations

import json
from pathlib import Path
from typing import Annotated, Any, Optional

import typer

from . import __version__
from .auth import TOKEN_STORE_ENV, TokenStore
from .client import BASE_URL_ENV, PrivateBoxAPIError, PrivateBoxClient

app = typer.Typer(help="PrivateBox CLI. Human-friendly by default, JSON when --json is set.")
auth_app = typer.Typer(help="Authentication commands")
items_app = typer.Typer(help="Mail item commands")
order_app = typer.Typer(help="Order commands")
app.add_typer(auth_app, name="auth")
app.add_typer(items_app, name="items")
app.add_typer(order_app, name="order")


class Ctx:
    def __init__(self, json_output: bool, token_store: str):
        self.json_output = json_output
        self.store = TokenStore(backend=token_store)


def _print(data: Any, json_output: bool) -> None:
    if json_output:
        typer.echo(json.dumps(data, indent=2, default=str))
        return
    if isinstance(data, dict) and "status_message" in data:
        typer.echo(f"✓ {data['status_message']}")
    typer.echo(json.dumps(data, indent=2, default=str))


def _fail(err: Exception, json_output: bool) -> None:
    if isinstance(err, PrivateBoxAPIError):
        payload = {"error": str(err), "status_code": err.status_code, "details": err.payload}
    else:
        payload = {"error": str(err)}
    if json_output:
        typer.echo(json.dumps(payload, indent=2))
    else:
        typer.secho(f"Error: {payload['error']}", fg=typer.colors.RED)
    raise typer.Exit(code=1)


def _client(ctx: Ctx, token_override: Optional[str] = None) -> PrivateBoxClient:
    token = token_override or ctx.store.get()
    return PrivateBoxClient(token=token)


def _version_callback(value: bool) -> None:
    if value:
        typer.echo(f"privatebox-cli {__version__}")
        raise typer.Exit()


@app.callback()
def main(
    ctx: typer.Context,
    json_output: Annotated[bool, typer.Option("--json", help="Output JSON for scripts/agents")] = False,
    token_store: Annotated[
        str,
        typer.Option(
            "--token-store",
            help="Token backend: auto|keyring|file (env PRIVATEBOX_TOKEN_STORE)",
            envvar=TOKEN_STORE_ENV,
        ),
    ] = "auto",
    version: Annotated[bool, typer.Option("--version", callback=_version_callback, is_eager=True)] = False,
) -> None:
    """Use PRIVATEBOX_BASE_URL to override API base URL."""
    _ = version
    ctx.obj = Ctx(json_output, token_store)


@auth_app.command("login")
def login(
    ctx: typer.Context,
    email: Annotated[str, typer.Option(help="Account email")],
    password: Annotated[str, typer.Option(prompt=True, hide_input=True, help="Account password")],
    device_id: Annotated[str, typer.Option(help="Device ID for trusted device flow")] = "privatebox-cli",
) -> None:
    """Authenticate and save Bearer token.

    Example:
      privatebox auth login --email you@example.com
    """
    c = _client(ctx.obj, token_override=None)
    try:
        data = c.request("POST", "/login", data=json.dumps({"email": email, "password": password}), headers={"X-DeviceID": device_id, **c._headers()})
        token = data.get("token")
        if not token:
            raise PrivateBoxAPIError("Login succeeded but token missing", payload=data)
        ctx.obj.store.save(token)
        if data.get("verification_required"):
            typer.secho("Login ok, but verification code required. Run: privatebox auth verify-code --code <123456>", fg=typer.colors.YELLOW)
        _print({"status_message": "Logged in", "account": data.get("email"), "verification_required": data.get("verification_required", False)}, ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@auth_app.command("verify-code")
def verify_code(
    ctx: typer.Context,
    code: Annotated[str, typer.Option(help="One-time verification code")],
) -> None:
    """Validate the one-time code sent by email."""
    c = _client(ctx.obj)
    try:
        _print(c.post("/user/validate_code", {"code": code}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@auth_app.command("logout")
def logout(ctx: typer.Context) -> None:
    """Remove locally stored token."""
    ctx.obj.store.clear()
    _print({"status_message": "Logged out"}, ctx.obj.json_output)


@app.command("status")
def status(ctx: typer.Context) -> None:
    """Check token validity by fetching current user profile."""
    c = _client(ctx.obj)
    try:
        _print(c.get("/user"), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@items_app.command("inbox")
def items_inbox(
    ctx: typer.Context,
    page: int = 1,
    items_per_page: int = 10,
    search: str = "",
) -> None:
    """List current inbox items (/items)."""
    c = _client(ctx.obj)
    try:
        _print(c.get("/items", {"page": page, "items_per_page": items_per_page, "search": search}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@items_app.command("sent")
def items_sent(ctx: typer.Context, page: int = 1, items_per_page: int = 10) -> None:
    """List sent items (/items/sent)."""
    c = _client(ctx.obj)
    try:
        _print(c.get("/items/sent", {"page": page, "items_per_page": items_per_page}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@items_app.command("scanned")
def items_scanned(ctx: typer.Context, page: int = 1, items_per_page: int = 10) -> None:
    """List scanned items (/items/scanned)."""
    c = _client(ctx.obj)
    try:
        _print(c.get("/items/scanned", {"page": page, "items_per_page": items_per_page}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@order_app.command("send-info")
def order_send_info(ctx: typer.Context) -> None:
    """Show order/send info text."""
    c = _client(ctx.obj)
    try:
        _print(c.get("/order/send/info"), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@order_app.command("scan-cost")
def order_scan_cost(ctx: typer.Context, item_count: Annotated[int, typer.Option(help="Number of items")]) -> None:
    """Estimate scan cost for N items."""
    c = _client(ctx.obj)
    try:
        _print(c.post("/order/scan/cost", {"item_count": item_count}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@order_app.command("scan")
def order_scan(
    ctx: typer.Context,
    items: Annotated[list[int], typer.Option(help="Repeat --items for each item id")],
    destroy: Annotated[bool, typer.Option(help="Queue item destruction after scan")] = False,
) -> None:
    """Request scans for item IDs."""
    c = _client(ctx.obj)
    try:
        _print(c.post("/order/scan", {"items": items, "destroy": destroy}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


@order_app.command("send-cost")
def order_send_cost(
    ctx: typer.Context,
    items: Annotated[list[int], typer.Option(help="Repeat --items for each item id")],
    country_iso: Annotated[str, typer.Option(help="Destination country code, e.g. NZ")],
    address_json: Annotated[Optional[Path], typer.Option(help="Optional JSON file for address object")] = None,
) -> None:
    """Estimate forwarding cost for item IDs.

    Example address JSON:
      {"address":"123 Test St","city":"Wellington","country_iso":"NZ","post_code":"6011"}
    """
    c = _client(ctx.obj)
    address: dict[str, Any] = {"country_iso": country_iso}
    if address_json:
        address.update(json.loads(address_json.read_text(encoding="utf-8")))
    try:
        _print(c.post("/order/send/cost", {"items": items, "address": address}), ctx.obj.json_output)
    except Exception as exc:
        _fail(exc, ctx.obj.json_output)


if __name__ == "__main__":
    app()
