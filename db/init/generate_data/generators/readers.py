import hashlib

from generators.base import stable_uuid


# key_hash is the SHA-256 of something that is not a reader key ("lqd_…"), so no key matches it
def generate(context: dict) -> list:
    rows = []
    for facility_key, name in context["seed"]["readers"]:
        digest = hashlib.sha256(f"seed-reader-without-key:{facility_key}".encode()).hexdigest()
        rows.append(
            {
                "id": stable_uuid("checkin_device", facility_key),
                "facility_id": context["facilities"][facility_key]["id"],
                "name": name,
                "key_hash": "\\x" + digest,
                "key_hint": "seed",
            }
        )
    return rows
