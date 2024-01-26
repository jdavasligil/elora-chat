const axios = require("axios");
require("dotenv").config();

const YOUTUBE_CLIENT_ID = process.env.YOUTUBE_CLIENT_ID;
const YOUTUBE_CLIENT_SECRET = process.env.YOUTUBE_CLIENT_SECRET;
const YOUTUBE_REDIRECT_URI = process.env.YOUTUBE_REDIRECT_URI;

// Placeholder for in-memory token storage (should be replaced with a proper storage solution)
let tokens = {};

const getAuthUrl = () => {
  const scopes = ["https://www.googleapis.com/auth/youtube.force-ssl"];
  const authUrl = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${YOUTUBE_CLIENT_ID}&redirect_uri=${encodeURIComponent(
    YOUTUBE_REDIRECT_URI
  )}&response_type=code&scope=${encodeURIComponent(scopes.join(" "))}`;
  return authUrl;
};

const getToken = async (code) => {
  const tokenUrl = "https://oauth2.googleapis.com/token";
  const params = new URLSearchParams();
  params.append("client_id", YOUTUBE_CLIENT_ID);
  params.append("client_secret", YOUTUBE_CLIENT_SECRET);
  params.append("code", code);
  params.append("grant_type", "authorization_code");
  params.append("redirect_uri", YOUTUBE_REDIRECT_URI);

  try {
    const response = await axios.post(tokenUrl, params);
    tokens = response.data; // Store tokens
    return tokens;
  } catch (error) {
    throw error;
  }
};

const refreshToken = async () => {
  const tokenUrl = "https://oauth2.googleapis.com/token";
  const params = new URLSearchParams();
  params.append("client_id", YOUTUBE_CLIENT_ID);
  params.append("client_secret", YOUTUBE_CLIENT_SECRET);
  params.append("refresh_token", tokens.refresh_token);
  params.append("grant_type", "refresh_token");

  try {
    const response = await axios.post(tokenUrl, params);
    const newTokens = response.data;
    tokens.access_token = newTokens.access_token;
    tokens.expiry_date = Date.now() + newTokens.expires_in * 1000;
    // You may also update the refresh_token if it is returned in the response
  } catch (error) {
    console.error("Error refreshing YouTube token:", error);
    throw error;
  }
};

const getValidAccessToken = async () => {
  if (Date.now() > tokens.expiry_date) {
    await refreshToken();
  }
  return tokens.access_token;
};

module.exports = {
  getAuthUrl,
  getToken,
  getValidAccessToken,
};
