"""
Usage (from LibreQuenda/db/init/generate_data/):
    python main.py

Outputs:
    ../csv_data/*.csv          — one CSV per table
    ../09_initial_data.sql     — COPY statements that load those CSVs
"""

import sys
from pathlib import Path

# Allow imports from this directory regardless of where python is invoked from
sys.path.insert(0, str(Path(__file__).parent))

from csv_writer import write_csv
from generators import (
    bookings,
    closures,
    facilities,
    opening_hours,
    readers,
    tenants,
    users,
)
from sql_writer import generate_sql

BASE_DIR = Path(__file__).parent
CSV_DIR = BASE_DIR.parent / "csv_data"
SQL_FILE = BASE_DIR.parent / "09_initial_data.sql"


# Shared state passed through every generator so they can cross-reference IDs.
def _empty_context() -> dict:
    return {
        "tenant_id": None,
        "facilities": {},  # key -> facility config + id
        "residents": {},   # key -> app_user id
    }


# (csv_filename, generator_module) — order matters: FKs must exist before they are referenced
PIPELINE = [
    ("tenant", tenants),
    ("facility", facilities),
    ("opening_hours", opening_hours),
    ("checkin_device", readers),
    ("app_user", users),
    ("closure", closures),
    ("booking", bookings),
]


def main() -> None:
    print("=" * 52)
    print("  Generating seed data")
    print("=" * 52)

    context = _empty_context()

    print("\n-- CSVs ----------------------------------------")
    for table_name, gen_module in PIPELINE:
        rows = gen_module.generate(context)
        write_csv(rows, CSV_DIR / f"{table_name}.csv")

    print("\n-- SQL -----------------------------------------")
    generate_sql(CSV_DIR, SQL_FILE)

    print("=" * 52)


if __name__ == "__main__":
    main()
