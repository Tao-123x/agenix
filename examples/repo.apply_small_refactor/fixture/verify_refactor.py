from pathlib import Path


content = Path("greeter.py").read_text()
if "def full_name(first, last):" not in content:
    raise SystemExit("missing full_name helper")
if 'return "Hello, " + full_name(first, last) + "!"' not in content:
    raise SystemExit("greeting does not delegate to full_name")
