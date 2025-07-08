import os
import time
from dotenv import load_dotenv
import modules.utils as utils
load_dotenv(os.path.expanduser("~/.config/anitr-cli/config"))

default_ui = utils.get_env("default_ui", "DEFAULT_UI", default="tui")
discord_rpc = utils.get_env("discord_rpc", "DISCORD_RPC", default="enabled")
save_position_on_quit = utils.get_bool_env(
    "save_position_on_quit", "SAVE_POSITION_ON_QUIT", default="false")

sources = ["AnimeciX (anm.cx)", "OpenAnime (openani.me)"]  # Kaynaklar

# Global Discord RPC durumu ve zaman
discord_client_id = "1383421771159572600"
rpc = None
rpc_initialized = False
start_time = int(time.time())

# Paths
anime_details = "/tmp/anime_details"
error_log = "/tmp/anitr-cli-error.log"
config_path = os.path.expanduser("~/.config/anitr-cli/config")

# GitHub
github_repo = "https://github.com/xeyossr/anitr-cli"

GITHUB_REPO = "xeyossr/anitr-cli"
CURRENT_VERSION = "3.6.0-2"
