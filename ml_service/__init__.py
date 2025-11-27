"""Expose package namespace for Docker imports."""

from .config import Settings  # noqa: F401
from .service import app  # noqa: F401

__all__ = ["Settings", "app"]

