def generate(context: dict) -> list:
    """The council each resident books in. Only for the seed SQL (a temporary table)."""
    return [
        {"user_id": r["id"], "tenant_id": context["tenants"][r["home"]]}
        for r in context["residents"].values()
    ]
