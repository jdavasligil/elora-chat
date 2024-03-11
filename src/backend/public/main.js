function escapeRegExp(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); // $& means the whole matched string
}

function loadImage(src) {
  return `/imageproxy?url=${encodeURIComponent(src)}`;
}

let ws;
const messageQueue = [];
let processing = false;

function initializeWebSocket() {
  if (
    ws &&
    (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)
  ) {
    console.log("WebSocket is already open or connecting.");
    return;
  }

  const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(`${wsProtocol}://${window.location.host}/ws/chat`);

  ws.onopen = function () {
    console.log("WebSocket Connection established");
  };

  ws.onmessage = function (event) {
    var msg = event.data;
    if (msg === "__keepalive__") {
      return;
    }

    try {
      var parsedMsg = JSON.parse(msg);
      messageQueue.push(parsedMsg);
      if (!processing) {
        processMessageQueue();
      }
    } catch (e) {
      console.error("Error parsing message:", msg, e);
    }
  };

  ws.onerror = function (error) {
    console.log("WebSocket Error: " + error);
  };

  ws.onclose = function () {
    console.log("WebSocket Connection closed");
    setTimeout(initializeWebSocket, 5000); // Attempt to reconnect after 5 seconds
  };
}

function processMessageQueue() {
  if (messageQueue.length === 0) {
    processing = false;
    return;
  }

  // If there's a large number of messages, only keep the last N
  const N = 60;
  if (messageQueue.length > N) {
    messageQueue.splice(0, messageQueue.length - N);
  }

  processing = true;
  const message = messageQueue.shift();
  const container = document.getElementById("chat-container");
  const messageElement = document.createElement("div");
  messageElement.classList.add("chat-message");

  let sourceBadgeHTML = "";
  if (message.source === "Twitch") {
    sourceBadgeHTML =
      '<img src="twitch-tile.svg" class="badge-icon" title="Twitch">';
  } else if (message.source === "YouTube") {
    sourceBadgeHTML =
      '<img src="youtube-tile.svg" class="badge-icon" title="YouTube">';
  }

  let badgesHTML = sourceBadgeHTML; // Start with the source badge
  message.badges.forEach((badge) => {
    if (badge.icons && badge.icons.length > 0) {
      const badgeImg = document.createElement("img");
      badgeImg.className = "badge-icon";
      badgeImg.title = badge.title;
      badgeImg.src = loadImage(badge.icons[0].url);
      badgesHTML += badgeImg.outerHTML;
    }
  });

  let messageWithEmotes = message.message;
  if (message.emotes && message.emotes.length > 0) {
    message.emotes.forEach((emote) => {
      const emoteImg = document.createElement("img");
      emoteImg.className = "emote-img";
      emoteImg.alt = emote.name;
      emoteImg.src = loadImage(emote.images[0].url);

      const escapedEmoteName = escapeRegExp(emote.name);
      const emoteRegex = new RegExp(escapedEmoteName, "g");
      messageWithEmotes = messageWithEmotes.replace(
        emoteRegex,
        emoteImg.outerHTML
      );
    });
  }

  // Replace black usernames with higher contrast color to show up on black background
  if (message.colour === "#000000") {
    message.colour = "#CCCCCC"; // Light grey for visibility
  }

  messageElement.innerHTML =
    badgesHTML +
    `<b><span style="color: ${message.colour}">${message.author}:</span></b> ${messageWithEmotes}`;
  container.appendChild(messageElement);
  container.scrollTop = container.scrollHeight; // Scroll to the bottom

  // Limit the number of messages in the chat container to 60
  let chatMessages = container.querySelectorAll(".chat-message");
  while (chatMessages.length > 60) {
    const oldestMessage = chatMessages[0];
    if (oldestMessage) {
      // Make sure the oldest message exists before removing it
      oldestMessage.parentNode.removeChild(oldestMessage);
    }
    // Update the chatMessages NodeList after removal
    chatMessages = container.querySelectorAll(".chat-message");
  }

  // Continue processing after a delay
  setTimeout(processMessageQueue, 0); // Delay of x ms between messages
}

// Initialize WebSocket connection
initializeWebSocket();

// Visibility change event handler
function handleVisibilityChange() {
  if (document.hidden) {
    console.log("Tab is inactive");
    // Do not close WebSocket; let the server handle timeouts
  } else {
    console.log("Tab is active, checking WebSocket connection.");
    initializeWebSocket(); // Ensure WebSocket is connected
  }
}

// Event listeners for page visibility and other relevant events
document.addEventListener("visibilitychange", handleVisibilityChange);
window.addEventListener("pageshow", handleVisibilityChange); // Triggered when navigating to a page
window.addEventListener("online", handleVisibilityChange); // Triggered when going online
window.addEventListener("focus", handleVisibilityChange); // Triggered when tab gains focus

// Optional: Clear WebSocket reference on page unload
window.addEventListener("beforeunload", function () {
  if (ws) {
    ws.close();
    ws = null;
  }
});

// Handle Popout Chat functionality
document.addEventListener("DOMContentLoaded", (event) => {
  const popoutChatBtn = document.getElementById("popoutChatBtn");
  let popoutWindow;

  popoutChatBtn.addEventListener("click", () => {
    const popoutFeatures =
      "scrollbars=no,resizable=yes,status=no,location=no,toolbar=no,menubar=no";
    popoutWindow = window.open("chat.html", "ChatPopout", popoutFeatures);
  });
});
