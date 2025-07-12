package routes

import "strings"

// Names of colours known to and defined by the frontend.
var NameColours = map[string]struct{}{
	"red":       {},
	"orange":    {},
	"yellow":    {},
	"green":     {},
	"lightblue": {},
	"blue":      {},
	"violet":    {},
	"pink":      {},
	"tan":       {},
	"olive":     {},
	"lime":      {},
	"sky":       {},
	"purple":    {},
}

// Creates a response message directly from EloraChat for all to see.
func CreateResponse(r string) Message {
	var m Message

	m.Author = "EloraChat"
	m.Colour = "#FFCD05"
	m.Message = r
	m.Tokens = []Token{{
		Type: TokenTypeText,
		Text: r,
	}}

	return m
}

// Processes commands from message, potentially transforming the message.
// TODO: Replace userSetting with database ptr and update table instead of map.
func ProcessCommand(m Message, userSetting map[string]string) (Message, error) {
	cmd, optsStr, _ := strings.Cut(m.Tokens[0].Text, " ")
	opts := strings.Split(optsStr, " ")

	switch cmd {
	case "color":
		fallthrough
	case "colour":
		if len(opts) == 0 {
			// Error? Print help message? Do nothing for now.
			break
		}
		colour := opts[0]
		if _, ok := NameColours[colour]; ok {
			userSetting[m.Author] = colour
		}
	}

	return m, nil
}
