import os
from flask import request

def run_cmd():
    user_input = request.args.get("command")
    os.system(user_input)
