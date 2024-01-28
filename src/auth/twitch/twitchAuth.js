const axios = require("axios");
require("dotenv").config();

const TWITCH_CLIENT_ID = process.env.TWITCH_CLIENT_ID;
const TWITCH_CLIENT_SECRET = process.env.TWITCH_CLIENT_SECRET;

// Placeholder for in-memory token storage (should be replaced with a proper storage solution)
let twitchTokens = {};

const refreshTwitchToken = async () => {
  const tokenUrl = "https://id.twitch.tv/oauth2/token";
  const params = new URLSearchParams({
    grant_type: "refresh_token",
    refresh_token: twitchTokens.refresh_token,
    client_id: TWITCH_CLIENT_ID,
    client_secret: TWITCH_CLIENT_SECRET,
  });

  try {
    const response = await axios.post(tokenUrl, params);
    const newTokens = response.data;
    twitchTokens.access_token = newTokens.access_token;
    twitchTokens.refresh_token =
      newTokens.refresh_token || twitchTokens.refresh_token; // Use the new refresh token if provided, otherwise keep the old one
    twitchTokens.expiry_date = Date.now() + newTokens.expires_in * 1000;
    return twitchTokens.access_token;
  } catch (error) {
    console.error("Error refreshing Twitch token:", error);
    throw error;
  }
};

const getValidTwitchAccessToken = async () => {
  if (!twitchTokens.access_token || Date.now() > twitchTokens.expiry_date) {
    await refreshTwitchToken();
  }
  return twitchTokens.access_token;
};

module.exports = {
  getValidTwitchAccessToken,
  refreshTwitchToken, // Export this if you want to manually trigger a token refresh for testing
};
