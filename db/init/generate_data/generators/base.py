import uuid
from datetime import datetime, timedelta

# Deterministic ids: re-running the generator produces the same CSVs
NAMESPACE = uuid.UUID("6f0c7a52-2f0e-4c9e-9d1a-7b5d3c1e8a40")


def stable_uuid(kind: str, key: str) -> str:
    return str(uuid.uuid5(NAMESPACE, f"{kind}:{key}"))


def council_ref(context: dict, tenant_id: str, ref: str | None) -> str | None:
    """A demo image as the council's own file: "<council id>/<folder>/<file>", in
    the council's private bucket. db/init/seed_media/<folder>/<file> is uploaded
    there by the storage-seed service, from the list main.py writes (refs.txt)."""
    if not ref:
        return ref
    stored = f"{tenant_id}/{ref}"
    context["media_refs"].add(stored)
    return stored


def add_minutes(clock: str, minutes: int) -> str:
    start = datetime.strptime(clock, "%H:%M")
    return (start + timedelta(minutes=minutes)).strftime("%H:%M")


def turns(opens: str, closes: str, slot_minutes: int) -> list:
    """Start times of the bookable turns of one opening window."""
    starts = []
    start = opens
    while add_minutes(start, slot_minutes) <= closes and add_minutes(start, slot_minutes) > start:
        starts.append(start)
        start = add_minutes(start, slot_minutes)
    return starts
