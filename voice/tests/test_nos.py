from lqvoice.nos import clauses


def test_long_sentences_in_clauses():
    short = "Un momento, que o miro."
    assert clauses(short) == [short]
    long = (
        "Na pista dous hai quendas das tres, das catro e media, das seis, das sete e media e das nove, de hora e media."
    )
    pieces = clauses(long, longest=50)
    assert len(pieces) > 1 and " ".join(pieces) == long
