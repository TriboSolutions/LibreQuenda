"""
Usage (from LibreQuenda/db/init/generate_data/):
    python main.py                  # dev: small, quick to load
    python main.py --profile prod   # production-sized: every table full (see profiles.py)

Outputs:
    ../csv_data/*.csv          — one CSV per table
    ../09_initial_data.sql     — COPY statements that load those CSVs
"""

import argparse
import sys
from pathlib import Path

# Allow imports from this directory regardless of where python is invoked from
sys.path.insert(0, str(Path(__file__).parent))

from csv_writer import write_csv
import profiles
from generators import (
    checkin,
    closures,
    facilities,
    facility_tags,
    opening_hours,
    readers,
    tenants,
    user_home,
    users,
)
from sql_writer import generate_sql

BASE_DIR = Path(__file__).parent
CSV_DIR = BASE_DIR.parent / "csv_data"
SQL_FILE = BASE_DIR.parent / "09_initial_data.sql"


# Shared state passed through every generator so they can cross-reference IDs.
def _empty_context(seed: dict) -> dict:
    return {
        "seed": seed,      # the profile: councils, residents, volumes
        "tenants": {},     # key -> tenant id
        "facilities": {},  # key -> facility config + id
        "residents": {},   # key -> app_user id and home council
    }


# (csv_filename, generator_module) — order matters: FKs must exist before they are referenced
PIPELINE = [
    ("tenant", tenants),
    ("facility", facilities),
    ("facility_tag", facility_tags),
    ("opening_hours", opening_hours),
    ("checkin_device", readers),
    ("facility_checkin", checkin),
    ("app_user", users),
    ("user_home", user_home),
    ("closure", closures),
]


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate LibreQuenda's seed data")
    parser.add_argument("--profile", choices=profiles.PROFILES, default="dev")
    args = parser.parse_args()
    seed = profiles.load(args.profile)

    print("=" * 52)
    print(f"  Generating seed data ({args.profile})")
    print("=" * 52)

    context = _empty_context(seed)
    # the CSVs of the other profile must not linger
    for old in CSV_DIR.glob("*.csv"):
        old.unlink()

    print("\n-- CSVs ----------------------------------------")
    for table_name, gen_module in PIPELINE:
        rows = gen_module.generate(context)
        write_csv(rows, CSV_DIR / f"{table_name}.csv")

    print("\n-- SQL -----------------------------------------")
    generate_sql(CSV_DIR, SQL_FILE, seed)

    print("=" * 52)


if __name__ == "__main__":
    main()
