import json


def generate(context: dict) -> list:
    """Entry codes and staff confirmation (the readers are checkin_device)."""
    rows = []
    for facility_key, entry_code, contacts in context["seed"]["checkin"]:
        rows.append(
            {
                "facility_id": context["facilities"][facility_key]["id"],
                "entry_code": entry_code,
                "staff_confirm": contacts is not None,
                "staff_contacts": json.dumps(contacts or [], ensure_ascii=False),
            }
        )
    return rows
