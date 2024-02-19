from chat_downloader import ChatDownloader
import json
import sys


def fetch_chat(url, message_groups=None):
    try:
        chat_downloader = ChatDownloader()
        chat = chat_downloader.get_chat(
            url=url, message_groups=message_groups or ["messages"]
        )

        for message in chat:
            # Initialize default color (grey for YouTube non-members)
            color = "#808080"

            if "colour" in message:  # Twitch messages
                color = message["colour"]
            else:  # YouTube messages
                for badge in message["author"].get("badges", []):
                    title = badge["title"].lower()
                    if "owner" in title:
                        color = "#FFFF00"  # Yellow
                    elif "moderator" in title:
                        color = "#0000FF"  # Blue
                    elif "member" in title:
                        color = "#008000"  # Green

            # Include color in the message data
            message_data = {
                "message": message["message"],
                "author": message["author"]["name"],
                "emotes": message.get("emotes", []),
                "badges": message["author"].get("badges", []),
                "colour": color,  # Add the color here
            }
            print(json.dumps(message_data))  # Print messages as JSON
            sys.stdout.flush()  # Ensure the output is flushed immediately
    except Exception as e:
        print(f"Error fetching chat: {e}", file=sys.stderr)


if __name__ == "__main__":
    url = sys.argv[1]  # Channel URL

    # Determine message_groups based on the platform in the URL
    message_groups = ["messages"] if "youtube" in url or "twitch" in url else []

    fetch_chat(url, message_groups=message_groups)
