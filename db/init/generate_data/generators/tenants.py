from config import COUNCILS, TIMEZONE

from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for council in COUNCILS:
        tenant_id = stable_uuid("tenant", council["key"])
        context["tenants"][council["key"]] = tenant_id
        rows.append(
            {
                "id": tenant_id,
                "slug": council["slug"],
                "name": council["name"],
                "timezone": TIMEZONE,
                "contact_email": council["contact_email"],
                "contact_phone": council["contact_phone"],
                "municipality_code": council["municipality_code"],
                "logo_ref": council["logo_ref"],
            }
        )
    return rows
