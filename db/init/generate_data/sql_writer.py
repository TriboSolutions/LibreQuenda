import csv
import os
from pathlib import Path

from config import TIMEZONE

# Path where csv_data/ lives inside the Docker container
CONTAINER_CSV_DIR = "/docker-initdb-staging/csv_data"

# Must follow FK dependency order
TABLE_ORDER = [
    "tenant",
    "facility",
    "facility_tag",
    "opening_hours",
    "checkin_device",
    "facility_checkin",
    "app_user",
    "user_home",
    "closure",
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
    # where each resident books: only a temporary table, read by the SQL below
    "user_home": None,
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
    "tenant_id": "UUID",
}

# Generated in SQL once the CSV tables are in, in this order: (file, what it adds).
# {LOCAL_DAY}, {TIMEZONE} and the profile's numbers (profiles.py) are filled in.
HERE = Path(__file__).parent
SQL_PARTS = [
    ("events.sql", "the councils' events  (before the upcoming bookings fill around them)"),
    ("upcoming.sql", "upcoming bookings  (today and the next days, staff ones too)"),
    ("public.sql", "public and shared bookings  (notes, invite links, people and guests who joined)"),
    ("history.sql", "booking history"),
    ("issues.sql", "reported problems  (the councils' maintenance lists)"),
    ("participation.sql", "announcements and polls  (with votes), and problems fixed lately"),
    ("audit.sql", "audit log"),
    ("notifications.sql", "notifications  (already emailed: nothing goes out)"),
]
TESTER_SQL = HERE / "tester.sql"
ENV_FILE = Path(__file__).parent.parent.parent.parent / ".env"


def tester_email() -> str:
    """Who tests locally: TESTER_EMAIL, else the Zitadel admin of the repo's .env; "" for nobody."""
    email = os.environ.get("TESTER_EMAIL", "")
    if not email and ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            if line.startswith("ZITADEL_ADMIN_EMAIL="):
                email = line.split("=", 1)[1].strip().strip('"').strip("'")
    return email


def superadmin_email() -> str:
    """The first superadministrator, as the stack sees it: SUPERADMIN_EMAIL, else the Zitadel admin."""
    email = os.environ.get("SUPERADMIN_EMAIL", "")
    if not email and ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            if line.startswith("SUPERADMIN_EMAIL="):
                email = line.split("=", 1)[1].strip().strip('"').strip("'")
    return email or tester_email()


def generate_zero_sql(sql_file: Path) -> None:
    """The zero seed: only the superadministrator, taken over at their first sign-in (AdoptSeedTester)."""
    email = superadmin_email()
    if not email:
        raise SystemExit("zero: no SUPERADMIN_EMAIL or ZITADEL_ADMIN_EMAIL in the environment or .env")
    quoted = email.replace("'", "''")
    sql_file.write_text(
        "-- ============================================================\n"
        "--  INITIAL DATA  (auto-generated — do not edit manually)\n"
        "--  Profile: zero. Re-generate with:  python generate_data/main.py --profile zero\n"
        "-- ============================================================\n\n"
        "-- the superadministrator, and nothing more\n"
        "INSERT INTO app_user (idp_subject, email, full_name, email_verified)\n"
        f"VALUES ('seed-tester', '{quoted}', 'Superadministrador', TRUE);\n"
        "INSERT INTO platform_admin (user_id) SELECT id FROM app_user WHERE idp_subject = 'seed-tester';\n",
        encoding="utf-8",
    )
    print(f"  [sql]     superadmin  ({email})")
    print(f"\n  Written -> {sql_file.relative_to(sql_file.parent.parent.parent)}")


def generate_sql(csv_dir: Path, sql_file: Path, seed: dict) -> None:
    lines = [
        "-- ============================================================",
        "--  INITIAL DATA  (auto-generated — do not edit manually)",
        f"--  Profile: {seed['name']}. Re-generate with:  python generate_data/main.py --profile {seed['name']}",
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
            if RELATIVE_TABLES[table]:
                lines.append(RELATIVE_TABLES[table])
        else:
            lines.append(
                f'COPY "{table}" ({col_list})\n'
                f"FROM '{container_path}'\n"
                f"WITH (FORMAT CSV, HEADER TRUE);"
            )

        lines.append("")

        print(f"  [sql]     {table}  ({row_count} rows)")

    values = {"LOCAL_DAY": LOCAL_DAY, "TIMEZONE": TIMEZONE, **{k: str(v) for k, v in seed["sql"].items()}}

    def fill(text: str) -> str:
        for key, value in values.items():
            text = text.replace("{" + key + "}", value)
        return text

    # the tester first: their booking takes the first free turn, before the generated ones fill them
    email = tester_email()
    if email:
        lines.append("-- the local tester (tester.sql)")
        lines.append(fill(TESTER_SQL.read_text(encoding="utf-8")).replace("{TESTER_EMAIL}", email.replace("'", "''")))
        print(f"  [sql]     tester  ({email}, a booking from the next full hour)")

    for name, what in SQL_PARTS:
        lines.append(f"-- {name}")
        lines.append(fill((HERE / name).read_text(encoding="utf-8")))
        print(f"  [sql]     {what}")

    sql_file.write_text("\n".join(lines), encoding="utf-8")
    print(f"\n  Written -> {sql_file.relative_to(sql_file.parent.parent.parent)}")
