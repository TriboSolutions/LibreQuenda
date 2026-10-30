# ============================================================
#  SEED DATA CONFIGURATION
#  Sample councils for development and demos.
# ============================================================

TIMEZONE = "Europe/Madrid"

# ISO weekdays: 1 = Monday ... 7 = Sunday
WEEKDAYS = [1, 2, 3, 4, 5]
SATURDAY = [6]
WEEKEND = [6, 7]

# Three real Galician councils. Names, addresses and coordinates of the facilities
# come from OpenStreetMap (© OpenStreetMap contributors, ODbL); the phone numbers
# and email are the councils' public ones. Opening hours, capacities, rules and
# bookings are made up for the demo. Coats of arms: see db/init/seed_media/CREDITS.md.
# Facility keys are "<council>/<facility>" everywhere below.
COUNCILS = [
    {
        "key": "lalin",
        "slug": "lalin",
        "name": "Concello de Lalín",
        "contact_email": "lalin@lalin.org",
        "contact_phone": "986 787 060",
        # files in db/init/seed_media, uploaded by the storage-seed service; the database keeps the reference
        "logo_ref": "councils/lalin.png",
        "facilities": [
            {
                "key": "padel-1", "tags": ["padel", "pista", "cuberta", "iluminacion", "vestiarios"], "slug": "pista-de-padel-1", "image_ref": "facilities/pista-de-padel-1.jpg", "description": "Pista de pádel cuberta e con iluminación no complexo Lalín Arena.", "latitude": 42.65921, "longitude": -8.11845, "name": "Pista de pádel 1 do Lalín Arena", "sport": "Pádel", "address": "Lalín Arena, Estrada Ourense - Santiago, 36500 Lalín",
                "capacity": 4, "slot_minutes": 90,
                "usage_rules_text": "Calzado de pádel obrigatorio.",
                # nine 90-minute turns on weekdays, three at weekends
                "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
            },
            {
                "key": "padel-2", "tags": ["padel", "pista", "cuberta", "iluminacion", "vestiarios"], "slug": "pista-de-padel-2", "image_ref": "facilities/pista-de-padel-2.jpg", "description": "Segunda pista de pádel cuberta do Lalín Arena, xunto á pista 1.", "latitude": 42.65924, "longitude": -8.1183, "name": "Pista de pádel 2 do Lalín Arena", "sport": "Pádel", "address": "Lalín Arena, Estrada Ourense - Santiago, 36500 Lalín",
                "capacity": 4, "slot_minutes": 90,
                "usage_rules_text": "Calzado de pádel obrigatorio.",
                "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
            },
            {
                "key": "pavillon", "tags": ["futbol-sala", "baloncesto", "balonman", "voleibol", "multiusos", "pavillon", "cuberta", "vestiarios", "accesible"], "slug": "pavillon-municipal-de-deportes", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista polideportiva cuberta para fútbol sala, baloncesto, balonmán e voleibol.", "latitude": 42.66453, "longitude": -8.10815, "name": "Pavillón Municipal de Deportes", "sport": "Polideportivo", "address": "Rúa do Polideportivo 2, 36500 Lalín",
                "capacity": 30, "slot_minutes": 60,
                "usage_rules_text": "Só calzado de sala.",
                "schedule": [(WEEKDAYS, "09:00", "22:00"), (SATURDAY, "10:00", "14:00")],
            },
            {
                "key": "tenis-1", "tags": ["tenis", "pista", "aire-libre"], "slug": "pista-de-tenis-1", "image_ref": "facilities/pista-de-tenis.jpg", "description": "Pista de tenis ao aire libre a carón do pavillón municipal.", "latitude": 42.66419, "longitude": -8.1075, "name": "Pista de tenis 1", "sport": "Tenis", "address": "Rúa da Banda de Música de Lalín, 36500 Lalín",
                "capacity": 4, "slot_minutes": 60,
                "usage_rules_text": None,
                "schedule": [(WEEKDAYS, "09:00", "21:00"), (WEEKEND, "10:00", "14:00")],
            },
            {
                "key": "tenis-2", "tags": ["tenis", "pista", "aire-libre"], "slug": "pista-de-tenis-2", "image_ref": "facilities/pista-de-tenis.jpg", "description": "Segunda pista de tenis ao aire libre a carón do pavillón municipal.", "latitude": 42.66441, "longitude": -8.10758, "name": "Pista de tenis 2", "sport": "Tenis", "address": "Rúa da Banda de Música de Lalín, 36500 Lalín",
                "capacity": 4, "slot_minutes": 60,
                "usage_rules_text": None,
                "schedule": [(WEEKDAYS, "09:00", "21:00"), (WEEKEND, "10:00", "14:00")],
            },
            {
                "key": "piscina-3", "tags": ["natacion", "piscina", "climatizada", "vestiarios", "accesible"], "slug": "piscina-rua-3", "image_ref": "facilities/piscina-rua-3.jpg", "description": "Rúa 3 das piscinas municipais, reservada para nado libre.", "latitude": 42.66458, "longitude": -8.10667, "name": "Piscinas municipais, rúa 3", "sport": "Natación", "address": "Paseo Pontiñas, 36500 Lalín",
                "capacity": 2, "slot_minutes": 45,
                "usage_rules_text": "Gorro obrigatorio.",
                # early morning and afternoon on weekdays
                "schedule": [(WEEKDAYS, "07:00", "10:00"), (WEEKDAYS, "16:00", "21:15"), (SATURDAY, "09:00", "13:30")],
            },
        ],
    },
    {
        "key": "lugo",
        "slug": "lugo",
        "name": "Concello de Lugo",
        "contact_email": None,
        "contact_phone": "982 297 100",
        "logo_ref": "councils/lugo.png",
        "facilities": [
            {
                "key": "pedreiras-2", "tags": ["natacion", "piscina", "cuberta", "climatizada", "vestiarios", "accesible"], "slug": "piscina-as-pedreiras-rua-2", "image_ref": "facilities/piscina-rua-3.jpg", "description": "Rúa 2 da piscina municipal cuberta As Pedreiras, reservada para nado libre.", "latitude": 42.99154, "longitude": -7.54363, "name": "Piscina municipal As Pedreiras, rúa 2", "sport": "Natación", "address": "Avenida da Filarmónica Lucense s/n, 27294 Lugo",
                "capacity": 2, "slot_minutes": 45,
                "usage_rules_text": "Gorro obrigatorio. Ducha antes de entrar na auga.",
                "schedule": [(WEEKDAYS, "07:30", "10:30"), (WEEKDAYS, "15:00", "21:45"), (WEEKEND, "10:00", "13:45")],
            },
            {
                "key": "frigsa-4", "tags": ["natacion", "piscina", "climatizada", "vestiarios"], "slug": "piscina-de-frigsa-rua-4", "image_ref": "facilities/piscina-rua-3.jpg", "description": "Rúa 4 da piscina municipal de Frigsa, reservada para nado libre.", "latitude": 43.02648, "longitude": -7.56188, "name": "Piscina municipal de Frigsa, rúa 4", "sport": "Natación", "address": "Rúa Monte Faro, 27003 Lugo",
                "capacity": 2, "slot_minutes": 45,
                "usage_rules_text": "Gorro obrigatorio.",
                "schedule": [(WEEKDAYS, "08:00", "11:00"), (WEEKDAYS, "16:00", "21:15")],
            },
            {
                "key": "frigsa-pavillon", "tags": ["futbol-sala", "baloncesto", "balonman", "multiusos", "pavillon", "cuberta", "vestiarios"], "slug": "pavillon-de-frigsa", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista polideportiva cuberta do Complexo Polideportivo de Frigsa.", "latitude": 43.02638, "longitude": -7.56127, "name": "Pavillón do Complexo Polideportivo de Frigsa", "sport": "Polideportivo", "address": "Avenida Infanta Elena, 27003 Lugo",
                "capacity": 30, "slot_minutes": 60,
                "usage_rules_text": "Só calzado de sala.",
                "schedule": [(WEEKDAYS, "09:00", "23:00"), (WEEKEND, "10:00", "14:00")],
            },
            {
                "key": "palacio", "tags": ["futbol-sala", "baloncesto", "balonman", "voleibol", "multiusos", "pavillon", "cuberta", "vestiarios", "accesible"], "slug": "palacio-municipal-de-deportes", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista central do Palacio Municipal de Deportes, para fútbol sala e outros deportes de pista.", "latitude": 43.00509, "longitude": -7.56229, "name": "Palacio Municipal de Deportes", "sport": "Fútbol sala", "address": "Ronda da República Arxentina, 27001 Lugo",
                "capacity": 12, "slot_minutes": 60,
                "usage_rules_text": "Só calzado de sala. Os clubs teñen preferencia nas fins de semana de competición.",
                "schedule": [(WEEKDAYS, "16:00", "23:00")],
            },
            {
                "key": "gorgoso", "tags": ["futbol", "futbol-7", "campo", "aire-libre", "cesped-artificial", "iluminacion", "vestiarios"], "slug": "campo-luis-lopez-gorgoso", "image_ref": "facilities/campo-de-futbol-7.jpg", "description": "Campo de fútbol de céspede artificial con iluminación e vestiarios.", "latitude": 43.02475, "longitude": -7.56094, "name": "Campo de fútbol Luis López Gorgoso", "sport": "Fútbol", "address": "Avenida Infanta Elena, 27003 Lugo",
                "capacity": 22, "slot_minutes": 90,
                "usage_rules_text": "Céspede artificial: só bota de tacos de goma.",
                "schedule": [(WEEKDAYS, "17:00", "23:00"), (SATURDAY, "09:00", "13:30")],
            },
        ],
    },
    {
        "key": "ourense",
        "slug": "ourense",
        "name": "Concello de Ourense",
        "contact_email": None,
        # Consello Municipal de Deportes, Avenida de Pardo de Cela 2
        "contact_phone": "988 370 539",
        "logo_ref": "councils/ourense.png",
        "facilities": [
            {
                "key": "remedios-pavillon", "tags": ["futbol-sala", "baloncesto", "voleibol", "multiusos", "pavillon", "cuberta", "vestiarios"], "slug": "pavillon-dos-remedios", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista polideportiva cuberta do complexo dos Remedios, a carón da Ponte Romana.", "latitude": 42.34393, "longitude": -7.87031, "name": "Pavillón dos Remedios", "sport": "Polideportivo", "address": "Avenida de Pardo de Cela 2, 32003 Ourense",
                "capacity": 30, "slot_minutes": 60,
                "usage_rules_text": "Só calzado de sala.",
                "schedule": [(WEEKDAYS, "08:00", "23:00"), (WEEKEND, "09:00", "14:00")],
            },
            {
                "key": "remedios-piscina-5", "tags": ["natacion", "piscina", "vestiarios"], "slug": "piscina-dos-remedios-rua-5", "image_ref": "facilities/piscina-rua-3.jpg", "description": "Rúa 5 das piscinas dos Remedios, reservada para nado libre.", "latitude": 42.34361, "longitude": -7.87117, "name": "Piscinas dos Remedios, rúa 5", "sport": "Natación", "address": "Avenida de Pardo de Cela 2, 32003 Ourense",
                "capacity": 2, "slot_minutes": 45,
                "usage_rules_text": "Gorro obrigatorio.",
                "schedule": [(WEEKDAYS, "07:00", "10:00"), (WEEKDAYS, "15:30", "21:30"), (SATURDAY, "10:00", "13:00")],
            },
            {
                "key": "remedios-campo", "tags": ["futbol", "campo", "aire-libre", "iluminacion", "vestiarios"], "slug": "campo-miguel-angel-os-remedios", "image_ref": "facilities/campo-de-futbol-7.jpg", "description": "Campo de fútbol do complexo dos Remedios, con iluminación e vestiarios.", "latitude": 42.34423, "longitude": -7.87139, "name": "Campo de fútbol Miguel Ángel - Os Remedios", "sport": "Fútbol", "address": "Avenida da Ribeira Sacra, 32003 Ourense",
                "capacity": 22, "slot_minutes": 90,
                "usage_rules_text": "Só bota de tacos de goma.",
                "schedule": [(WEEKDAYS, "17:30", "23:30"), (SATURDAY, "09:00", "13:30")],
            },
            {
                "key": "oira-pavillon", "tags": ["futbol-sala", "baloncesto", "balonman", "multiusos", "pavillon", "cuberta"], "slug": "pavillon-de-oira", "image_ref": "facilities/pavillon-municipal.jpg", "description": "Pista cuberta do Complexo Deportivo de Oira, á beira do Miño.", "latitude": 42.35612, "longitude": -7.85269, "name": "Pavillón de Oira", "sport": "Polideportivo", "address": "Paseo de Oira, 32004 Ourense",
                "capacity": 24, "slot_minutes": 60,
                "usage_rules_text": "Só calzado de sala.",
                "schedule": [(WEEKDAYS, "10:00", "22:00"), (SATURDAY, "10:00", "14:00")],
            },
            {
                "key": "oira-padel", "tags": ["padel", "pista", "aire-libre"], "slug": "pista-de-padel-de-oira", "image_ref": "facilities/pista-de-padel-1.jpg", "description": "Pista de pádel ao aire libre do Complexo Deportivo de Oira.", "latitude": 42.35499, "longitude": -7.85376, "name": "Pista de pádel de Oira", "sport": "Pádel", "address": "Complexo Deportivo de Oira, Paseo de Oira, 32004 Ourense",
                "capacity": 4, "slot_minutes": 90,
                "usage_rules_text": "Calzado de pádel obrigatorio.",
                "schedule": [(WEEKDAYS, "09:00", "22:30"), (WEEKEND, "10:00", "14:30")],
            },
        ],
    },
]

