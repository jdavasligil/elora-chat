package routes

import (
	"reflect"
	"testing"
)

func TestTokenizer(t *testing.T) {
	tokenizer := Tokenizer{
		EmoteCache: map[string]*Emote{
			"Clap": {
				ID:     "1",
				Name:   "Clap",
				Images: []Image{{URL: "https://cdn.test.net/v1/emotes/1/1x.webp"}},
			},
			"Clap2": {
				ID:     "2",
				Name:   "Clap2",
				Images: []Image{{URL: "https://cdn.test.net/v1/emotes/2/1x.webp"}},
			},
			"KEKW": {
				ID:     "3",
				Name:   "KEKW",
				Images: []Image{{URL: "https://cdn.test.net/v1/emotes/3/1x.webp"}},
			},
			"FeelsGoodMan": {
				ID:     "4",
				Name:   "FeelsGoodMan",
				Images: []Image{{URL: "https://cdn.test.net/v1/emotes/4/1x.webp"}},
			},
		},
	}
	tests := []struct {
		Message  string
		Expected []Token
	}{
		{ // Gibberish with leading whitespace
			Message: "  2[qrp]3-4t[ #(YT$ jd  ",
			Expected: []Token{
				{TokenTypeText, "2[qrp]3-4t[ #(YT$ jd", nil},
			},
		},
		{ // Emotes
			Message: "KEKW KEKW FeelsGoodMan !!!",
			Expected: []Token{
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
				{TokenTypeEmote, "FeelsGoodMan", tokenizer.EmoteCache["FeelsGoodMan"]},
				{TokenTypeText, "!!!", nil},
			},
		},
		{ // Semicolons all over
			Message: ":jaja::#:!@#:",
			Expected: []Token{
				{TokenTypeText, ":jaja::#:!@#:", nil},
			},
		},
		{ // colour:text
			Message: "white:text",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
				{TokenTypeText, "text", nil},
			},
		},
		{ // colour:<WS>text
			Message: "white: text",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
				{TokenTypeText, "text", nil},
			},
		},
		{ // colour:effect:emote
			Message: "rainbow:wave2:KEKW",
			Expected: []Token{
				{TokenTypeColour, "rainbow", nil},
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
			},
		},
		{ // effect:color:emote
			Message: "wave2:rainbow:KEKW",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeColour, "rainbow", nil},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
			},
		},
		{ // effect:Emote:Emote
			Message: "wave2:KEKW:KEKW",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeText, "KEKW:KEKW", nil},
			},
		},
		{ // leading sep
			Message: ":cyan:text",
			Expected: []Token{
				{TokenTypeText, ":cyan:text", nil},
			},
		},
		{ // MANY SEP
			Message: ":::::::::",
			Expected: []Token{
				{TokenTypeText, ":::::::::", nil},
			},
		},
		{ // effect:MANY SEP
			Message: "wave2:::::::::",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeText, "::::::::", nil},
			},
		},
		{ // pattern
			Message: "patternq3q3q3q3:I am a bumblebee!!!",
			Expected: []Token{
				{TokenTypePattern, "q3q3q3q3", nil},
				{TokenTypeText, "I am a bumblebee!!!", nil},
			},
		},
	}
	for count, test := range tests {
		iterator := tokenizer.Iter(test.Message)
		tokens := []Token{}
		for tok := range iterator {
			tokens = append(tokens, tok)
		}
		if !reflect.DeepEqual(tokens, test.Expected) {
			t.Logf("Test [%d] FAILED:\n\nExpected: %v\nGot:      %v\n\n", count, test.Expected, tokens)
			t.Fail()
		}
	}
}
