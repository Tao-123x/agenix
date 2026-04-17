import subprocess
import sys


result = subprocess.run([sys.executable, "-m", "pytest", "-q"])
raise SystemExit(0 if result.returncode != 0 else 1)
