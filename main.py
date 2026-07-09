#!/usr/bin/env python3
import sys
from config import save_repo, load_repo
from git_wrapper import setup_repo, save_and_push, sync_changes, show_status, run_command

def print_help():
    print("""
gittool - Making Git simple.

Available Commands:
  gittool init             Initialize a brand new local repository
  gittool repo add <url>   Link a remote GitHub/GitLab repository
  gittool status           See what files you have changed
  gittool save ["message"] Track, commit, and PUSH all changes (message optional)
  gittool sync             Pull latest remote changes and push yours
    """)

def main():
    args = sys.argv[1:]

    if not args or args[0] in ["--help", "-h", "help"]:
        print_help()
        return

    command = args[0]

    if command == "init":
        print("Initializing local repository...")
        run_command(["git", "init"])
        run_command(["git", "branch", "-M", "main"])
        print("Successfully initialized local repository on branch 'main'!")

    elif command == "repo":
        if len(args) >= 3 and args[1] == "add":
            repo_url = args[2]
            save_repo(repo_url)
            setup_repo(repo_url)
        else:
            print("Usage: gittool repo add <repository_url>")

    elif command == "status":
        show_status()

    elif command == "save":
        message = args[1] if len(args) >= 2 else None
        save_and_push(message)

    elif command == "sync":
        repo_url = load_repo()
        if not repo_url:
            print("Error: No repository linked yet. Run 'gittool repo add <url>' first.")
            return
        sync_changes()

    else:
        print(f"Unknown command: '{command}'")
        print_help()

if __name__ == "__main__":
    main()
