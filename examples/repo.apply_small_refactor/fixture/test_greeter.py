from greeter import greeting


def test_greeting_strips_names():
    assert greeting(" Ada ", " Lovelace ") == "Hello, Ada Lovelace!"
