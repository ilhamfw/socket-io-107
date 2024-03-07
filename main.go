package main

import (
	"log"
	"net/http"
	"runtime/debug"

	socketio "github.com/googollee/go-socket.io"
)

type Message struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

func main() {
	server := socketio.NewServer(nil)
	var connectedUsers = make(map[string]socketio.Conn)

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		connectedUsers[s.ID()] = s
		log.Println("connected:", s.ID())
		// Broadcast the new user's ID to all connected users
		for _, conn := range connectedUsers {
			if conn.ID() != s.ID() {
				conn.Emit("user_connected", s.ID())
			}
		}
		return nil
	})

	// Panic recovery middleware
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Printf("recovering from panic, reason: %v", e)
		debug.PrintStack()
		s.Emit("error", "An unexpected error occurred")
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)
		// Remove the user from the connected users map
		delete(connectedUsers, s.ID())
		// Broadcast to all users that someone has disconnected
		for _, conn := range connectedUsers {
			conn.Emit("user_disconnected", s.ID())
		}
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg Message) {
		// Broadcast message to all clients except the sender
		for _, user := range connectedUsers {
			if user.ID() != s.ID() {
				user.Emit("msg", msg)
			}
		}
	})

	server.OnEvent("/chat", "bye", func(s socketio.Conn) {
		log.Println("bye from:", s.ID())
		s.Close()
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	http.Handle("/socket.io/", corsMiddleware(server))
    http.Handle("/", http.FileServer(http.Dir("asset")))

    log.Println("Serving at localhost:8000...")
    log.Fatal(http.ListenAndServe(":8000", nil))
}

// corsMiddleware wraps the Socket.IO server to handle CORS.
func corsMiddleware(next http.Handler) http.HandlerFunc {
    allowedOrigins := map[string]bool{
        "http://localhost:5173": true,
        "http://socket-io.107.jp": true, // Make sure the scheme (http/https) matches your client
    }
    
    return func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        
        // Check if the origin is in the allowed list
        if _, ok := allowedOrigins[origin]; ok {
            w.Header().Set("Access-Control-Allow-Origin", origin)
        }

        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

        // If it's a preflight request, stop here
        if r.Method == "OPTIONS" {
            return
        }

        next.ServeHTTP(w, r)
    }
}
