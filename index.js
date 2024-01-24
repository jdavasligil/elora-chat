require("dotenv").config();
const express = require("express");
const axios = require("axios");
const app = express();

const PORT = process.env.PORT || 3000;

// Routes
app.get("/", (req, res) => {
  res.send("Welcome to EloraChat!");
});

app.get("/auth/twitch", (req, res) => {
  const scope = "chat:read+chat:edit+channel:read:subscriptions";
  const redirectUri = `${process.env.TWITCH_REDIRECT_URI}`;
  const twitchAuthUrl = `https://id.twitch.tv/oauth2/authorize?client_id=${
    process.env.TWITCH_CLIENT_ID
  }&redirect_uri=${encodeURIComponent(
    redirectUri
  )}&response_type=code&scope=${scope}`;

  res.redirect(twitchAuthUrl);
});

app.get("/auth/twitch/callback", async (req, res) => {
  const code = req.query.code;
  if (!code) {
    return res.status(400).send("No code provided");
  }

  try {
    const response = await axios.post(
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

    const accessToken = response.data.access_token;
    const refreshToken = response.data.refresh_token;

    // Here you would typically save the tokens and redirect to your frontend application
    console.log("Access Token:", accessToken);
    console.log("Refresh Token:", refreshToken);

    res.send("Authentication successful!"); // Placeholder response
  } catch (error) {
    console.error(
      "Error exchanging code for tokens:",
      error.response?.data || error.message
    );
    res.status(500).send("Internal Server Error");
  }
});

// Start the server
app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
