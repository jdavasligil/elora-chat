# EloraChat/python/fetch_chat.py

from chat_downloader import ChatDownloader
import json
import sys


def fetch_chat(url, message_groups=None, max_messages=10):
    try:
        chat_downloader = ChatDownloader()
        chat = chat_downloader.get_chat(
            url=url,
            message_groups=message_groups or [],
            max_messages=max_messages,
        )
        messages = [message for message in chat]
        print(json.dumps(messages))  # Print messages as JSON
    except Exception as e:
        print(f"Error fetching chat: {e}", file=sys.stderr)


if __name__ == "__main__":
    url = sys.argv[1]  # Channel URL

    # Default values
    message_groups = []
    max_messages = 10

    # Extract platform from URL to determine message_groups
    if "youtube" in url:
        message_groups = ["messages"]
    elif "twitch" in url:
        message_groups = ["messages"]

    # Optionally, parse max_messages from command-line arguments
    if len(sys.argv) > 2:
        try:
            max_messages = int(sys.argv[2])
        except ValueError:
            print(
                "Warning: Invalid max_messages value. Using default.", file=sys.stderr
            )

    fetch_chat(url, message_groups=message_groups, max_messages=max_messages)
