const express = require("express");
const axios = require("axios");
const { getAuthUrl, getToken, getValidAccessToken } = require("./youtubeAuth");
const { startChatClient } = require("../../chatClients/youtube/youtubeClient");
require("dotenv").config();

const router = express.Router();

// Helper function to get the liveChatId
async function getLiveChatId(accessToken) {
  try {
    // Make a request to the YouTube Data API's liveBroadcasts.list endpoint
    const response = await axios.get(
      "https://www.googleapis.com/youtube/v3/liveBroadcasts",
      {
        params: {
          part: "snippet",
          broadcastStatus: "active",
          broadcastType: "all",
        },
        headers: {
          Authorization: `Bearer ${accessToken}`,
          Accept: "application/json",
        },
      }
    );

    // Check if there are any active broadcasts
    const broadcasts = response.data.items;
    if (!broadcasts || broadcasts.length === 0) {
      throw new Error("No active live broadcasts found.");
    }

    // Assuming the first broadcast is the one we want
    const liveBroadcast = broadcasts[0];

    // Extract the liveChatId
    const liveChatId = liveBroadcast.snippet.liveChatId;
    if (!liveChatId) {
      throw new Error(
        "The live broadcast does not have an associated live chat."
      );
    }

    return liveChatId;
  } catch (error) {
    console.error("Error fetching liveChatId:", error.response || error);
    throw error;
  }
}

router.get("/", (req, res) => {
  res.redirect(getAuthUrl());
});

router.get("/callback", async (req, res) => {
  const { code } = req.query;
  if (!code) {
    return res.status(400).send("No code provided");
  }

  try {
    const data = await getToken(code);
    // Store or update the tokens in your preferred method here
    // For now, it will use the access token directly from the data object

    // Retrieve the liveChatId for the active live broadcast
    const liveChatId = await getLiveChatId(data.access_token);

    // Start the YouTube chat client with the liveChatId
    startChatClient(liveChatId);

    res.send("YouTube Authentication successful!");
  } catch (error) {
    console.error(
      "Error during YouTube authentication or starting chat client:",
      error
    );
    res.status(500).send("YouTube Authentication failed");
  }
});

module.exports = router;
