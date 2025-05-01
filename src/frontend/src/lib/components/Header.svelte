<script>
  import { TwitchIcon } from './icons';
</script>

<div id="header">
  <img src="images/logo.jpg" alt="Logo" id="logo" class="login-button" />

  <div class="buttons">
    <!-- Twitch Login Button -->
    <button
      id="twitch-login-button"
      title="Login with Twitch"
      class="login-button"
      style="display: block;"
    >
      <TwitchIcon alt="Login with Twitch" />
    </button>

    <!-- Logout Button -->
    <button id="logout-button" style="display: none">logout</button>

    <!-- Refresh and Popout Buttons -->
    <button id="refresh-server-button" title="Refresh Server" style="display: none;">
      <img src="images/refresh2.png" alt="Refresh" />
    </button>

    <button id="popout-chat-button" title="Popout Chat" style="display: none;">
      <img src="images/popout.png" alt="Popout" />
    </button>
  </div>
</div>

<style lang="scss">
  #header {
    display: flex;
    flex-shrink: 1;
    align-items: center;
    justify-content: space-between;

    padding: 5px 10px 0;
  }

  #logo {
    height: 30px;
    width: 30px;

    border-radius: 50%;
    margin-right: 10px;
  }

  .login-button {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
  }

  .buttons {
    display: flex;
    justify-content: space-between;
    gap: 10px;
  }

  .buttons :global {
    svg,
    img {
      height: 30px;
      width: 30px;
    }
  }

  #twitch-login-button {
    padding: 0;
    background: none;
    border: none;
    color: #bbb;
    font-size: 1em;
    cursor: pointer;

    transition: color 0.3s;
  }

  #logout-button {
    padding: 0;
    background: none;
    border: none;
    color: #bbb;
    font-size: 1em;
    cursor: pointer;

    transition: color 0.3s;
  }
</style>
