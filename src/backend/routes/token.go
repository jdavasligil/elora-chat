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

var TextEffects = map[string]struct{}{
	"wave":   {},
	"wave2":  {},
	"shake":  {},
	"slide":  {},
	"scroll": {},
}

var TextColours = map[string]struct{}{
	"yellow": {},
	"red":    {},
	"green":  {},
	"cyan":   {},
	"purple": {},
	"white":  {},
	"flash1": {},
	"flash2": {},
	"flash3": {},
	"glow1":  {},
	"glow2":  {},
	"glow3":  {},
	"rainbow":  {},
	"pattern":  {},
}

type Token struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emote *Emote `json:"emote"`
}

type Tokenizer struct {
	EmoteCache map[string]Emote
}

// Returns an iterator over the string which yields tokens.
func (p Tokenizer) Iter(s string) iter.Seq[Token] {
	return func(yield func(Token) bool) {
		scanner := bufio.NewScanner(strings.NewReader(s))
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			word := scanner.Text()
			tok := Token{Type: TokenTypeText, Text: word}

			if emote, ok := p.EmoteCache[word]; ok {
				tok.Type = TokenTypeEmote
				tok.Emote = &emote
			} else if word[len(word)-1] == ':' {
				text := word[:len(word)-1]
				if _, ok := TextEffects[text]; ok {
					tok.Type = TokenTypeEffect
					tok.Text = text
				} else if _, ok := TextColours[text]; ok {
					tok.Type = TokenTypeColour
					tok.Text = text
				}
			}
			if !yield(tok) {
				return
			}
		}
	}
}
