"""One-time tickets: what a call may do, handed over before its WebSocket opens.

The browser posts its access token to /session and opens the WebSocket with the
ticket it gets back, so the token never travels in a URL (which proxies log).
"""

import secrets
import time
from dataclasses import dataclass, field


@dataclass(frozen=True)
class Ticket:
    kind: str  # "web" or "phone"
    headers: dict[str, str] = field(repr=False)  # for LibreQuenda's MCP server
    signed_in: bool = False


class Tickets:
    def __init__(self, ttl: float = 30.0):
        self._ttl = ttl
        self._by_id: dict[str, tuple[float, Ticket]] = {}

    def issue(self, ticket: Ticket) -> str:
        self._expire()
        key = secrets.token_urlsafe(24)
        self._by_id[key] = (time.monotonic() + self._ttl, ticket)
        return key

    def redeem(self, key: str | None) -> Ticket | None:
        """The ticket, once: a second use, or one too late, gets nothing."""
        self._expire()
        if not key:
            return None
        entry = self._by_id.pop(key, None)
        return entry[1] if entry else None

    def pending(self) -> int:
        self._expire()
        return len(self._by_id)

    def _expire(self) -> None:
        now = time.monotonic()
        for key in [k for k, (until, _) in self._by_id.items() if until < now]:
            del self._by_id[key]
