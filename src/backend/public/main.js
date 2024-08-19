const useDeployedApi = false; // NOTE: be sure to set this to false when building/deploying
const deployedUrl = 'https://elorachat.wizg.xyz';

function escapeRegExp(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); // $& means the whole matched string
}

function loadImage(src) {
  return `${useDeployedApi ? deployedUrl : ''}/imageproxy?url=${encodeURIComponent(src)}`;
}

let ws;
const messageQueue = [];
let processing = false;

// Call initializeWebSocket() only if the user is logged in
function initializeWebSocket() {
  console.log("Initializing WebSocket");
  const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";

  const localUrl = `${wsProtocol}://${window.location.host}`;
  const wsUrl = `${useDeployedApi ? deployedUrl : localUrl}/ws/chat`;
  
  if (
    ws &&
    (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)
  ) {
    console.log(
      "WebSocket is already connected or connecting. No action taken."
    );
    return;
  }

  console.log("WebSocket URL:", wsUrl);
  ws = new WebSocket(wsUrl);

  ws.onopen = function () {
    console.log("WebSocket Connection established");
  };

  ws.onmessage = function (event) {
    // console.log("Message received: ", event.data);
    const msg = event.data;
    if (msg === "__keepalive__") {
      return;
    }

    try {
      const parsedMsg = JSON.parse(msg);
      messageQueue.push(parsedMsg);
      if (!processing) {
        processMessageQueue();
      }
    } catch (e) {
      console.error("Error parsing message:", msg, e);
    }
  };

  ws.onerror = function (error) {
    console.error("WebSocket Error:", error);
  };

  ws.onclose = function () {
    console.log("WebSocket Connection closed. Attempting to reconnect...");
    // Removed the setTimeout here to avoid automatic reconnection.
    // The reconnection attempt will be managed by the visibility change or manual triggers.
  };
}

