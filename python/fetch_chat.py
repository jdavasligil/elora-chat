from chat_downloader import ChatDownloader
import json
import sys


def fetch_chat(url, message_groups=None):
    chat_downloader = ChatDownloader()
    try:
        chat = chat_downloader.get_chat(url=url, message_groups=message_groups)

        for message in chat:
            # Extract required information if available
            message_data = {
                "message": message.get("message", ""),
                "author": message["author"].get("name", "Unknown"),
                "emotes": message.get("emotes", []),
                "badges": message["author"].get("badges", []),
            }
            print(json.dumps(message_data))  # Print messages as JSON
            sys.stdout.flush()  # Ensure the output is flushed immediately

    except Exception as e:
        print(f"Error fetching chat: {e}", file=sys.stderr)


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python fetch_chat.py <stream_url>", file=sys.stderr)
        sys.exit(1)

    url = sys.argv[1]  # Channel URL

    # Determine message_groups based on the platform in the URL
    message_groups = ["messages"]
    if "twitch.tv" in url:
        message_groups = ["messages"]
    elif "youtube.com" in url:
        message_groups = ["messages"]

    fetch_chat(url, message_groups=message_groups)
