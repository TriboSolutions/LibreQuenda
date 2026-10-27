from config import BOOKINGS

from generators.base import add_minutes, stable_uuid, turns


# Relative rows: converted to timestamps when the database is created (see sql_writer.py)
def generate(context: dict) -> list:
    rows = []
    for facility_key, resident_key, day_offset, start in BOOKINGS:
        facility = context["facilities"][facility_key]
        slot = facility["slot_minutes"]
        valid_starts = {s for _, opens, closes in facility["schedule"] for s in turns(opens, closes, slot)}
        if start not in valid_starts:
            raise ValueError(f"{facility_key} {start} is not one of its {slot}-minute turns")
        rows.append(
            {
                "id": stable_uuid("booking", f"{facility_key}:{day_offset}:{start}"),
                "facility_id": facility["id"],
                "user_id": context["residents"][resident_key],
                "day_offset": day_offset,
                "start_time": start,
                "end_time": add_minutes(start, slot),
            }
        )
    return rows
