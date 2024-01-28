const express = require("express");
const axios = require("axios");
const { getValidTwitchAccessToken, setTwitchTokens } = require("./twitchAuth");
const {
  startTwitchChatClient,
} = require("../../chatClients/twitch/twitchClient");
require("dotenv").config();

const router = express.Router();

router.get("/", (req, res) => {
  // Redirect to Twitch authentication URL with the required scopes
  const redirectUri = process.env.TWITCH_REDIRECT_URI;
  const clientId = process.env.TWITCH_CLIENT_ID;
  const scopes = "chat:read+chat:edit";
  const authUrl = `https://id.twitch.tv/oauth2/authorize?client_id=${clientId}&redirect_uri=${encodeURIComponent(
    redirectUri
  )}&response_type=code&scope=${scopes}`;
  res.redirect(authUrl);
});

router.get("/callback", async (req, res) => {
  const { code } = req.query;
  if (!code) {
    return res.status(400).send("No code provided");
  }

  try {
    // Exchange the code for an access token
    const tokenResponse = await axios.post(
      "https://id.twitch.tv/oauth2/token",
      null,
      {
        params: {
          client_id: process.env.TWITCH_CLIENT_ID,
          client_secret: process.env.TWITCH_CLIENT_SECRET,
          code,
          grant_type: "authorization_code",
          redirect_uri: redirectUri,
        },
      }
    );

    // Store the tokens using setTwitchTokens function
    setTwitchTokens({
      access_token: tokenResponse.data.access_token,
      refresh_token: tokenResponse.data.refresh_token,
      expiry_date: Date.now() + tokenResponse.data.expires_in * 1000,
    });

    // Start the Twitch chat client
    startTwitchChatClient(await getValidTwitchAccessToken());

    res.send("Twitch Authentication successful!");
  } catch (error) {
    console.error("Error during Twitch authentication:", error);
    res.status(500).send("Twitch Authentication failed");
  }
});

module.exports = router;
