# EloraChat/python/fetch_chat.py

from chat_downloader import ChatDownloader
import json
import sys


def fetch_chat(platform, url):
    chat = ChatDownloader().get_chat(
        url, max_messages=10
    )  # Adjust max_messages as needed
    messages = [message.json() for message in chat]
    print(json.dumps(messages))  # Print messages as JSON


if __name__ == "__main__":
    platform = sys.argv[1]  # 'twitch' or 'youtube'
    url = sys.argv[2]  # Channel URL
    fetch_chat(platform, url)
