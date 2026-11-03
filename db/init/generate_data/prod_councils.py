"""
The production-sized seed: the three dev councils (real facilities from
OpenStreetMap, see config.py) plus more real Galician councils whose facilities
are made up from templates. Names, addresses and pins of those are invented
(the pins near each town's centre); only the council names and INE codes are real.
"""

import random

from config import SATURDAY, WEEKDAYS, WEEKEND

# (key, name, INE code, latitude, longitude of the town centre)
TOWNS = [
    ("vigo", "Concello de Vigo", "36057", 42.2328, -8.7226),
    ("pontevedra", "Concello de Pontevedra", "36038", 42.4310, -8.6444),
    ("santiago", "Concello de Santiago de Compostela", "15078", 42.8782, -8.5448),
    ("coruna", "Concello da Coruña", "15030", 43.3623, -8.4115),
    ("ferrol", "Concello de Ferrol", "15036", 43.4832, -8.2369),
    ("naron", "Concello de Narón", "15054", 43.5167, -8.1528),
    ("vilagarcia", "Concello de Vilagarcía de Arousa", "36060", 42.5945, -8.7652),
    ("monforte", "Concello de Monforte de Lemos", "27031", 42.5218, -7.5146),
    ("verin", "Concello de Verín", "32085", 41.9413, -7.4362),
    ("carballo", "Concello de Carballo", "15019", 43.2131, -8.6911),
]

# One facility of each kind; a town gets several, some more than once (numbered)
TEMPLATES = {
    "padel": {
        "name": "Pista de pádel {n}", "sport": "Pádel", "image_ref": "facilities/pista-de-padel-1.jpg",
        "tags": ["padel", "pista", "cuberta", "iluminacion", "vestiarios"],
        "description": "Pista de pádel cuberta do complexo deportivo municipal.",
        "capacity": 4, "slot_minutes": 90, "usage_rules_text": "Calzado de pádel obrigatorio.",
        "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
    },
    "tenis": {
        "name": "Pista de tenis {n}", "sport": "Tenis", "image_ref": "facilities/pista-de-tenis.jpg", "show_weather": True,
        "tags": ["tenis", "pista", "aire-libre", "iluminacion"],
        "description": "Pista de tenis ao aire libre con iluminación.",
        "capacity": 4, "slot_minutes": 60, "usage_rules_text": None,
        "schedule": [(WEEKDAYS, "09:00", "22:00"), (WEEKEND, "10:00", "14:00")],
    },
    "pavillon": {
        "name": "Pavillón municipal {n}", "sport": "Polideportivo", "image_ref": "facilities/pavillon-municipal.jpg",
        "tags": ["futbol-sala", "baloncesto", "balonman", "voleibol", "multiusos", "pavillon", "cuberta", "vestiarios", "accesible"],
        "description": "Pista polideportiva cuberta para fútbol sala, baloncesto, balonmán e voleibol.",
        "capacity": 30, "slot_minutes": 60, "usage_rules_text": "Só calzado de sala.",
        "accessibility_info": "Acceso sen chanzos dende a rúa, ascensor á grada e vestiario adaptado.",
        "schedule": [(WEEKDAYS, "09:00", "23:00"), (WEEKEND, "10:00", "14:00")],
    },
    "piscina": {
        "name": "Piscina municipal, rúa {n}", "sport": "Natación", "image_ref": "facilities/piscina-rua-3.jpg",
        "tags": ["natacion", "piscina", "cuberta", "climatizada", "vestiarios", "accesible"],
        "description": "Rúa da piscina municipal climatizada, reservada para nado libre.",
        "capacity": 2, "slot_minutes": 45, "usage_rules_text": "Gorro obrigatorio. Ducha antes de entrar na auga.",
        "schedule": [(WEEKDAYS, "07:00", "10:00"), (WEEKDAYS, "15:00", "21:45"), (WEEKEND, "10:00", "13:45")],
    },
    "futbol7": {
        "name": "Campo de fútbol 7 {n}", "sport": "Fútbol", "image_ref": "facilities/campo-de-futbol-7.jpg", "show_weather": True,
        "tags": ["futbol", "futbol-7", "campo", "aire-libre", "cesped-artificial", "iluminacion", "vestiarios"],
        "description": "Campo de fútbol 7 de céspede artificial con iluminación e vestiarios.",
        "capacity": 14, "slot_minutes": 90, "usage_rules_text": "Céspede artificial: só bota de tacos de goma.",
        "schedule": [(WEEKDAYS, "17:00", "23:00"), (SATURDAY, "09:00", "13:30")],
    },
    "ximnasio": {
        "name": "Sala de musculación {n}", "sport": "Musculación", "image_ref": "facilities/pavillon-municipal.jpg",
        "tags": ["musculacion", "ximnasia", "sala", "cuberta", "vestiarios", "adultos"],
        "description": "Sala de musculación e cardio do centro deportivo municipal.",
        "capacity": 12, "slot_minutes": 60, "usage_rules_text": "Toalla obrigatoria. Maiores de 16 anos.", "min_age": 16,
        "schedule": [(WEEKDAYS, "07:00", "22:00"), (SATURDAY, "09:00", "14:00")],
    },
    "rocodromo": {
        "name": "Rocódromo municipal {n}", "sport": "Escalada", "image_ref": "facilities/pavillon-municipal.jpg",
        "tags": ["escalada", "sala", "cuberta", "xuvenil", "adultos"],
        "description": "Parede de escalada en bloque, con colchóns e presas renovadas cada tempada.",
        "capacity": 8, "slot_minutes": 90, "usage_rules_text": "Pés de gato obrigatorios. Menores de 14, acompañados.", "min_age": 14,
        "schedule": [(WEEKDAYS, "16:00", "22:00"), (SATURDAY, "10:00", "14:30")],
    },
    "squash": {
        "name": "Pista de squash {n}", "sport": "Squash", "image_ref": "facilities/pista-de-padel-2.jpg",
        "tags": ["squash", "pista", "cuberta", "vestiarios"],
        "description": "Pista de squash cuberta.",
        "capacity": 2, "slot_minutes": 45, "usage_rules_text": "Lentes protectoras recomendadas.",
        "schedule": [(WEEKDAYS, "09:00", "22:00"), (WEEKEND, "10:00", "14:00")],
    },
    "fronton": {
        "name": "Frontón municipal {n}", "sport": "Frontón", "image_ref": "facilities/pavillon-municipal.jpg",
        "tags": ["fronton", "pista", "cuberta", "iluminacion"],
        "description": "Frontón cuberto, para pelota e frontenis.",
        "capacity": 4, "slot_minutes": 60, "usage_rules_text": None,
        "schedule": [(WEEKDAYS, "10:00", "22:00"), (WEEKEND, "10:00", "14:00")],
    },
    "atletismo": {
        "name": "Pista de atletismo {n}", "sport": "Atletismo", "image_ref": "facilities/campo-de-futbol-7.jpg", "show_weather": True,
        "tags": ["atletismo", "pista", "aire-libre", "iluminacion", "vestiarios", "accesible"],
        "description": "Pista de atletismo de oito rúas, para adestramento libre.",
        "capacity": 40, "slot_minutes": 60, "usage_rules_text": "As rúas 1 e 2 son para series; trota polas de fóra.",
        "schedule": [(WEEKDAYS, "07:00", "22:00"), (WEEKEND, "08:00", "14:00")],
    },
}

