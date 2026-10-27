import csv
from pathlib import Path


def write_csv(rows: list, filepath: Path) -> None:
    if not rows:
        print(f"  [skip]    {filepath.name}  (no rows)")
        return

    filepath.parent.mkdir(parents=True, exist_ok=True)

    with open(filepath, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=rows[0].keys())
        writer.writeheader()
        writer.writerows(rows)

    print(f"  [csv]     {filepath.name}  ({len(rows)} rows)")
