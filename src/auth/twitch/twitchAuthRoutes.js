const express = require("express");
const router = express.Router();
const axios = require("axios");
require("dotenv").config();
let twitchAccessToken = null;

const startTwitchChatClient = require("../../chatClients/twitch/twitchClient");

// Define the route for starting the Twitch authentication process
router.get("/", (req, res) => {
  // Redirect to Twitch authentication URL
  const redirectUri = `${process.env.TWITCH_REDIRECT_URI}`;
  const clientId = process.env.TWITCH_CLIENT_ID;
  const authUrl = `https://id.twitch.tv/oauth2/authorize?client_id=${clientId}&redirect_uri=${redirectUri}&response_type=code&scope=chat:read+chat:edit`;
  res.redirect(authUrl);
});

// Define the callback route for Twitch authentication
router.get("/callback", async (req, res) => {
  // Extract code from query parameters
  const code = req.query.code;
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
          code: code,
          grant_type: "authorization_code",
          redirect_uri: process.env.TWITCH_REDIRECT_URI,
        },
      }
    );

    // Store the access token
    twitchAccessToken = tokenResponse.data.access_token;

    // Initialize the Twitch chat client
    // Assuming the username is the same as the Twitch username used for authentication
    const twitchUsername = "hp_az"; // Replace with actual username
    startTwitchChatClient(twitchUsername, twitchAccessToken);

    res.send("Authentication successful!"); // You might want to redirect or handle this differently
  } catch (error) {
    console.error("Error during Twitch authentication:", error);
    res.status(500).send("Authentication failed");
  }
});

module.exports = router;