# What a town has, by size (Vigo and A Coruña the most)
BIG = ["padel", "padel", "padel", "tenis", "tenis", "pavillon", "pavillon", "piscina", "piscina", "piscina",
       "futbol7", "futbol7", "ximnasio", "rocodromo", "squash", "atletismo"]
MEDIUM = ["padel", "padel", "tenis", "pavillon", "piscina", "piscina", "futbol7", "ximnasio", "squash", "fronton"]
SMALL = ["padel", "tenis", "pavillon", "piscina", "futbol7", "fronton"]
SIZES = {"vigo": BIG, "coruna": BIG, "santiago": BIG, "pontevedra": MEDIUM, "ferrol": MEDIUM, "naron": MEDIUM,
         "vilagarcia": MEDIUM, "monforte": SMALL, "verin": SMALL, "carballo": SMALL}

PLACES = ["Complexo deportivo municipal", "Centro deportivo do barrio", "Cidade deportiva", "Instalacións do parque"]


def _slug(text: str) -> str:
    import unicodedata
    plain = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode().lower()
    return "-".join("".join(c if c.isalnum() else " " for c in plain).split())


def councils(seed: int = 7) -> list:
    rng = random.Random(seed)
    out = []
    for key, name, ine, lat, lon in TOWNS:
        town = "A Coruña" if key == "coruna" else name.split(" de ", 1)[1]
        facilities = []
        seen = {}
        for kind in SIZES[key]:
            seen[kind] = seen.get(kind, 0) + 1
            t = TEMPLATES[kind]
            n = seen[kind]
            alone = SIZES[key].count(kind) == 1 and kind != "piscina"
            label = t["name"].replace(" {n}", "") if alone else t["name"].format(n=n)
            place = PLACES[(n + len(kind)) % len(PLACES)]
            facilities.append({
                "key": f"{kind}-{n}",
                "slug": _slug(label),
                "name": label,
                "sport": t["sport"],
                "tags": t["tags"],
                "image_ref": t["image_ref"],
                "description": t["description"],
                "address": f"{place}, {town}",
                "latitude": round(lat + rng.uniform(-0.012, 0.012), 5),
                "longitude": round(lon + rng.uniform(-0.016, 0.016), 5),
                "capacity": t["capacity"],
                "slot_minutes": t["slot_minutes"],
                "usage_rules_text": t["usage_rules_text"],
                "accessibility_info": t.get("accessibility_info"),
                "show_weather": t.get("show_weather", False),
                "min_age": t.get("min_age"),
                "schedule": t["schedule"],
            })
        out.append({
            "key": key, "slug": key, "name": name, "contact_email": None, "contact_phone": None,
            "municipality_code": ine, "logo_ref": None, "facilities": facilities,
        })
    return out
