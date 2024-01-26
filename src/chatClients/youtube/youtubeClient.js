const axios = require("axios");
const { getValidAccessToken } = require("../../auth/youtube/youtubeAuth");

const getLiveChatMessages = async (liveChatId, pageToken) => {
  const accessToken = getValidAccessToken();

  try {
    const response = await axios.get(
      "https://www.googleapis.com/youtube/v3/liveChat/messages",
      {
        params: {
          liveChatId,
          part: "id,snippet,authorDetails",
          pageToken,
          maxResults: 200,
        },
        headers: {
          Authorization: `Bearer ${accessToken}`,
          Accept: "application/json",
        },
      }
    );

    return response.data;
  } catch (error) {
    console.error("Error fetching YouTube live chat messages:", error);
    throw error;
  }
};

const startChatClient = (liveChatId) => {
  let nextPageToken;
  let pollingIntervalMillis = 10000;

  const pollMessages = async () => {
    try {
      const chatData = await getLiveChatMessages(liveChatId, nextPageToken);
      const {
        items,
        pollingIntervalMillis: interval,
        nextPageToken: token,
      } = chatData;

      nextPageToken = token;
      pollingIntervalMillis = interval || pollingIntervalMillis;

      items.forEach((message) => {
        const { displayName, messageText } = message.snippet.displayMessage;
        console.log(`${displayName}: ${messageText}`);
      });

      setTimeout(pollMessages, pollingIntervalMillis);
    } catch (error) {
      console.error("Error in pollMessages:", error);
      setTimeout(pollMessages, pollingIntervalMillis);
    }
  };

  pollMessages();
};

module.exports = {
  startChatClient,
};
