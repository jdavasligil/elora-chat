from chat_downloader import ChatDownloader

url = "https://www.twitch.tv/Johnstone"  # Replace with an actual Twitch channel URL
chat_downloader = ChatDownloader()

chat = chat_downloader.get_chat(url, message_groups=["messages"], max_messages=1)

for message in chat:
    print(message)  # This will print out the entire message dictionary
    if "emotes" in message:  # Check if emotes are included
        print("Emotes:", message["emotes"])
