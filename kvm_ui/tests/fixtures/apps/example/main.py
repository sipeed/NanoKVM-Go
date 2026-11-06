"""Minimal App fixture used by the native simulator tests."""

from appbase import app


@app()
def main(ctx):
    ctx.display.clear()
