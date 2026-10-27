import uuid
from datetime import datetime, timedelta

# Deterministic ids: re-running the generator produces the same CSVs
NAMESPACE = uuid.UUID("6f0c7a52-2f0e-4c9e-9d1a-7b5d3c1e8a40")


def stable_uuid(kind: str, key: str) -> str:
    return str(uuid.uuid5(NAMESPACE, f"{kind}:{key}"))


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
