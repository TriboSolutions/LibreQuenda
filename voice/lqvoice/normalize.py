"""Galician text normalisation for speech: what the agent writes, as it has to be said.

The Nós voices read graphemes, so "18:30", "24/11" or "986 787 060" must reach them
as words. Only the voice gets this text: captions and the conversation keep the
agent's own wording.
"""

import re

UNITS = [
    "cero",
    "un",
    "dous",
    "tres",
    "catro",
    "cinco",
    "seis",
    "sete",
    "oito",
    "nove",
    "dez",
    "once",
    "doce",
    "trece",
    "catorce",
    "quince",
    "dezaseis",
    "dezasete",
    "dezaoito",
    "dezanove",
]
TENS = {
    20: "vinte",
    30: "trinta",
    40: "corenta",
    50: "cincuenta",
    60: "sesenta",
    70: "setenta",
    80: "oitenta",
    90: "noventa",
}
HUNDREDS = {
    1: "cento",
    2: "douscentos",
    3: "trescentos",
    4: "catrocentos",
    5: "cincocentos",
    6: "seiscentos",
    7: "setecentos",
    8: "oitocentos",
    9: "novecentos",
}
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
ORDINALS = {
    1: "primeiro",
    2: "segundo",
    3: "terceiro",
    4: "cuarto",
    5: "quinto",
    6: "sexto",
    7: "sétimo",
    8: "oitavo",
    9: "noveno",
    10: "décimo",
}
# nouns the agent counts that are feminine: "dúas pistas", "unha persoa"
FEMININE = {
    "pista",
    "pistas",
    "persoa",
    "persoas",
    "hora",
    "horas",
    "praza",
    "prazas",
    "quenda",
    "quendas",
    "rúa",
    "rúas",
    "semana",
    "semanas",
    "reserva",
    "reservas",
    "instalación",
    "instalacións",
    "partida",
    "partidas",
    "piscina",
    "piscinas",
    "estrela",
    "estrelas",
    "vez",
    "veces",
}
ABBREVIATIONS = [
    (r"\bAvda\.", "avenida"),
    (r"\bAv\.", "avenida"),
    (r"\bPza\.", "praza"),
    (r"\bs/n\b", "sen número"),
    (r"\bn\.?º", "número"),
    (r"\bkm\b", "quilómetros"),
    (r"\bmin\b\.?", "minutos"),
    (r"\bp\. ?ex\.", "por exemplo"),
    (r"\bc/", "rúa "),
    (r"\bDNI\b", "de ene i"),
]


def _feminine(word: str) -> str:
    return {"un": "unha", "dous": "dúas"}.get(word, word)


def number(n: int, feminine: bool = False) -> str:
    """A whole number in Galician words, 0 to 999 999."""
    if n < 0:
        return "menos " + number(-n, feminine)
    if n < 20:
        word = UNITS[n]
        return _feminine(word) if feminine else word
    if n < 100:
        tens, unit = divmod(n, 10)
        return TENS[tens * 10] + ("" if unit == 0 else " e " + number(unit, feminine))
    if n < 1000:
        hundreds, rest = divmod(n, 100)
        if n == 100:
            return "cen"
        head = HUNDREDS[hundreds]
        if feminine and hundreds > 1:
            head = head.replace("centos", "centas").replace("dous", "dúas")
        return head + ("" if rest == 0 else " " + number(rest, feminine))
    if n < 1_000_000:
        thousands, rest = divmod(n, 1000)
        head = "mil" if thousands == 1 else number(thousands, feminine) + " mil"
        if rest == 0:
            return head
        return head + " " + number(rest, feminine)
    return " ".join(UNITS[int(d)] for d in str(n))


def _hour12(hour: int) -> tuple[str, str]:
    """The hour as said (feminine: "unha", "dúas") and its part of the day."""
    if hour == 0 or hour == 24:
        return "doce", "da noite"
    if hour < 6:
        return number(hour, True), "da madrugada"
    if hour < 12:
        return number(hour, True), "da mañá"
    if hour == 12:
        return "doce", "do mediodía"
    spoken = number(hour - 12, True)
    return spoken, "da tarde" if hour < 21 else "da noite"


def time_of_day(hour: int, minute: int) -> str:
    """18:30 -> "seis e media da tarde"; 13:45 -> "dúas menos cuarto da tarde"."""
    if minute > 30 and minute % 5 == 0:
        spoken, part = _hour12((hour + 1) % 24)
        rest = 60 - minute
        tail = "menos cuarto" if rest == 15 else "menos " + number(rest)
        return f"{spoken} {tail} {part}"
    spoken, part = _hour12(hour)
    if minute == 0:
        return f"{spoken} {part}"
    tail = {15: "e cuarto", 30: "e media"}.get(minute, "e " + number(minute))
    return f"{spoken} {tail} {part}"


