require("dotenv").config();
const express = require("express");
const axios = require("axios");
const app = express();

const PORT = process.env.PORT || 3000;

// Welcome route
app.get("/", (req, res) => {
  res.send("Welcome to EloraChat! Please go to /auth/twitch to authenticate.");
});

// Redirect to Twitch for authorization with the required scopes
app.get("/auth/twitch", (req, res) => {
  const scope = "chat:read chat:edit channel:read:subscriptions bits:read";
  const redirectUri = encodeURIComponent(process.env.TWITCH_REDIRECT_URI);
  const twitchAuthUrl = `https://id.twitch.tv/oauth2/authorize?client_id=${
    process.env.TWITCH_CLIENT_ID
  }&redirect_uri=${redirectUri}&response_type=code&scope=${encodeURIComponent(
    scope
  )}`;
  res.redirect(twitchAuthUrl);
});

// Twitch callback URL
app.get("/auth/twitch/callback", async (req, res) => {
  const code = req.query.code;
  if (!code) {
    return res
      .status(400)
      .send("No code provided. Please authorize the application.");
  }

  try {
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

    const accessToken = tokenResponse.data.access_token;
    // You now have the access token that you can use to make requests to Twitch's API on behalf of the user.
    // For now, let's just display a simple message to the user.
    res.send("You are successfully authenticated! Token: " + accessToken);
  } catch (error) {
    console.error(
      "Error during the authentication process: ",
      error.response?.data || error.message
    );
    res.status(500).send("Error during the authentication process.");
  }
});

// Start the server
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
