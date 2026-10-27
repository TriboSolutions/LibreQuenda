from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for key, facility in context["facilities"].items():
        for weekdays, opens, closes in facility["schedule"]:
            for weekday in weekdays:
                rows.append(
                    {
                        "id": stable_uuid("opening_hours", f"{key}:{weekday}:{opens}"),
                        "facility_id": facility["id"],
                        "weekday": weekday,
                        "opens_at": opens,
                        "closes_at": closes,
                    }
                )
    return rows
