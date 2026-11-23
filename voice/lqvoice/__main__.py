import os

import uvicorn

if __name__ == "__main__":
    uvicorn.run(
        "lqvoice.server:create_app",
        factory=True,
        host=os.environ.get("VOICE_HOST", "0.0.0.0"),
        port=int(os.environ.get("VOICE_PORT", "8000")),
        access_log=False,  # no URLs or addresses of callers in the logs
        ws_ping_interval=20,
    )
