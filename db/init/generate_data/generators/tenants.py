from config import TENANT, TIMEZONE

from generators.base import stable_uuid


def generate(context: dict) -> list:
    tenant_id = stable_uuid("tenant", TENANT["key"])
    context["tenant_id"] = tenant_id
    return [
        {
            "id": tenant_id,
            "slug": TENANT["slug"],
            "name": TENANT["name"],
            "timezone": TIMEZONE,
            "contact_email": TENANT["contact_email"],
            "contact_phone": TENANT["contact_phone"],
            "logo_ref": TENANT["logo_ref"],
        }
    ]
