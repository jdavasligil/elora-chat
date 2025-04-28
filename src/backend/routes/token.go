package routes

import (
	"bufio"
	"iter"
	"strings"
)

const (
	TokenTypeText    = "text"
	TokenTypeEmote   = "emote"
	TokenTypeColour  = "colour"
	TokenTypeEffect  = "effect"
	TokenTypePattern = "pattern"
)

const TextEffectSep = ":"

var TextEffects = map[string]struct{}{
	"wave":   {},
	"wave2":  {},
	"shake":  {},
	"slide":  {},
	"scroll": {},
}

var TextColours = map[string]struct{}{
	"yellow":  {},
	"red":     {},
	"green":   {},
	"cyan":    {},
	"purple":  {},
	"white":   {},
	"flash1":  {},
	"flash2":  {},
	"flash3":  {},
	"glow1":   {},
	"glow2":   {},
	"glow3":   {},
	"rainbow": {},
}

type Token struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emote *Emote `json:"emote"`
}

type Tokenizer struct {
	EmoteCache map[string]*Emote
}

// Recursively check for the presence of colors and effects
//
// Formats:
//
//	color:text
//	effect:text
//	color:effect:text
//	effect:colour:text
//
// Returns: (yield result, last word)
func (p Tokenizer) iterWordEffect(yield func(Token) bool, word string, depth int) (bool, string) {

	// Base Case: Empty string
	if word == "" {
		return true, ""
	}

	tok := Token{Type: TokenTypeText, Text: word}

	// Base Case: Emote
	if emote, ok := p.EmoteCache[word]; ok {
		tok.Type = TokenTypeEmote
		tok.Emote = emote
		return yield(tok), ""
	}

	// Base Case: Depth limit
	if depth == 2 {
		return true, word
	}

	prefix, postfix, sepFound := strings.Cut(word, TextEffectSep)

	if !sepFound || prefix == "" {
		return true, word
	}

	if _, ok := TextColours[prefix]; ok {
		tok.Type = TokenTypeColour
		tok.Text = prefix
	} else if _, ok := TextEffects[prefix]; ok {
		tok.Type = TokenTypeEffect
		tok.Text = prefix
	} else if 7 <= len(prefix) && len(prefix) <= 15 && prefix[:7] == "pattern" {
		// Note: Len("pattern") = 7 and pattern opcode max length is 8.
		tok.Type = TokenTypePattern
		tok.Text = prefix[7:]
	} else {
		return true, word
	}

	if !yield(tok) {
		return false, ""
	}

	// Recursively tokenize next effect
	return p.iterWordEffect(yield, postfix, depth+1)
}

// Returns an iterator over the string which yields tokens.
func (p Tokenizer) Iter(s string) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		var sb strings.Builder
		scanner := bufio.NewScanner(strings.NewReader(s))
		scanner.Split(bufio.ScanWords)

		ok := scanner.Scan()
		if !ok {
			return
		}

		word := scanner.Text()

		// Recursively tokenize text effects
		cont, lastWord := p.iterWordEffect(yield, word, 0)
		if !cont {
			return
		}
		sb.WriteString(lastWord)
		sb.WriteByte(' ')

		tok := Token{Type: TokenTypeText}

		// Scan the rest of the message for emotes
		for scanner.Scan() {
			word := scanner.Text()

			// Check for emote
			if emote, ok := p.EmoteCache[word]; ok {
				// yield text before emote
				tok.Text = strings.TrimSpace(sb.String())
				if tok.Text != "" {
					if !yield(tok) {
						return
					}
				}
				tok.Type = TokenTypeEmote
				tok.Text = word
				tok.Emote = emote
				if !yield(tok) {
					return
				}
				sb.Reset()
			} else {
				tok.Type = TokenTypeText
				tok.Emote = nil
				sb.WriteString(word)
				sb.WriteByte(' ')
			}
		}

		tok.Text = strings.TrimSpace(sb.String())

		// yield remaining text at end of message scan
		if tok.Text != "" {
			if !yield(tok) {
				return
			}
		}
	}
}
