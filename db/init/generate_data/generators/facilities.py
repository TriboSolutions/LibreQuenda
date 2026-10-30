from config import COUNCILS

from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for council in COUNCILS:
        for f in council["facilities"]:
            key = f"{council['key']}/{f['key']}"
            facility_id = stable_uuid("facility", key)
            context["facilities"][key] = {"id": facility_id, **f}
            rows.append(
                {
                    "id": facility_id,
                    "tenant_id": context["tenants"][council["key"]],
                    "slug": f["slug"],
                    "name": f["name"],
                    "description": f["description"],
                    "address": f["address"],
                    "image_ref": f["image_ref"],
                    "show_weather": f.get("show_weather", False),
                    "latitude": f["latitude"],
                    "longitude": f["longitude"],
                    "capacity": f["capacity"],
                    "slot_minutes": f["slot_minutes"],
                    # every facility uses its council's booking window (custom_rules off,
                    # the window columns at the same defaults as tenant_booking_rules)
                    "usage_rules_text": f["usage_rules_text"],
                }
            )
    return rows
