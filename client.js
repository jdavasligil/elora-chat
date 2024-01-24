const tmi = require("tmi.js");

// This function starts a Twitch chat client with the provided username and token
function startChatClient(username, token) {
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

  const client = new tmi.client(opts);

  client.on("message", (channel, userstate, message, self) => {
    if (self) {
      return;
    } // Ignore messages from the bot itself
    console.log(`[${channel}] ${userstate["display-name"]}: ${message}`);
  });

  client.on("connected", (addr, port) => {
    console.log(`* Connected to ${addr}:${port}`);
  });

  // Connect the client to the server
  client.connect();
}

module.exports = { startChatClient };