function sanitizeMessage(message) {
  // replace < and > with HTML entities to prevent XSS attacks
  return message.replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function addMessageEffects(message) {
  const colors = [
    'yellow', 'red', 'green', 'cyan', 'purple', 'pink', 'rainbow',
    'glow1', 'glow2', 'glow3', 'flash1', 'flash2', 'flash3'
  ];
  const colorCommands = colors.reduce((accumulator, color) => ({
    ...accumulator,
    [color]: `color-${color}`
  }), {});

  const commands = {
    ...colorCommands,
    bold: 'text-bold',
    italic: 'text-italic',
    wave: 'effect-wave',
    shake: 'effect-shake'
  };

  const lastCommandIndex = message.indexOf(': ');
  const effectNames = lastCommandIndex >= 0 ? message.substr(0, lastCommandIndex).split(':') : [];
  const messageText = lastCommandIndex >= 0 ? message.substr(lastCommandIndex + 2) : message;
  const effects = effectNames
                  .map(effect => commands.hasOwnProperty(effect) ? commands[effect] : null)
                  .filter(value => !!value)
                  .join(' ');

  return { messageText, effects };
}

function processMessageQueue() {
  // console.log("Processing message queue", messageQueue);
  if (messageQueue.length === 0) {
    processing = false;
    return;
  }

  // If there's a large number of messages, only keep the last N
  const N = 200;
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

  let messageWithEmotes = sanitizeMessage(message.message);
  const { messageText, effects } = addMessageEffects(messageWithEmotes);
  messageWithEmotes = messageText;

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
    `<b><span style="color: ${message.colour}">${message.author}:</span></b> <span class="${effects}">${messageWithEmotes}</span>`;
  // Prepend new message at the start of the container, which visually appears at the bottom
  container.insertBefore(messageElement, container.firstChild);

  // Scroll to the bottom of the chat container
  // The 'flex-direction: column-reverse' means we actually want to scroll to the top
  container.scrollTop = 0;

  // Limit the number of messages in the chat container to N
  let chatMessages = container.querySelectorAll(".chat-message");
  while (chatMessages.length > N) {
    const oldestMessage = chatMessages[chatMessages.length - 1];
    if (oldestMessage) {
      oldestMessage.parentNode.removeChild(oldestMessage);
    }
    // Update the chatMessages NodeList after removal
    chatMessages = container.querySelectorAll(".chat-message");
  }

  // Continue processing after a delay
  setTimeout(processMessageQueue, 0); // Delay of x ms between messages
}

function checkLoginStatus() {
  fetch(`${useDeployedApi ? deployedUrl : ''}/check-session`, {
    method: "GET",
    credentials: "include", // Important for cookies to be sent with the request
  })
    .then((response) => {
      if (response.ok) {
        return response.json(); // Process the body of the response
      } else {
        throw new Error("Session check failed");
      }
    })
    .then((sessionData) => {
      if (sessionData.services && sessionData.services.length > 0) {
        updateUIForLoggedInUser(sessionData.services);
      } else {
        updateUIForLoggedOutUser();
      }
      initializeWebSocket(); // Initialize WebSocket connection here, regardless of login status
    })
    .catch((error) => {
      console.error("Error checking login status:", error);
      updateUIForLoggedOutUser();
      initializeWebSocket(); // Initialize WebSocket connection even if there's an error
    });
}

function updateUIForLoggedInUser(loggedInServices) {
  const twitchLoginButton = document.getElementById("twitchLoginButton");
  const popoutChatBtn = document.getElementById("popoutChatBtn");
  const refreshServerBtn = document.getElementById("refreshServerBtn");

  // Display the Twitch login button only if Twitch is not logged in
  twitchLoginButton.style.display = loggedInServices.includes("twitch")
    ? "none"
    : "block";

  // Since YouTube login is handled by the backend, we show popout and refresh buttons if Twitch is logged in
  const isTwitchLoggedIn = loggedInServices.includes("twitch");
  popoutChatBtn.style.display = isTwitchLoggedIn ? "block" : "none";
  refreshServerBtn.style.display = isTwitchLoggedIn ? "block" : "none";

  // Always show the logout button if Twitch is logged in
  document.getElementById("logoutButton").style.display = isTwitchLoggedIn
    ? "block"
    : "none";
}

function updateUIForLoggedOutUser() {
  // Show the Twitch login button and hide the logout button, popout, and refresh buttons
  document.getElementById("twitchLoginButton").style.display = "block";
  document.getElementById("logoutButton").style.display = "none";
  document.getElementById("popoutChatBtn").style.display = "none";
  document.getElementById("refreshServerBtn").style.display = "none";
}

function logout() {
  // Correctly handle logout by making a request to the backend endpoint
  fetch("/logout", {
    method: "POST",
    credentials: "include", // Important for cookies to be sent with the request
  })
    .then((response) => {
      if (response.ok) {
        localStorage.removeItem("sessionToken"); // Optionally remove from localStorage if used elsewhere
        updateUIForLoggedOutUser();
        window.location.href = "/";
      }
    })
    .catch((error) => console.error("Error logging out:", error));
}

document.addEventListener("DOMContentLoaded", () => {
  checkLoginStatus();

  document
    .getElementById("twitchLoginButton")
    .addEventListener("click", function () {
      window.location.href = "/login/twitch";
    });

  // Removed event listener for YouTube login button

  document.getElementById("logoutButton").addEventListener("click", logout);
  document
    .getElementById("sendMessageButton")
    .addEventListener("click", sendMessage);

  // Remaining event listeners unchanged
  document.addEventListener("visibilitychange", handleVisibilityChange);
  window.addEventListener("pageshow", handleVisibilityChange);
  window.addEventListener("online", handleVisibilityChange);
  window.addEventListener("focus", handleVisibilityChange);

  window.addEventListener("beforeunload", function () {
    if (ws) {
      ws.close();
      ws = null;
    }
  });

  const popoutChatBtn = document.getElementById("popoutChatBtn");
  const refreshServerBtn = document.getElementById("refreshServerBtn");

  popoutChatBtn.addEventListener("click", () => {
    const popoutFeatures =
      "scrollbars=no,resizable=yes,status=no,location=no,toolbar=no,menubar=no";
    window.open("chat.html", "ChatPopout", popoutFeatures);
  });

  refreshServerBtn.addEventListener("click", () => {
    fetch("/restart-server", { method: "POST" })
      .then((response) => response.json())
      .then((data) => console.log(data))
      .catch((error) => console.error("Error:", error));
  });
});

function handleVisibilityChange() {
  if (!document.hidden) {
    console.log("Tab is active, checking WebSocket connection.");
    if (checkLoginStatus()) {
      if (!ws || ws.readyState === WebSocket.CLOSED) {
        initializeWebSocket();
      }
    }
  }
}

// Function to send a message
function sendMessage() {
  const messageInput = document.getElementById("messageInput");
  const message = messageInput.value;
  if (!message) {
    console.log("No message to send");
    return;
  }

  messageInput.value = ""; // Clear the input immediately

  fetch("/auth/send-message", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ message: message }),
    credentials: "include", // Important for session handling
  })
    .then((response) => {
      if (response.ok) {
        console.log("Message sent successfully");
      } else {
        console.error("Failed to send message");
      }
    })
    .catch((error) => console.error("Error sending message:", error));
}

document
  .getElementById("messageInput")
  .addEventListener("keydown", function (e) {
    if (e.key === "Enter") {
      sendMessage();
    }
  });
