package client

import (
	"fmt"

	irc "github.com/thoj/go-ircevent"
)

func StartChatClient(username, token string) {
	con := irc.IRC(username, username)
	con.Password = fmt.Sprintf("oauth:%s", token)
	con.UseTLS = true
	con.Debug = true

	con.AddCallback("001", func(e *irc.Event) {
		con.Join("#" + username)
	})

	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		fmt.Printf("Message from %s: %s\n", e.Nick, e.Message())
	})

	err := con.Connect("irc.chat.twitch.tv:6697")
	if err != nil {
		fmt.Printf("Failed to connect to IRC: %s\n", err)
		return
	}

	con.Loop()
}
