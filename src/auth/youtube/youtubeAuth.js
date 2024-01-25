const axios = require("axios");
require("dotenv").config();

const YOUTUBE_CLIENT_ID = process.env.YOUTUBE_CLIENT_ID;
const YOUTUBE_CLIENT_SECRET = process.env.YOUTUBE_CLIENT_SECRET;
const YOUTUBE_REDIRECT_URI = process.env.YOUTUBE_REDIRECT_URI;

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
    return response.data; // { access_token, refresh_token, expires_in, etc. }
  } catch (error) {
    throw error;
  }
};

module.exports = {
  getAuthUrl,
  getToken,
};
