sudo rm -rf .venv

python3 -m venv .venv

.venv/bin/pip install semgrep
.venv/bin/pip install pip-audit