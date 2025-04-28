package routes

import (
	"bufio"
	"iter"
	"strings"
)

const (
	TokenTypeText   = "text"
	TokenTypeEmote  = "emote"
	TokenTypeColour = "colour"
	TokenTypeEffect = "effect"
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
	"pattern": {},
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
func (p Tokenizer) iterWordEffect(yield func(Token) bool, word string, depth int) {

	// Base Case: Empty string
	if word == "" {
		return
	}

	tok := Token{Type: TokenTypeText, Text: word}

	// Base Case: Emote
	if emote, ok := p.EmoteCache[word]; ok {
		tok.Type = TokenTypeEmote
		tok.Emote = emote
		yield(tok)
		return
	}

	// Base Case: Depth limit
	if depth == 2 {
		yield(tok)
		return
	}

	prefix, postfix, sepFound := strings.Cut(word, TextEffectSep)

	if !sepFound || prefix == "" {
		yield(tok)
		return
	}

	if _, ok := TextColours[prefix]; ok {
		tok.Type = TokenTypeColour
	} else if _, ok := TextEffects[prefix]; ok {
		tok.Type = TokenTypeEffect
	} else {
		yield(tok)
		return
	}

	tok.Text = prefix
	yield(tok)

	// Recursively tokenize next effect
	p.iterWordEffect(yield, postfix, depth+1)
}

// Returns an iterator over the string which yields tokens.
func (p Tokenizer) Iter(s string) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		scanner := bufio.NewScanner(strings.NewReader(s))
		scanner.Split(bufio.ScanWords)

		// Check first word for color or effects
		ok := scanner.Scan()
		if !ok {
			return
		}
		word := scanner.Text()

		// Recursively tokenize text effects
		p.iterWordEffect(yield, word, 0)

		// Scan the rest of the message for emotes
		for scanner.Scan() {
			word := scanner.Text()

			// Assume word is text
			tok := Token{Type: TokenTypeText, Text: word}

			// Check for emote
			if emote, ok := p.EmoteCache[word]; ok {
				tok.Type = TokenTypeEmote
				tok.Emote = emote
			}

			if !yield(tok) {
				return
			}
		}
	}
}
