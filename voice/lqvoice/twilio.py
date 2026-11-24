"""Twilio Programmable Voice: the webhook's signature check and the TwiML that streams the call to us."""

import base64
import hashlib
import hmac
from xml.sax.saxutils import quoteattr


def valid_signature(auth_token: str, url: str, params: dict[str, str], signature: str) -> bool:
    """Twilio signs the webhook URL followed by every form field, sorted by name (HMAC-SHA1)."""
    payload = url + "".join(f"{key}{params[key]}" for key in sorted(params))
    expected = base64.b64encode(hmac.new(auth_token.encode(), payload.encode(), hashlib.sha1).digest()).decode()
    return hmac.compare_digest(expected, signature or "")


def stream_twiml(stream_url: str, ticket: str) -> str:
    return (
        '<?xml version="1.0" encoding="UTF-8"?><Response><Connect>'
        f'<Stream url={quoteattr(stream_url)}><Parameter name="ticket" value={quoteattr(ticket)}/></Stream>'
        "</Connect></Response>"
    )
