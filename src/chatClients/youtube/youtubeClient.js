const axios = require("axios");
const { getValidAccessToken } = require("../../auth/youtube/youtubeAuth");
let isShuttingDown = false;

const getLiveChatMessages = async (liveChatId, pageToken) => {
  if (isShuttingDown) return;

  const accessToken = await getValidAccessToken();

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
    isShuttingDown = true;
    // Graceful shutdown
    if (error.response && error.response.status === 403) {
      // Or any other status code that indicates quota problems
      console.error("API quota error, shutting down YouTube client:", error);
      // Optionally, set up a timer to attempt to restart after a certain period
    } else {
      // Handle other errors
      console.error("An error occurred:", error);
    }
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
        const displayName = message.authorDetails.displayName;
        const messageText = message.snippet.displayMessage; // displayMessage is a string
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