# Sample residents. Never shown: the public day view only exposes busy intervals.
RESIDENTS = [
    {"key": "uxia", "full_name": "Uxía Pereira", "email": "uxia.pereira@example.com"},
    {"key": "brais", "full_name": "Brais Otero", "email": "brais.otero@example.com"},
    {"key": "antia", "full_name": "Antía Vázquez", "email": "antia.vazquez@example.com"},
    {"key": "xoan", "full_name": "Xoán Castro", "email": "xoan.castro@example.com"},
    # residents who only appear in the booking history (see history.sql)
    {"key": "iago", "full_name": "Iago Fernández", "email": "iago.fernandez@example.com"},
    {"key": "noa", "full_name": "Noa Rodríguez", "email": "noa.rodriguez@example.com"},
    {"key": "martin", "full_name": "Martín López", "email": "martin.lopez@example.com"},
    {"key": "lucia", "full_name": "Lucía Gómez", "email": "lucia.gomez@example.com"},
    {"key": "breixo", "full_name": "Breixo Souto", "email": "breixo.souto@example.com"},
    {"key": "sabela", "full_name": "Sabela Rial", "email": "sabela.rial@example.com"},
    {"key": "roi", "full_name": "Roi Barreiro", "email": "roi.barreiro@example.com"},
    {"key": "carme", "full_name": "Carme Dopico", "email": "carme.dopico@example.com"},
    {"key": "anxo", "full_name": "Anxo Varela", "email": "anxo.varela@example.com"},
    {"key": "aldara", "full_name": "Aldara Pena", "email": "aldara.pena@example.com"},
    {"key": "pablo", "full_name": "Pablo Iglesias Sanmartín", "email": "pablo.iglesias@example.com"},
    {"key": "maria", "full_name": "María Fraga", "email": "maria.fraga@example.com"},
    {"key": "hugo", "full_name": "Hugo Seoane", "email": "hugo.seoane@example.com"},
    {"key": "olaia", "full_name": "Olaia Carballo", "email": "olaia.carballo@example.com"},
    {"key": "daniel", "full_name": "Daniel Rey", "email": "daniel.rey@example.com"},
    {"key": "iria", "full_name": "Iria Mosquera", "email": "iria.mosquera@example.com"},
]

