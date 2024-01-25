const express = require("express");
const youtubeAuth = require("./youtubeAuth");

const router = express.Router();

// Temporary in-memory storage for tokens
// In a production environment, you should use a secure database instead
let youtubeTokens = {};

router.get("/", (req, res) => {
  // Redirect to YouTube authentication URL
  const authUrl = youtubeAuth.getAuthUrl();
  res.redirect(authUrl);
});

router.get("/callback", async (req, res) => {
  const { code } = req.query;
  if (!code) {
    return res.status(400).send("No code provided");
  }

  try {
    // Exchange the code for an access token
    const data = await youtubeAuth.getToken(code);

    // Store the access and refresh tokens in memory
    // You should encrypt these tokens and store them in a secure database
    youtubeTokens = {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      expiryDate: new Date(Date.now() + data.expires_in * 1000),
    };

    // TODO: Start the YouTube client or perform other actions as needed

    res.send("YouTube Authentication successful!");
  } catch (error) {
    console.error("Error during YouTube authentication:", error);
    res.status(500).send("YouTube Authentication failed");
  }
});

module.exports = router;
