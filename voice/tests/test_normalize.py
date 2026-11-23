import pytest

from lqvoice.normalize import normalize, number, time_of_day


@pytest.mark.parametrize(
    ("n", "feminine", "said"),
    [
        (0, False, "cero"),
        (1, True, "unha"),
        (2, False, "dous"),
        (2, True, "dúas"),
        (21, False, "vinte e un"),
        (22, True, "vinte e dúas"),
        (100, False, "cen"),
        (101, False, "cento un"),
        (200, True, "dúascentas"),
        (555, False, "cincocentos cincuenta e cinco"),
        (1000, False, "mil"),
        (2026, False, "dous mil vinte e seis"),
        (1500, False, "mil cincocentos"),
    ],
)
def test_number(n, feminine, said):
    assert number(n, feminine) == said


@pytest.mark.parametrize(
    ("hour", "minute", "said"),
    [
        (18, 30, "seis e media da tarde"),
        (9, 0, "nove da mañá"),
        (13, 15, "unha e cuarto da tarde"),
        (13, 45, "dúas menos cuarto da tarde"),
        (21, 0, "nove da noite"),
        (12, 0, "doce do mediodía"),
        (0, 0, "doce da noite"),
        (10, 10, "dez e dez da mañá"),
        (16, 50, "cinco menos dez da tarde"),
    ],
)
def test_time_of_day(hour, minute, said):
    assert time_of_day(hour, minute) == said


@pytest.mark.parametrize(
    ("text", "said"),
    [
        ("Hai quenda ás 18:30.", "Hai quenda ás seis e media da tarde."),
        ("Libre ás 13:00", "Libre á unha da tarde"),
        ("Abre ás 9 h", "Abre ás nove da mañá"),
        ("de 16.30h a 18.00h", "de catro e media a seis da tarde"),
        (
            "das 15:00 ás 16:30 e de 19:30 a 21:00",
            "das tres ás catro e media da tarde e de sete e media da tarde a nove da noite",
        ),
        ("ás 15:00 (15:00-16:30)", "ás tres da tarde (tres a catro e media da tarde)"),
        ("O 24/11 ou o 2026-11-25", "O vinte e catro de novembro ou o vinte e cinco de novembro"),
        ("Chama ao 986 787 060.", "Chama ao nove oito seis, sete oito sete, cero seis cero."),
        ("Quedan 2 pistas e 3 prazas", "Quedan dúas pistas e tres prazas"),
        ("Son 2 euros", "Son dous euros"),
        ("Avda. da Coruña s/n", "avenida da Coruña sen número"),
        ("**Pista 1**", "Pista un"),
        ("a 3,5 km", "a tres coma cinco quilómetros"),
        ("Á 1ª hora", "Á primeira hora"),
    ],
)
def test_normalize(text, said):
    assert normalize(text) == said
