from chat_downloader import ChatDownloader
import json
import sys

# Constant must be a power of 2
MESSAGE_BUF_MAX = 128

# Performs mod 2 using a bit mask (& operation)
MESSAGE_MOD_MASK = MESSAGE_BUF_MAX - 1


def fetch_chat(url, message_groups=None):
    # Track which unique messages have been  before
    message_id_set = set()
    # Keep  messages in a circle buffer to continuously delete old messages
    message_id_buf = [0]*MESSAGE_BUF_MAX
    message_id_idx = 0
    try:
        chat_downloader = ChatDownloader()
        while True:
            chat = chat_downloader.get_chat(
                url=url,
                message_groups=message_groups or ["messages"],
                interruptible_retry=False,
                inactivity_timeout=10.0,
            )
            assert chat is not None, "chat is None"
            for message in chat:
                id = message["message_id"]
                if id in message_id_set:
                    continue
                else:
                    # Store the unique id
                    message_id_set.add(id)
                    message_id_buf[message_id_idx] = id
                    # Move to the next oldest id in the circle buffer
                    message_id_idx = (message_id_idx + 1) & MESSAGE_MOD_MASK
                    old_id = message_id_buf[message_id_idx]
                    # Discard the oldest id since we only need the last 100
                    message_id_set.discard(old_id)

                # Initialize default color (grey for YouTube non-members)
                color = "#808080"

                if "colour" in message:  # Twitch messages
                    author = message["author"]["display_name"]
                    color = message["colour"]
                else:  # YouTube messages
                    author = message["author"]["name"]
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
                    "author": author,
                    "emotes": message.get("emotes", []),
                    "badges": message["author"].get("badges", []),
                    "colour": color,  # Add the color here
                }

                print(json.dumps(message_data), flush=True)

    except Exception as e:
        print(f"fetch_chat: Error fetching chat: {e}", file=sys.stderr)
        return


if __name__ == "__main__":
    url = sys.argv[1]  # Channel URL

    # Determine message_groups based on the platform in the URL
    message_groups = ["messages"] if "youtube" in url or "twitch" in url else []

    fetch_chat(url, message_groups=message_groups)
