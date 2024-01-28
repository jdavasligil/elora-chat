const tmi = require("tmi.js");

// This function starts a Twitch chat client with the provided username and token
function startChatClient(username, token) {
  console.log(username);
  const opts = {
    options: { debug: true },
    connection: {
      secure: true,
      reconnect: true,
    },
    identity: {
      username,
      password: `oauth:${token}`,
    },
    channels: [username],
  };

  const client = new tmi.Client(opts);

  client.on("message", (channel, userstate, message, self) => {
    if (self) return; // Ignore messages from the bot itself

    // Handle incoming messages here
    console.log(`Message from ${userstate["display-name"]}: ${message}`);
  });

  client.connect().catch(console.error);
}

module.exports = startChatClient;
