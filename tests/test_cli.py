from typer.testing import CliRunner

from privatebox import __version__
from privatebox.cli import app

runner = CliRunner()


def test_version_flag() -> None:
    result = runner.invoke(app, ["--version"])
    assert result.exit_code == 0
    assert __version__ in result.stdout


def test_status_without_token_fails_cleanly() -> None:
    result = runner.invoke(app, ["--json", "status"])
    assert result.exit_code == 1
    assert '"error"' in result.stdout
