from chat_downloader import ChatDownloader
import json
import sys
import time


def continuous_fetch_chat(url, message_groups=None):
    try:
        chat_downloader = ChatDownloader()
        chat = chat_downloader.get_chat(url=url, message_groups=message_groups or [])

        for message in chat:
            # Print each message as JSON to stdout
            print(json.dumps(message))
            sys.stdout.flush()  # Ensure Python flushes the printed messages immediately
            time.sleep(
                0.1
            )  # Sleep briefly to simulate real-time fetching; adjust as needed
    except Exception as e:
        print(f"Error fetching chat: {e}", file=sys.stderr)


if __name__ == "__main__":
    url = sys.argv[1]  # Channel URL

    # Determine message_groups based on the platform in the URL
    message_groups = []
    if "youtube" in url:
        message_groups = ["messages"]
    elif "twitch" in url:
        message_groups = ["messages"]

    continuous_fetch_chat(url, message_groups=message_groups)
