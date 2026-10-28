def generate(context: dict) -> list:
    """Which tags each facility carries, by tag slug: the catalogue comes from 06_tags.sql."""
    rows = []
    for facility in context["facilities"].values():
        for slug in facility.get("tags", []):
            rows.append({"facility_id": facility["id"], "tag_slug": slug})
    return rows
