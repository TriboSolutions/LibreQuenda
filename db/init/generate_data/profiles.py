"""
Two seeds from the same generator:

  dev   small and quick to load: three councils, ~60 residents, a week of
        upcoming bookings. What local development and the screenshots need.
  zero  nothing but the superadministrator (SUPERADMIN_EMAIL, else the Zitadel
        admin of .env): no councils, facilities, residents or bookings.
  prod  production-sized: thirteen councils, ~120 facilities, thousands of
        residents, two weeks of upcoming bookings, problems, notices and log.
        For demos, load testing and indicators that look like a real year.

Both: residents live in one council (a person books in one council at a time),
upcoming bookings respect the per-person limits, some are public with a note,
others shared by invite link, with people and guests who joined.
"""

import random

import prod_councils
from config import CHECKIN, CLOSURES, COUNCILS, READERS, RESIDENTS
from people import residents

PROFILES = ("dev", "prod", "zero")


def load(name: str) -> dict:
    if name == "dev":
        councils = COUNCILS
        return {
            "name": "dev",
            "councils": councils,
            "residents": residents(60, [c["key"] for c in councils], RESIDENTS, seed=1),
            "closures": CLOSURES,
            "readers": READERS,
            "checkin": CHECKIN,
            "sql": {
                "HISTORY_DAYS": 400,
                "UPCOMING_DAYS": 7,
                "UPCOMING_FILL": 0.75,       # how busy the next days are, against the history's demand
                "MAX_UPCOMING_PER_PERSON": 3,
                "STAFF_SHARE": 0.06,         # booked by staff for someone without an account
                "PUBLIC_SHARE": 0.45,        # upcoming bookings anyone can join
                "INVITE_SHARE": 0.25,        # …and shared by invite link only
                "EXTRA_ISSUES": 12,
                "EXTRA_AUDIT": 25,
            },
        }
    if name == "prod":
        extra = prod_councils.councils()
        councils = COUNCILS + extra
        closures, readers, checkin = list(CLOSURES), list(READERS), list(CHECKIN)
        rng = random.Random(11)
        for council in extra:
            for f in council["facilities"]:
                key = f"{council['key']}/{f['key']}"
                kind = f["key"].rsplit("-", 1)[0]
                if kind in ("pavillon", "piscina", "ximnasio", "atletismo", "futbol7"):
                    readers.append((key, "Control de acceso" if kind != "futbol7" else "Porta do campo"))
                if kind in ("padel", "squash", "fronton", "rocodromo"):
                    checkin.append((key, True, None))
                if kind == "pavillon":
                    checkin.append((key, False, [{"name": "Conserxería", "phone": "", "place": "Mostrador da entrada"}]))
                opens, closes = f["schedule"][0][1], f["schedule"][0][2]
                if rng.random() < 0.35:
                    closures.append((key, -rng.randint(5, 380), opens, closes, rng.choice(PAST_REASONS)))
                if rng.random() < 0.12:
                    closures.append((key, rng.randint(1, 13), opens, closes, rng.choice(UPCOMING_REASONS)))
        return {
            "name": "prod",
            "councils": councils,
            "residents": residents(4500, [c["key"] for c in councils], RESIDENTS, seed=2),
            "closures": closures,
            "readers": readers,
            "checkin": checkin,
            "sql": {
                "HISTORY_DAYS": 400,
                "UPCOMING_DAYS": 14,
                "UPCOMING_FILL": 0.8,
                "MAX_UPCOMING_PER_PERSON": 3,
                "STAFF_SHARE": 0.08,
                "PUBLIC_SHARE": 0.3,
                "INVITE_SHARE": 0.3,
                "EXTRA_ISSUES": 400,
                "EXTRA_AUDIT": 600,
            },
        }
    raise ValueError(f"unknown profile {name!r}: one of {', '.join(PROFILES)}")


PAST_REASONS = ["Mantemento", "Limpeza a fondo", "Revisión das instalacións", "Competición federada",
                "Avaría na iluminación", "Pintura das liñas", "Evento municipal"]
UPCOMING_REASONS = ["Mantemento programado", "Competición federada", "Evento municipal", "Revisión anual"]
