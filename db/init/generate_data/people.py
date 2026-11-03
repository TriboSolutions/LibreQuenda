"""
Made-up residents with common Galician names. Seeded random numbers: the same
people every time. Emails are on example.com, so nothing ever reaches anyone.
"""

import random
import unicodedata

GIVEN_NAMES = [
    "Uxía", "Brais", "Antía", "Xoán", "Iago", "Noa", "Martín", "Lucía", "Breixo", "Sabela",
    "Roi", "Carme", "Anxo", "Aldara", "Pablo", "María", "Hugo", "Olaia", "Daniel", "Iria",
    "Xiana", "Martiño", "Alba", "Pedro", "Laura", "Xabier", "Sara", "Diego", "Paula", "Adrián",
    "Carla", "Manuel", "Nerea", "Xosé", "Andrea", "Alejandro", "Lorena", "David", "Rosa", "Óscar",
    "Ánxela", "Fernando", "Beatriz", "Rubén", "Eva", "Álvaro", "Elena", "Sergio", "Marta", "Xurxo",
    "Cristina", "Raúl", "Alicia", "Santiago", "Lois", "Rocío", "Tomás", "Begoña", "Antón", "Patricia",
    "Ramón", "Silvia", "Alexandre", "Inés", "Miguel", "Sofía", "Carlos", "Claudia", "Javier", "Irene",
    "Gonzalo", "Mónica", "Rodrigo", "Raquel", "Enrique", "Montserrat", "Iván", "Susana", "Marcos", "Natalia",
    "Xulio", "Dolores", "Bieito", "Uxío", "Rosalía", "Vicente", "Erea", "Fiz", "Icía", "Lara",
]

SURNAMES = [
    "Pereira", "Otero", "Vázquez", "Castro", "Fernández", "Rodríguez", "López", "Gómez", "Souto", "Rial",
    "Barreiro", "Dopico", "Varela", "Pena", "Iglesias", "Fraga", "Seoane", "Carballo", "Rey", "Mosquera",
    "García", "González", "Martínez", "Sánchez", "Díaz", "Pérez", "Álvarez", "Romero", "Rivas", "Lorenzo",
    "Domínguez", "Novoa", "Vilar", "Couto", "Outeiro", "Gil", "Méndez", "Prieto", "Losada", "Cid",
    "Blanco", "Míguez", "Teixeira", "Bouzas", "Calvo", "Abal", "Baamonde", "Cabanas", "Paz", "Lema",
    "Ferreiro", "Regueiro", "Conde", "Sueiro", "Figueiras", "Carreira", "Mariño", "Pazos", "Torres", "Ríos",
    "Freire", "Nogueira", "Villar", "Quintela", "Amoedo", "Lago", "Rama", "Cancela", "Taboada", "Moure",
]

LANGUAGES = ["gl"] * 6 + ["es"] * 3 + ["en"]


def _ascii(text: str) -> str:
    return unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode().lower().replace(" ", "")


def residents(count: int, homes: list, named: list, seed: int) -> list:
    """`named` first (hand-written, used by the SQL files by key), then made-up people up to `count`.
    Each one lives in one council of `homes`, round robin: a person books in one council at a time."""
    rng = random.Random(seed)
    people = []
    taken = set()
    for i, person in enumerate(named):
        taken.add(person["key"])
        people.append({**person, "home": person.get("home") or homes[i % len(homes)]})
    while len(people) < count:
        given = rng.choice(GIVEN_NAMES)
        first, second = rng.sample(SURNAMES, 2)
        key = f"{_ascii(given)}.{_ascii(first)}.{_ascii(second)}"
        if key in taken:
            continue
        taken.add(key)
        people.append({"key": key, "full_name": f"{given} {first} {second}", "home": homes[len(people) % len(homes)]})
    for person in people:
        person.setdefault("email", f"{person['key']}@example.com")
        # ages 16 to 75, most between 25 and 55
        age = max(16, min(75, int(rng.gauss(40, 13))))
        person["birth_date"] = f"{2026 - age}-{rng.randint(1, 12):02d}-{rng.randint(1, 28):02d}"
        person["preferred_language"] = rng.choice(LANGUAGES)
        person["email_reminders"] = rng.random() < 0.8
    return people
