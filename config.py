import json
import os

CONFIG_FILE = ".gittool_config.json"

def save_repo(repo_url: str):
    """Saves the repository URL to a local config file."""
    data = {"repo": repo_url}
    with open(CONFIG_FILE, "w") as f:
        json.dump(data, f, indent=4)

def load_repo() -> str | None:
    """Loads the repository URL if it exists."""
    if os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE, "r") as f:
            data = json.load(f)
            return data.get("repo")
    return None
