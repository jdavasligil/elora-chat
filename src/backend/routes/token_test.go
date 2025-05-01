package routes

import (
	"iter"
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
	type Test struct {
		Name     string
		Message  string
		Expected []Token
	}
	// Test effect parsing regardless of whitespace conditions
	effectTests := []Test{
		{
			Name:    "colourEmpty",
			Message: "white:",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
			},
		},
		{
			Name:    "<WS>colourEmpty",
			Message: "    white:",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
			},
		},
		{
			Name:    "effectEmpty",
			Message: "wave2:",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
			},
		},
		{
			Name:    "colour:text",
			Message: "white:text",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
				{TokenTypeText, "text", nil},
			},
		},
		{
			Name:    "color:<WS>text",
			Message: "white:  text",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
				{TokenTypeText, "text", nil},
			},
		},
		{
			Name:    "color:<WS>emote",
			Message: "white:  Clap2",
			Expected: []Token{
				{TokenTypeColour, "white", nil},
				{TokenTypeEmote, "Clap2", tokenizer.EmoteCache["Clap2"]},
			},
		},
		{
			Name:    "color:effect:emote",
			Message: "rainbow:wave2:KEKW",
			Expected: []Token{
				{TokenTypeColour, "rainbow", nil},
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
			},
		},
		{
			Name:    "effect:color:emote",
			Message: "wave2:rainbow:KEKW",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeColour, "rainbow", nil},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
			},
		},
		{
			Name:    "effect:color:<WS>emote",
			Message: "wave2:rainbow:   KEKW",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeColour, "rainbow", nil},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
			},
		},
		{
			Name:    "effect:emote:emote",
			Message: "wave2:KEKW:KEKW",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeText, "KEKW:KEKW", nil},
			},
		},
		{
			Name:    "leadingSep",
			Message: ":cyan:text",
			Expected: []Token{
				{TokenTypeText, ":cyan:text", nil},
			},
		},
		{
			Name:    "manySep",
			Message: ":::::::::",
			Expected: []Token{
				{TokenTypeText, ":::::::::", nil},
			},
		},
		{
			Name:    "effect:manySep",
			Message: "wave2:::::::::",
			Expected: []Token{
				{TokenTypeEffect, "wave2", nil},
				{TokenTypeText, "::::::::", nil},
			},
		},
		{
			Name:    "semicolons",
			Message: ":jaja::#:!@#:",
			Expected: []Token{
				{TokenTypeText, ":jaja::#:!@#:", nil},
			},
		},
		{
			Name:    "semicolonEmpty",
			Message: ":   ",
			Expected: []Token{
				{TokenTypeText, ":", nil},
			},
		},
		{
			Name:    "patternNoOps",
			Message: "pattern:I am a bumblebee!!!",
			Expected: []Token{
				{TokenTypeText, "I am a bumblebee!!!", nil},
			},
		},
		{
			Name:    "patternEmpty",
			Message: "pattern:",
			Expected: []Token{
			},
		},
		{
			Name:    "patternMax",
			Message: "patternq3q3q3q3:I am a bumblebee!!!",
			Expected: []Token{
				{TokenTypePattern, "q3q3q3q3", nil},
				{TokenTypeText, "I am a bumblebee!!!", nil},
			},
		},
	}
	iterTests := []Test{
		{
			Name:    "randomWS",
			Message: "  2[qrp]3-4t[    #(YT$ jd  ",
			Expected: []Token{
				{TokenTypeText, "2[qrp]3-4t[ #(YT$ jd", nil},
			},
		},
		{
			Name:    "emotesWS",
			Message: "KEKW KEKW    FeelsGoodMan  !!!",
			Expected: []Token{
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
				{TokenTypeEmote, "KEKW", tokenizer.EmoteCache["KEKW"]},
				{TokenTypeEmote, "FeelsGoodMan", tokenizer.EmoteCache["FeelsGoodMan"]},
				{TokenTypeText, "!!!", nil},
			},
		},
		{
			Name:    "emotesSmashed",
			Message: "KEKWKEKWFeelsGoodMan!",
			Expected: []Token{
				{TokenTypeText, "KEKWKEKWFeelsGoodMan!", nil},
			},
		},
		{
			Name:    "colonNoEffect",
			Message: "Hey, you guys know about Gunz: The Duel?",
			Expected: []Token{
				{TokenTypeText, "Hey, you guys know about Gunz: The Duel?", nil},
			},
		},
		{
			Name:    "colonEffectTypo",
			Message: "gren:This is green!",
			Expected: []Token{
				{TokenTypeText, "gren:This is green!", nil},
			},
		},
		{
			Name:    "emoteSolo",
			Message: "Clap",
			Expected: []Token{
				{TokenTypeEmote, "Clap", tokenizer.EmoteCache["Clap"]},
			},
		},
	}

	// Helper to run iterator tests
	RunIterTest := func(t *testing.T, iterator iter.Seq[Token], test Test) {
		tokens := []Token{}
		for tok := range iterator {
			tokens = append(tokens, tok)
		}
		if !reflect.DeepEqual(tokens, test.Expected) {
			t.Logf("\n\nMessage:  <%s>\nExpected: %v\nGot:      %v\n\n", test.Message, test.Expected, tokens)
			t.Fail()
		}
	}

	for _, test := range effectTests {
		t.Run("Iter-"+test.Name, func(t *testing.T) {
			RunIterTest(t, tokenizer.Iter(test.Message), test)
		})
	}
	for _, test := range iterTests {
		t.Run("Iter-"+test.Name, func(t *testing.T) {
			RunIterTest(t, tokenizer.Iter(test.Message), test)
		})
	}
}
