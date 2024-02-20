package main

import (
	"log"
	"net/http"

	"github.com/hpwn/EloraChat/src/backend/routes" // Ensure this is the correct path to your routes package

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Set up WebSocket chat routes
	routes.SetupChatRoutes(r)

	// Serve static files from the "public" directory
	fs := http.FileServer(http.Dir("public"))
	r.PathPrefix("/").Handler(http.StripPrefix("/", fs))

	// Start fetching chat messages
	chatURLs := []string{
		// "https://www.twitch.tv/hp_az",
		// "https://www.youtube.com/@hp_az/live",
		// "https://youtube.com/live/7NA555IYE24?feature=share",
		// "https://youtube.com/live/7455sVTXUPU?feature=share",
		"https://www.youtube.com/watch?v=jfKfPfyJRdk", // lofi girl live
		// "https://www.youtube.com/watch?v=39VeO9p7Vn0", // dayo live test stream
		// "https://www.youtube.com/live/6sjf7R0o-ss?si=WkdXIOu83_7Sglk2&t=7500", // ludwig live test stream
		// "https://www.twitch.tv/Johnstone",
		"https://www.twitch.tv/Hypnoshark",
		// "https://www.twitch.tv/QTCinderella",
		"https://www.twitch.tv/Quin69",
		// "https://www.twitch.tv/jakenbakeLIVE",
		"https://www.twitch.tv/Knut",
		// "https://www.youtube.com/@dayoman/live",
		// "https://www.twitch.tv/dayoman",
		// "https://www.twitch.tv/forsen", // basically a hard code for constant chats
		// "https://www.twitch.tv/jynxzi", // basically a hard code for constant chats
		// "https://www.youtube.com/watch?v=Gtqw9b8g2wk",
		// "https://www.twitch.tv/abel",
		// "https://www.youtube.com/watch?v=VebOqD00Zj8",
		// "https://www.youtube.com/watch?v=pCTxDYFdEOk",
		// "https://youtube.com/live/JHqR9hq70No?feature=share",
		"https://www.twitch.tv/papaplatte",
		"https://www.youtube.com/watch?v=REmPV-EPwPc", // ninja yt
	}
	routes.StartChatFetch(chatURLs)

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal("Server start error:", err)
	}
}