# Relative to the day the database is created: (facility, day offset, start, end, reason)
CLOSURES = [
    ("lalin/tenis-1", 0, "12:00", "14:00", "Cambio da rede"),
    ("ourense/oira-padel", 2, "09:00", "13:30", "Limpeza dos cristais"),
    # past: the bookings they caught show as cancelled by the council in the indicators
    ("lalin/pavillon", -23, "09:00", "22:00", "Mantemento do parqué"),
    ("lugo/gorgoso", -9, "17:00", "23:00", "Campo asolagado polo temporal"),
    ("ourense/remedios-piscina-5", -41, "07:00", "21:30", "Revisión da depuradora"),
]

# QR readers at the entrance: (facility, name). Check-ins in the history only
# happen where there is one. Seeded without a usable key: create a new reader
# from the admin page to try the check-in.
READERS = [
    ("lalin/padel-1", "Entrada do Lalín Arena"),
    ("lalin/pavillon", "Entrada principal"),
    ("lugo/pedreiras-2", "Control de acceso"),
    ("lugo/gorgoso", "Porta do campo"),
    ("ourense/remedios-pavillon", "Entrada dos Remedios"),
    ("ourense/oira-pavillon", "Porta de Oira"),
]

# Days of booking history before the day the database is created (see history.sql)
HISTORY_DAYS = 400

