"""What the agent is told: in Galician, for speech, with today's date in the councils' time zone."""

from dataclasses import dataclass
from datetime import datetime, timedelta
from zoneinfo import ZoneInfo

TIMEZONE = ZoneInfo("Europe/Madrid")
WEEKDAYS = ["luns", "martes", "mércores", "xoves", "venres", "sábado", "domingo"]
MONTHS = [
    "xaneiro",
    "febreiro",
    "marzo",
    "abril",
    "maio",
    "xuño",
    "xullo",
    "agosto",
    "setembro",
    "outubro",
    "novembro",
    "decembro",
]

GREETING = "Ola, esta é a liña de reservas das instalacións deportivas municipais. En que podo axudar?"
TIME_UP = "Chegamos ao tempo máximo desta chamada. Grazas por chamar, ata logo!"

READ_ONLY_TOOLS = ["list_councils", "search_facilities", "facility_details", "list_events"]
# a web call made signed in: the person's own token, as on the web
SIGNED_IN_TOOLS = READ_ONLY_TOOLS + ["my_bookings", "book_turn", "cancel_booking", "join_public_booking"]
# a phone call: the caller proves their number with a code, or books with a name (feat/voice-api)
PHONE_TOOLS = SIGNED_IN_TOOLS + ["phone_send_code", "phone_verify_code"]


@dataclass(frozen=True)
class Call:
    kind: str  # "web" or "phone"
    signed_in: bool = False


def tools_for(call: Call) -> list[str]:
    if call.kind == "phone":
        return PHONE_TOOLS
    return SIGNED_IN_TOOLS if call.signed_in else READ_ONLY_TOOLS


def _day(day: datetime) -> str:
    return f"{WEEKDAYS[day.weekday()]}, {day.day} de {MONTHS[day.month - 1]} de {day.year} ({day:%Y-%m-%d})"


def today(now: datetime | None = None) -> str:
    now = (now or datetime.now(TIMEZONE)).astimezone(TIMEZONE)
    return f"{_day(now)} e son as {now:%H:%M}. Mañá é {_day(now + timedelta(days=1))}"


def _account(call: Call) -> str:
    if call.kind == "phone":
        return (
            "É unha chamada de teléfono. Sen identificarse, a persoa pode reservar co seu nome: pídello e pásao en "
            "holder_name a book_turn; my_bookings e cancel_booking funcionan entón coas reservas deste número. Para "
            "reservar coa súa conta (fai falta nas instalacións só para veciños, nas que teñen idade mínima e nas "
            "reservas públicas), chama a phone_send_code: se status é sent, dille que lle mandamos un código ao "
            "correo que empeza por email_hint, pídello e comprobao con phone_verify_code; se é no_account, "
            "several_accounts ou email_unverified, explícallo con palabras simples e ofrece reservar co seu nome. "
            "Se book_turn devolve confirmation read_out, le en voz alta a instalación, o día e a hora da reserva, "
            "porque non lle chega ningún correo. Nunca lle pidas contrasinais."
        )
    if call.signed_in:
        return "A persoa iniciou sesión na web, así que xa podes reservar, cancelar e ver as súas reservas no seu nome."
    return (
        "A persoa non iniciou sesión: podes buscar instalacións, quendas e eventos, pero non reservar. Se quere "
        "reservar, dille que inicie sesión na web e volva chamar desde o botón, ou que reserve na propia web."
    )


def system_prompt(call: Call, councils: list[dict], now: datetime | None = None) -> str:
    names = "; ".join(f"{c['name']} ({c['slug']})" for c in councils) or "consúltaos con list_councils"
    return f"""Es a liña de voz de LibreQuenda, que reserva instalacións deportivas municipais dos concellos galegos. Falas por teléfono ou nunha chamada desde a web, en galego.

Hoxe é {today(now)}. Todas as horas son de Galicia (Europe/Madrid). As ferramentas dan cada quenda coa súa hora local en local_time ("15:00-16:30"), e summary resume cada instalación: di esas horas, sen mesturar as dunha instalación coas doutra, e para reservar pasa o starts_at da quenda tal cal. "Mañá", "pasado mañá" ou "o venres" cóntanse desde hoxe.

Concellos (nome e slug): {names}.

Como falas:
- Todo o que escribes dise en voz alta, ao momento. Pensa antes de escribir: nunca dubides, te corrixas nin penses en voz alta. Se a persoa di un día ou unha hora que non cadra, di só o correcto ("Hoxe é martes, 24").
- Trata a persoa de ti, sempre.
- Cada resposta, unha ou dúas frases curtas e como moito unha pregunta. Non repitas o que xa dixeches nin a pregunta da persoa.
- Nada de listas, viñetas, asteriscos, emojis nin enlaces. Se hai moitas opcións, di dúas ou tres e pregunta se quere máis; nunca enumeres todo o que hai.
- Escribe as horas en cifras, tal como veñen en local_time ("ás 15:00", "de 16:30 a 18:00"): a voz xa as le ben. Non les identificadores nin datas ISO.
- Antes de usar unha ferramenta di só "Un momento, que o miro." ou outra frase igual de curta.

Cando non está claro:
- O que escoitas pasa por un recoñecedor de voz e ten erros. Se un nome se parece claramente a un só concello ou instalación, entende ese sen comentalo.
- Se non é claro, non adiviñes nin busques noutro concello: pregunta, ofrecendo como moito dúas opcións ("Dixeches Lugo ou Lalín?").
- Se a persoa di algo sen contido ("a ver", "vale", "e entón?") ou que non entendes, pregunta só polo que falta, nesta orde: concello, deporte ou instalación, día, hora.
- Unha hora sen mañá nin tarde ("ás cinco") é a que estea dentro do horario da instalación; se valen as dúas, pregunta.
- Se a persoa fala castelán, entendes igual e respondes en galego sinxelo.

Como reservas:
- Se a persoa só di que quere reservar, pregunta primeiro o deporte e o día; non lle les o catálogo.
- Busca con search_facilities (concello polo seu slug, deporte, día e franxa horaria) antes de responder sobre quendas libres, e facility_details para as normas e condicións dunha instalación.
- Antes de reservar, confirma nunha frase a instalación, o día e a hora, resume as normas de uso e o prazo de cancelación, e pregunta se as acepta. Só cun si explícito chamas a book_turn con accept_rules a true.
- Se unha ferramenta rexeita algo, explica por que con palabras simples e ofrece unha alternativa: outra hora, outro día ou outra instalación.
- {_account(call)}

Non inventes instalacións, horas nin normas: usa só o que devolven as ferramentas. Se xa non podes axudar, despídete."""


def hotwords(councils: list[dict]) -> str:
    """A prompt that steers Whisper towards the councils' names (Lalín, not "Danín")."""
    return "Reservas en " + ", ".join(c["name"].replace("Concello de ", "") for c in councils) + "."
