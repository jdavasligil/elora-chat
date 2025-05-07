import { TextEffect } from '$lib/types/effects';
import { FragmentType, type Emote, type Fragment } from '$lib/types/messages';
import { buildApiUrl } from './misc';

export function sanitizeMessage(message: string): string {
  // replace < and > with HTML entities to prevent XSS attacks
  return message.replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export function addMessageEffects(message: string): { messageText: string; effects: string } {
  const colors = [
    'yellow',
    'red',
    'green',
    'cyan',
    'purple',
    'pink',
    'rainbow',
    'glow1',
    'glow2',
    'glow3',
    'flash1',
    'flash2',
    'flash3'
  ];
  const colorCommands = colors.reduce(
    (accumulator, color) => ({
      ...accumulator,
      [color]: `color-${color}`
    }),
    {}
  );

  const commands = {
    ...colorCommands,
    bold: 'text-bold',
    italic: 'text-italic',
    wave: 'effect-wave',
    shake: 'effect-shake'
  };

  const lastCommandIndex = message.indexOf(': ');
  const effectNames = lastCommandIndex >= 0 ? message.substr(0, lastCommandIndex).split(':') : [];
  let messageText = lastCommandIndex >= 0 ? message.substr(lastCommandIndex + 2) : message;
  const effects = effectNames
    .map((effect) =>
      // eslint-disable-next-line no-prototype-builtins
      commands.hasOwnProperty(effect) ? commands[effect as keyof typeof commands] : null
    )
    .filter((value) => !!value)
    .join(' ');

  // if no effects were found, set the message text back to the original message content
  if (effects.length <= 0) {
    messageText = message;
  }

  return { messageText, effects };
}

function escapeRegExp(string: string): string {
  return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); // $& means the whole matched string
}

export function loadImage(source: string): string {
  return buildApiUrl(`/imageproxy?url=${encodeURIComponent(source)}`);
}

export function imageFromEmote(emote: Emote): HTMLImageElement {
      const emoteImg = document.createElement('img');
      emoteImg.className = 'emote-image';
      emoteImg.alt = emote.name;
      emoteImg.src = loadImage(emote.images[0].url);
      return emoteImg;
}

export function replaceEmotes(message: string, emotes: Emote[]): string {
  let newMessage = message;

  if (emotes && emotes.length > 0) {
    emotes.forEach((emote) => {
      const emoteImg = imageFromEmote(emote);
      const escapedEmoteName = escapeRegExp(emote.name);
      const emoteRegex = new RegExp(escapedEmoteName, 'g');
      newMessage = newMessage.replace(emoteRegex, emoteImg.outerHTML);
    });
  }

  return newMessage;
}

function* fragmentGenerator(fragments: Fragment[]): Generator<Fragment, Fragment, boolean> {
  for (const fragment of fragments) {
    yield fragment;
  }
  return { type: FragmentType.Text, text: "", emote: null };
}

// Formats message fragments into HTML and returns style classes.
// Parses chat effects with recursive descent following the OSRS wiki spec.
export function formatMessageFragments(fragments: Fragment[]): { messageWithHTML: string; effects: string } {
  const effectList: string[] = [];
  const messageList: string[] = [];
  const fragmentGen = fragmentGenerator(fragments)

  // Rule for handling text and emotes from the top level
  function handleTextEmote() {
    const nextFrag = fragmentGen.next();
    if (nextFrag.done) {
      return;
    }
    const fragment = nextFrag.value
    if (fragment.type === FragmentType.Emote && !!fragment.emote) {
      messageList.push(imageFromEmote(fragment.emote).outerHTML);
    } else {
      messageList.push(sanitizeMessage(fragment.text));
    }
    handleTextEmote();
  }

  // Rule for handling effects that require wrapping characters with span
  function handleSpanEffect() {
    const nextFrag = fragmentGen.next();
    if (nextFrag.done) {
      return;
    }
    const fragment = nextFrag.value
    if (fragment.type === FragmentType.Emote && !!fragment.emote) {
      messageList.push(imageFromEmote(fragment.emote).outerHTML);
    } else {

      messageList.push(sanitizeMessage(fragment.text).split('').map(c => `<span>${c}</span>`).join());
    }
    handleSpanEffect();
  }

  // Rule for handling effects
  function handleEffect(effect: string) {
    switch (effect) {
      case TextEffect.Wave:
      case TextEffect.Wave2:
      case TextEffect.Shake:
        handleSpanEffect();
        break;
      default:
        handleTextEmote();
        break;
    }
  }

  // Rule for handling color fragments
  function handleColor() {
    const nextFrag = fragmentGen.next();
    if (nextFrag.done) {
      return;
    }
    const fragment = nextFrag.value
    if (fragment.type === FragmentType.Effect) {
        effectList.push("effect-"+fragment.text);
        handleEffect(fragment.text);
    } else {
      if (fragment.type === FragmentType.Emote && !!fragment.emote) {
        messageList.push(imageFromEmote(fragment.emote).outerHTML);
      } else {
        messageList.push(sanitizeMessage(fragment.text));
      }
      handleTextEmote();
    }
  }

  // Top level entry point for recursive descent parsing
  function recursiveDescent() {
    const nextFrag = fragmentGen.next();
    if (nextFrag.done) {
      return;
    }
    const fragment = nextFrag.value
    switch (fragment.type) {
      case FragmentType.Text:
        messageList.push(sanitizeMessage(fragment.text));
        handleTextEmote();
        break;
      case FragmentType.Emote:
        if (fragment.emote) {
          messageList.push(imageFromEmote(fragment.emote).outerHTML);
          handleTextEmote();
          break;
        }
      case FragmentType.Colour:
        effectList.push("color-"+fragment.text);
        handleColor();
        break;
      case FragmentType.Effect:
        effectList.push("effect-"+fragment.text);
        handleEffect(fragment.text);
        break;
      case FragmentType.Pattern:
        // TODO: Handle custom patterns
        handleColor();
        break;
    }
  }

  // Recursively generate the HTML message and gather styles
  recursiveDescent();

  return { messageWithHTML: messageList.join(), effects: effectList.join(" ") };
}
