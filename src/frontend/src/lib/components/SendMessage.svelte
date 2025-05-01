<div id="message-send-container">
  <input id="message-input" type="text" placeholder="Type a message..." />
  <button id="send-message-button">send</button>
</div>

<style lang="scss">
  #message-send-container {
    display: flex;
    flex-shrink: 1;
    align-items: center;

    margin: 10px 5px 0 5px;
    padding-bottom: 10px;
  }

  #message-input {
    width: auto;
    height: 50px;
    flex-grow: 1;

    background: #333;
    color: white;
    font-size: 1.2em;
    border: 1px solid #666;
    border-radius: 5px;

    margin-right: 10px;
    padding: 10px;
  }

  #send-message-button {
    height: 50px;
    flex-shrink: 0;

    background-color: #444;
    color: #bbb;
    font-size: 1em;
    border: 1px solid #666;
    border-radius: 5px;
    cursor: pointer;

    padding: 10px 20px;

    transition: background-color 0.3s;
  }
</style>
