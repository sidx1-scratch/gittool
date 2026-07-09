import subprocess
import sys
import os
from datetime import datetime

def run_command(command: list[str]) -> bool:
    """Helper function to run system commands safely."""
    try:
        result = subprocess.run(command, check=True, text=True, capture_output=True)
        if result.stdout:
            print(result.stdout.strip())
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error: {e.stderr.strip()}", file=sys.stderr)
        return False

def setup_repo(repo_url: str):
    """Initializes git and links the remote repository."""
    print("Initializing local repository...")
    if not os.path.exists(".git"):
        run_command(["git", "init"])
        run_command(["git", "branch", "-M", "main"])
    
    print(f"Linking remote repository: {repo_url}")
    subprocess.run(["git", "remote", "remove", "origin"], capture_output=True)
    run_command(["git", "remote", "add", "origin", repo_url])
    print("Repository successfully linked!")

def save_and_push(message: str | None):
    """Stages all changes, commits them, and immediately pushes to main."""
    if not message:
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        message = f"Automatic backup: {timestamp}"

    print("Staging all changes...")
    run_command(["git", "add", "."])
    
    print(f"Saving changes with message: '{message}'")
    if not run_command(["git", "commit", "-m", message]):
        print("Nothing to save or push.")
        return
    
    print("Pushing changes immediately to remote 'main'...")
    run_command(["git", "push", "-u", "origin", "main"])
    print("Save and push complete!")

def sync_changes():
    """Pulls the latest updates from remote and pushes your local changes."""
    print("Fetching latest updates from remote...")
    run_command(["git", "pull", "origin", "main", "--rebase"])
    
    print("Pushing your changes to remote...")
    run_command(["git", "push", "-u", "origin", "main"])
    print("Sync complete!")

def show_status():
    """Shows what files have been changed."""
    run_command(["git", "status", "-s"])
