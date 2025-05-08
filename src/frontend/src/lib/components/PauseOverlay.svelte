<script lang="ts">
  let { newMessageCount }: { newMessageCount: number } = $props();
</script>

<div id="pauseOverlay">
  {#if newMessageCount === 0}
    <p>Chat Paused</p>
  {:else if newMessageCount === 1}
    <p>1 New Message</p>
  {:else}
    <p>{newMessageCount} New Messages</p>
  {/if}
</div>

<style lang="scss">
  #pauseOverlay {
    position: absolute;
    top: 80%;
    left: 50%;
    transform: translate(-50%, -80%);
    z-index: 1000;
    text-align: center;
    border: 1px solid #242729;
    border-radius: 8px;
    background-color: rgba(12, 15, 18, 0.8);
    font-weight: bold;
    padding: 8px;
  }
</style>
