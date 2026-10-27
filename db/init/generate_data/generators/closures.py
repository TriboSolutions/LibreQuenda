from config import CLOSURES

from generators.base import stable_uuid


# Relative rows: converted to timestamps when the database is created (see sql_writer.py)
def generate(context: dict) -> list:
    rows = []
    for facility_key, day_offset, start, end, reason in CLOSURES:
        rows.append(
            {
                "id": stable_uuid("closure", f"{facility_key}:{day_offset}:{start}"),
                "facility_id": context["facilities"][facility_key]["id"],
                "day_offset": day_offset,
                "start_time": start,
                "end_time": end,
                "reason": reason,
            }
        )
    return rows
