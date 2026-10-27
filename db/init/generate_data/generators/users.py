from config import RESIDENTS

from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for r in RESIDENTS:
        user_id = stable_uuid("app_user", r["key"])
        context["residents"][r["key"]] = user_id
        rows.append(
            {
                "id": user_id,
                "idp_subject": f"seed-{r['key']}",
                "email": r["email"],
                "full_name": r["full_name"],
            }
        )
    return rows
