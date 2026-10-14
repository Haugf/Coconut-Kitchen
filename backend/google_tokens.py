"""Print the local tokens of the Google speakers on an account, as JSON.

The mirror server runs this with ~/mirror/venv/bin/python (set up by
`mirror google`) when it needs a token to read the speaker's timers. It
reads the sign-in saved in google.json and prints only
[{"name": ..., "token": ...}] on stdout, which the server keeps in memory.
"""

import json
import sys

from glocaltokens.client import GLocalAuthenticationTokens

with open(sys.argv[1]) as f:
    auth = json.load(f)

client = GLocalAuthenticationTokens(
    username=auth["email"],
    master_token=auth["masterToken"],
    android_id=auth["androidId"],
)
devices = client.get_google_devices(disable_discovery=True, force_homegraph_reload=True)
print(json.dumps([{"name": d.device_name, "token": d.local_auth_token} for d in devices if d.local_auth_token]))
