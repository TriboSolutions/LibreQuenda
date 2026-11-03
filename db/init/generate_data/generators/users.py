from generators.base import stable_uuid


def generate(context: dict) -> list:
    rows = []
    for r in context["seed"]["residents"]:
        user_id = stable_uuid("app_user", r["key"])
        context["residents"][r["key"]] = {"id": user_id, "home": r["home"]}
        rows.append(
            {
                "id": user_id,
                "idp_subject": f"seed-{r['key']}",
                "email": r["email"],
                "full_name": r["full_name"],
                "email_verified": True,  # booking needs a verified email
                "birth_date": r["birth_date"],  # …and a birth date
                "preferred_language": r["preferred_language"],
                "email_reminders": r["email_reminders"],
                "created_at": "2025-06-01 10:00:00+02",
            }
        )
    return rows