def _time(match: re.Match) -> str:
    article, hour, minute = match.group("article"), int(match.group("h")), int(match.group("m") or 0)
    if hour > 24 or minute > 59:
        return match.group(0)
    said = time_of_day(hour, minute)
    if article:
        # "á unha", "ás dúas": the article agrees with the hour as said
        singular = said.startswith("unha ")
        lower = article.lower()
        if lower in ("ás", "á"):
            article = "á" if singular else "ás"
        elif lower in ("as", "a"):
            article = "a" if singular else "as"
        elif lower in ("das", "da"):
            article = "da" if singular else "das"
        return f"{article} {said}"
    return said


TIME = re.compile(
    r"(?:\b(?P<article>[áÁ]s?|[aA]s|[dD]as|[dD]a)\s+)?\b(?P<h>[01]?\d|2[0-4])"
    r"(?:[:.](?P<m>[0-5]\d)(?:\s*h\b\.?)?|\s*h\b\.?)"
)


# "de tres da tarde a catro da tarde" -> "de tres a catro da tarde"
SAME_PART = re.compile(
    r"(da madrugada|da mañá|do mediodía|da tarde|da noite) ((?:a|á|ás|ata|ata as|ata a) (?:[a-záéíóúñ]+ ){1,4}?)\1"
)


def _date(match: re.Match) -> str:
    groups = match.groupdict()
    # the year is left out: bookings are days or weeks away
    if groups.get("iy"):
        month, day = int(groups["im"]), int(groups["id"])
    else:
        day, month = int(groups["d"]), int(groups["mo"])
    if not (1 <= month <= 12 and 1 <= day <= 31):
        return match.group(0)
    return f"{number(day)} de {MONTHS[month - 1]}"


DATE = re.compile(
    r"\b(?P<iy>\d{4})-(?P<im>\d{2})-(?P<id>\d{2})\b"
    r"|\b(?P<d>[0-3]?\d)/(?P<mo>[01]?\d)(?:/(?P<y>\d{4}|\d{2}))?\b"
)
PHONE = re.compile(r"(?<!\d)(?<!\d[.,])(?:\+\d{2,3}[ .]?)?\d{3}(?:[ .]?\d{2,3}){2,3}(?!\d)(?![.,]\d)")
ORDINAL = re.compile(r"\b(\d{1,2})\.?[ºª]")
PERCENT = re.compile(r"(\d+(?:[.,]\d+)?)\s*%")
EUROS = re.compile(r"(\d+(?:[.,]\d+)?)\s*€")
DECIMAL = re.compile(r"\b(\d+),(\d+)\b")
NUMBER = re.compile(r"\b\d+\b(?:\s+(\w+))?")


def _phone(match: re.Match) -> str:
    groups = re.split(r"[ .]", match.group(0).replace("+", "máis "))
    return ", ".join(" ".join(UNITS[int(d)] if d.isdigit() else d for d in g) if g != "máis" else g for g in groups)


def _ordinal(match: re.Match) -> str:
    n = int(match.group(1))
    word = ORDINALS.get(n, number(n))
    return word[:-1] + "a" if match.group(0).endswith("ª") and word.endswith("o") else word


def _number(match: re.Match) -> str:
    digits = re.match(r"\d+", match.group(0)).group(0)
    following = match.group(1)
    said = number(int(digits), feminine=bool(following and following.lower() in FEMININE))
    return said + match.group(0)[len(digits) :]


def normalize(text: str) -> str:
    """Galician text as the voice should say it."""
    text = re.sub(r"[*_#`>|]+", "", text)  # markdown the model may still write
    for pattern, said in ABBREVIATIONS:
        text = re.sub(pattern, said, text)
    text = DATE.sub(_date, text)
    text = TIME.sub(_time, text)
    text = SAME_PART.sub(r"\2\1", text)
    text = PHONE.sub(_phone, text)
    text = ORDINAL.sub(_ordinal, text)
    text = PERCENT.sub(lambda m: f"{m.group(1)} por cento", text)
    text = EUROS.sub(lambda m: f"{m.group(1)} euros", text)
    text = DECIMAL.sub(lambda m: f"{number(int(m.group(1)))} coma {number(int(m.group(2)))}", text)
    text = NUMBER.sub(_number, text)
    text = text.replace("&", " e ").replace("@", " arroba ")
    return re.sub(r"\s{2,}", " ", text).strip()


async def tts_transform(text: str, _text_type) -> str:
    """Pipecat text transform: the voice gets the normalised text, captions keep the original."""
    return normalize(text)
