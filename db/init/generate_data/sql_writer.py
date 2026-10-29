import csv
import os
from pathlib import Path

from config import HISTORY_DAYS, TIMEZONE

# Path where csv_data/ lives inside the Docker container
CONTAINER_CSV_DIR = "/docker-initdb-staging/csv_data"

# Must follow FK dependency order
TABLE_ORDER = [
    "tenant",
    "facility",
    "facility_tag",
    "opening_hours",
    "checkin_device",
    "app_user",
    "closure",
    "booking",
]

# Tables loaded into a temporary table first and converted on the way in: a day
# offset and local times become timestamps relative to the day the database is
# created (so the demo always has a realistic "today"), and tag slugs become tag ids.
LOCAL_DAY = f"(now() AT TIME ZONE '{TIMEZONE}')::date"
RELATIVE_TABLES = {
    # by tag slug: the tag ids are made by 06_tags.sql
    "facility_tag": (
        "INSERT INTO facility_tag (facility_id, tag_id)\n"
        "SELECT s.facility_id, t.id\n"
        "FROM seed_facility_tag s JOIN tag t ON t.slug = s.tag_slug;"
    ),
    "closure": (
        "INSERT INTO closure (id, facility_id, starts_at, ends_at, reason)\n"
        "SELECT id, facility_id,\n"
        f"       ({LOCAL_DAY} + day_offset + start_time) AT TIME ZONE '{TIMEZONE}',\n"
        f"       ({LOCAL_DAY} + day_offset + end_time) AT TIME ZONE '{TIMEZONE}',\n"
        "       reason\n"
        "FROM seed_closure;"
    ),
    "booking": (
        "INSERT INTO booking (id, facility_id, user_id, starts_at, ends_at)\n"
        "SELECT id, facility_id, user_id,\n"
        f"       ({LOCAL_DAY} + day_offset + start_time) AT TIME ZONE '{TIMEZONE}',\n"
        f"       ({LOCAL_DAY} + day_offset + end_time) AT TIME ZONE '{TIMEZONE}'\n"
        "FROM seed_booking;"
    ),
}
RELATIVE_COLUMN_TYPES = {
    "id": "UUID",
    "facility_id": "UUID",
    "user_id": "UUID",
    "day_offset": "INT",
    "start_time": "TIME",
    "end_time": "TIME",
    "reason": "TEXT",
    "tag_slug": "TEXT",
}

# Generated in SQL once the schedule is loaded, after the CSV tables
HISTORY_SQL = Path(__file__).parent / "history.sql"
CHECKIN_SQL = Path(__file__).parent / "checkin.sql"
TESTER_SQL = Path(__file__).parent / "tester.sql"
ENV_FILE = Path(__file__).parent.parent.parent.parent / ".env"


def tester_email() -> str:
    """Who tests locally: TESTER_EMAIL, else the Zitadel admin of the repo's .env; "" for nobody."""
    email = os.environ.get("TESTER_EMAIL", "")
    if not email and ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            if line.startswith("ZITADEL_ADMIN_EMAIL="):
                email = line.split("=", 1)[1].strip().strip('"').strip("'")
    return email


def generate_sql(csv_dir: Path, sql_file: Path) -> None:
    lines = [
        "-- ============================================================",
        "--  INITIAL DATA  (auto-generated — do not edit manually)",
        "--  Re-generate by running:  python generate_data/main.py",
        "-- ============================================================",
        "",
    ]

    for table in TABLE_ORDER:
        csv_path = csv_dir / f"{table}.csv"

        if not csv_path.exists():
            print(f"  [skip]    {table}  (CSV not found)")
            continue

        # Read the header to get column names in the correct order
        with open(csv_path, newline="", encoding="utf-8") as f:
            columns = next(csv.reader(f))
            row_count = sum(1 for _ in f)

        col_list = ", ".join(columns)
        container_path = f"{CONTAINER_CSV_DIR}/{table}.csv"

        lines.append(f"-- {table}")
        if table in RELATIVE_TABLES:
            col_defs = ", ".join(f"{c} {RELATIVE_COLUMN_TYPES[c]}" for c in columns)
            lines.append(f"CREATE TEMP TABLE seed_{table} ({col_defs});")
            lines.append(
                f"COPY seed_{table} ({col_list})\n"
                f"FROM '{container_path}'\n"
                f"WITH (FORMAT CSV, HEADER TRUE);"
            )
            lines.append(RELATIVE_TABLES[table])
        else:
            lines.append(
                f'COPY "{table}" ({col_list})\n'
                f"FROM '{container_path}'\n"
                f"WITH (FORMAT CSV, HEADER TRUE);"
            )

        lines.append("")

        print(f"  [sql]     {table}  ({row_count} rows)")

    lines.append("-- ways to check in (checkin.sql)")
    lines.append(CHECKIN_SQL.read_text(encoding="utf-8"))
    print("  [sql]     ways to check in  (staff contacts, entry codes)")

    email = tester_email()
    if email:
        lines.append("-- the local tester (tester.sql)")
        lines.append(TESTER_SQL.read_text(encoding="utf-8").replace("{TESTER_EMAIL}", email.replace("'", "''")))
        print(f"  [sql]     tester  ({email}, a booking from the next full hour)")

    lines.append("-- booking history (history.sql)")
    lines.append(
        HISTORY_SQL.read_text(encoding="utf-8")
        .replace("{LOCAL_DAY}", LOCAL_DAY)
        .replace("{HISTORY_DAYS}", str(HISTORY_DAYS))
        .replace("{TIMEZONE}", TIMEZONE)
    )
    print(f"  [sql]     booking history  ({HISTORY_DAYS} days)")

    sql_file.write_text("\n".join(lines), encoding="utf-8")
    print(f"\n  Written -> {sql_file.relative_to(sql_file.parent.parent.parent)}")
