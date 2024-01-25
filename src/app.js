const express = require("express");
const twitchAuthRoutes = require("./auth/twitch/twitchAuthRoutes");
const startTwitchChatClient = require("./chatClients/twitch/twitchClient");

const app = express();
const PORT = process.env.PORT || 3000;

// Setup Twitch authentication routes
app.use("/auth/twitch", twitchAuthRoutes);

// The Twitch chat client will be started after
// successful authentication in twitchAuthRoutes.js

app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});

module.exports = app;