# Relative to the day the database is created: (facility, resident, day offset, start)
# The length is the facility's slot; every start must be one of its turns.
BOOKINGS = [
    ("lalin/padel-1", "uxia", 0, "10:30"),
    ("lalin/padel-1", "brais", 0, "18:00"),
    ("lalin/padel-1", "antia", 0, "19:30"),
    ("lalin/padel-2", "xoan", 0, "19:30"),
    ("lalin/padel-2", "uxia", 0, "21:00"),
    ("lalin/pavillon", "brais", 0, "18:00"),
    ("lalin/pavillon", "antia", 0, "19:00"),
    ("lalin/tenis-2", "uxia", 0, "17:00"),
    ("lalin/piscina-3", "uxia", 0, "18:15"),
    ("lalin/piscina-3", "brais", 0, "19:00"),
    ("lalin/padel-1", "xoan", 1, "19:30"),
    ("lugo/pedreiras-2", "antia", 0, "18:45"),
    ("lugo/pedreiras-2", "xoan", 0, "19:30"),
    ("lugo/frigsa-4", "brais", 0, "17:30"),
    ("lugo/frigsa-pavillon", "xoan", 0, "20:00"),
    ("lugo/palacio", "brais", 0, "21:00"),
    ("lugo/gorgoso", "antia", 0, "20:00"),
    ("lugo/gorgoso", "uxia", 1, "18:30"),
    ("ourense/remedios-pavillon", "xoan", 0, "18:00"),
    ("ourense/remedios-pavillon", "uxia", 0, "19:00"),
    ("ourense/remedios-piscina-5", "antia", 0, "17:45"),
    ("ourense/remedios-campo", "brais", 0, "20:30"),
    ("ourense/oira-pavillon", "antia", 0, "19:00"),
    ("ourense/oira-padel", "xoan", 0, "18:00"),
    ("ourense/oira-padel", "brais", 1, "19:30"),
]
