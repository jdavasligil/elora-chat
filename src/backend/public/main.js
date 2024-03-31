function escapeRegExp(string) {
  return string.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); // $& means the whole matched string
}

function loadImage(src) {
  return `/imageproxy?url=${encodeURIComponent(src)}`;
}

let ws;
const messageQueue = [];
let processing = false;

// Call initializeWebSocket() only if the user is logged in
function initializeWebSocket() {
  console.log("Initializing WebSocket");
  const wsProtocol = window.location.protocol === "https:" ? "wss" : "ws";
  const wsUrl = `${wsProtocol}://${window.location.host}/ws/chat`;
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
  fetch("/check-session", {
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
        initializeWebSocket(); // Initialize WebSocket connection here
      } else {
        updateUIForLoggedOutUser();
      }
    })
    .catch((error) => {
      console.error("Error checking login status:", error);
      updateUIForLoggedOutUser();
    });
}

function updateUIForLoggedInUser(loggedInServices) {
  // Hide or show the login buttons based on the loggedInService parameter
  const twitchLoginButton = document.getElementById("twitchLoginButton");
  const youtubeLoginButton = document.getElementById("youtubeLoginButton");

  // Assuming 'loggedInService' is a string that indicates which service is logged in
  // For example, it could be "twitch", "youtube", or "both"
  if (loggedInServices.includes("twitch")) {
    twitchLoginButton.style.display = "none";
  } else {
    twitchLoginButton.style.display = "block";
  }

  if (loggedInServices.includes("youtube")) {
    youtubeLoginButton.style.display = "none";
  } else {
    youtubeLoginButton.style.display = "block";
  }

  // Always show the logout button if the user is logged into any service
  document.getElementById("logoutButton").style.display = "block";
}

function updateUIForLoggedOutUser() {
  // Show login buttons and hide the logout button
  document.getElementById("twitchLoginButton").style.display = "block";
  document.getElementById("youtubeLoginButton").style.display = "block";
  document.getElementById("logoutButton").style.display = "none";
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

  document
    .getElementById("youtubeLoginButton")
    .addEventListener("click", function () {
      window.location.href = "/login/youtube";
    });

  document.getElementById("logoutButton").addEventListener("click", logout);
  document
    .getElementById("sendMessageButton")
    .addEventListener("click", sendMessage);

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
