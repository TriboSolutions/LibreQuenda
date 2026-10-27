from config import FACILITIES

from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for f in FACILITIES:
        facility_id = stable_uuid("facility", f["key"])
        context["facilities"][f["key"]] = {"id": facility_id, **f}
        rows.append(
            {
                "id": facility_id,
                "tenant_id": context["tenant_id"],
                "slug": f["slug"],
                "name": f["name"],
                "description": f["description"],
                "sport": f["sport"],
                "address": f["address"],
                "image_ref": f["image_ref"],
                "latitude": f["latitude"],
                "longitude": f["longitude"],
                "capacity": f["capacity"],
                "slot_minutes": f["slot_minutes"],
                "max_advance_days": f["max_advance_days"],
                "min_cancel_hours": f["min_cancel_hours"],
                "max_active_bookings_per_user": f["max_active_bookings_per_user"],
                "usage_rules_text": f["usage_rules_text"],
            }
        )
    return rows
