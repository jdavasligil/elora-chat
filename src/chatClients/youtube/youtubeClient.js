const axios = require("axios");
const { getValidAccessToken } = require("../../auth/youtube/youtubeAuth");

const getLiveChatMessages = async (liveChatId, pageToken) => {
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
    console.error(
      "An error occurred while fetching YouTube live chat messages:",
      error
    );
    throw error;
  }
};

const startChatClient = (liveChatId) => {
  let nextPageToken = null;

  const pollMessages = async () => {
    try {
      const chatData = await getLiveChatMessages(liveChatId, nextPageToken);
      const {
        items,
        pollingIntervalMillis,
        nextPageToken: newToken,
      } = chatData;

      nextPageToken = newToken; // Update the nextPageToken for the next poll

      // Process and log chat messages
      items.forEach((message) => {
        const displayName = message.authorDetails.displayName;
        const messageText = message.snippet.displayMessage;
        console.log(`${displayName}: ${messageText}`);
      });

      // Poll for more messages after the specified interval
      setTimeout(pollMessages, pollingIntervalMillis);
    } catch (error) {
      console.error("Error in pollMessages:", error);
      // Retry the poll after some time if there is an error
      setTimeout(pollMessages, 10000); // Default to 10 seconds
    }
  };

  // Start polling
  pollMessages();
};

module.exports = {
  startChatClient,
};
