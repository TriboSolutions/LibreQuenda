# ============================================================
#  SEED DATA CONFIGURATION
#  Sample council for development and demos.
# ============================================================

TIMEZONE = "Europe/Madrid"

TENANT = {
    "key": "demo",
    "slug": "demo",
    "name": "Concello de Demostración",
    "contact_email": "deportes@demo.example",
    "contact_phone": "986 000 000",
    # files in db/init/seed_media, uploaded by the storage-seed service; the database keeps the reference
    "logo_ref": "councils/demo.png",
}

PARK = "Complexo deportivo do Parque, Rúa do Parque 2"

# ISO weekdays: 1 = Monday ... 7 = Sunday
WEEKDAYS = [1, 2, 3, 4, 5]
SATURDAY = [6]
WEEKEND = [6, 7]

FACILITIES = [
    {
        "key": "padel-1", "slug": "pista-de-padel-1", "image_ref": "facilities/pista-de-padel-1.jpg", "description": "Pista de pádel con cristal e iluminación LED, ao aire libre e cuberta en caso de choiva.", "latitude": 42.6612, "longitude": -8.1132, "name": "Pista de pádel 1", "sport": "Pádel", "address": PARK,
        "capacity": 4, "slot_minutes": 90, "max_advance_days": 7, "min_cancel_hours": 4,
        "max_active_bookings_per_user": 2, "usage_rules_text": "Calzado de pádel obrigatorio.",
        # nine 90-minute turns on weekdays, three at weekends
        "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
    },
    {
        "key": "padel-2", "slug": "pista-de-padel-2", "image_ref": "facilities/pista-de-padel-2.jpg", "description": "Segunda pista de pádel do complexo, xunto á pista 1. Vestiarios compartidos.", "latitude": 42.6613, "longitude": -8.1129, "name": "Pista de pádel 2", "sport": "Pádel", "address": PARK,
        "capacity": 4, "slot_minutes": 90, "max_advance_days": 7, "min_cancel_hours": 4,
        "max_active_bookings_per_user": 2, "usage_rules_text": "Calzado de pádel obrigatorio.",
        "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
    },
    {
        "key": "pavillon", "slug": "pavillon-municipal", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista polideportiva cuberta de 40 × 20 m para fútbol sala, baloncesto, balonmán e voleibol. Bancadas para 300 persoas.", "latitude": 42.6586, "longitude": -8.1101, "name": "Pavillón municipal", "sport": "Polideportivo", "address": "Avenida da Feira 14",
        "capacity": 30, "slot_minutes": 60, "max_advance_days": 14, "min_cancel_hours": 24,
        "max_active_bookings_per_user": 2, "usage_rules_text": "Só calzado de sala.",
        "schedule": [(WEEKDAYS, "09:00", "22:00"), (SATURDAY, "10:00", "14:00")],
    },
    {
        "key": "tenis", "slug": "pista-de-tenis", "image_ref": "facilities/pista-de-tenis.jpg", "description": "Pista de tenis de resina con iluminación. O concello presta raquetas no control de acceso.", "latitude": 42.6609, "longitude": -8.1136, "name": "Pista de tenis", "sport": "Tenis", "address": PARK,
        "capacity": 4, "slot_minutes": 60, "max_advance_days": 7, "min_cancel_hours": 2,
        "max_active_bookings_per_user": 2, "usage_rules_text": None,
        "schedule": [(WEEKDAYS, "09:00", "21:00"), (WEEKEND, "10:00", "14:00")],
    },
    {
        "key": "futbol-7", "slug": "campo-de-futbol-7", "image_ref": "facilities/campo-de-futbol-7.jpg", "description": "Campo de céspede artificial de fútbol 7 con iluminación e vestiarios.", "latitude": 42.6641, "longitude": -8.1178, "name": "Campo de fútbol 7", "sport": "Fútbol 7", "address": "Campo da Veiga, Camiño da Veiga s/n",
        "capacity": 14, "slot_minutes": 60, "max_advance_days": 14, "min_cancel_hours": 24,
        "max_active_bookings_per_user": 1, "usage_rules_text": "Céspede artificial: só bota de tacos de goma.",
        # evenings on weekdays, Saturday morning
        "schedule": [(WEEKDAYS, "17:00", "23:00"), (SATURDAY, "09:00", "14:00")],
    },
    {
        "key": "piscina-3", "slug": "piscina-rua-3", "image_ref": "facilities/piscina-rua-3.jpg", "description": "Rúa 3 da piscina climatizada de 25 m, reservada para nado libre. Auga a 28 °C.", "latitude": 42.6583, "longitude": -8.1095, "name": "Piscina municipal, rúa 3", "sport": "Natación", "address": "Piscina climatizada, Avenida da Feira 16",
        "capacity": 2, "slot_minutes": 45, "max_advance_days": 3, "min_cancel_hours": 2,
        "max_active_bookings_per_user": 2, "usage_rules_text": "Gorro obrigatorio.",
        # early morning and afternoon on weekdays
        "schedule": [(WEEKDAYS, "07:00", "10:00"), (WEEKDAYS, "16:00", "21:15"), (SATURDAY, "09:00", "13:30")],
    },
]

# Sample residents. Never shown: the public day view only exposes busy intervals.
RESIDENTS = [
    {"key": "uxia", "full_name": "Uxía Pereira", "email": "uxia.pereira@example.com"},
    {"key": "brais", "full_name": "Brais Otero", "email": "brais.otero@example.com"},
    {"key": "antia", "full_name": "Antía Vázquez", "email": "antia.vazquez@example.com"},
    {"key": "xoan", "full_name": "Xoán Castro", "email": "xoan.castro@example.com"},
]

# Relative to the day the database is created: (facility, day offset, start, end, reason)
CLOSURES = [
    ("tenis", 0, "12:00", "14:00", "Cambio da rede"),
]

# Relative to the day the database is created: (facility, resident, day offset, start)
# The length is the facility's slot; every start must be one of its turns.
BOOKINGS = [
    ("padel-1", "uxia", 0, "10:30"),
    ("padel-1", "brais", 0, "18:00"),
    ("padel-1", "antia", 0, "19:30"),
    ("padel-2", "xoan", 0, "19:30"),
    ("padel-2", "uxia", 0, "21:00"),
    ("pavillon", "brais", 0, "18:00"),
    ("pavillon", "antia", 0, "19:00"),
    ("pavillon", "xoan", 0, "20:00"),
    ("tenis", "uxia", 0, "17:00"),
    ("futbol-7", "brais", 0, "19:00"),
    ("futbol-7", "antia", 0, "20:00"),
    ("futbol-7", "xoan", 0, "21:00"),
    ("piscina-3", "uxia", 0, "18:15"),
    ("piscina-3", "brais", 0, "19:00"),
    ("padel-1", "xoan", 1, "19:30"),
    ("futbol-7", "uxia", 1, "20:00"),
]
