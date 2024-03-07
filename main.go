package main

import (
    "log"
    "net/http"

    socketio "github.com/googollee/go-socket.io"
    "github.com/gorilla/handlers"
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

    server.OnDisconnect("/", func(s socketio.Conn, reason string) {
        log.Println("closed", reason)
        delete(connectedUsers, s.ID())
        for _, conn := range connectedUsers {
            conn.Emit("user_disconnected", s.ID())
        }
    })

    server.OnEvent("/chat", "msg", func(s socketio.Conn, msg Message) {
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

    server.OnError("/", func(s socketio.Conn, e error) {
        log.Println("meet error:", e)
    })

    go func() {
        if err := server.Serve(); err != nil {
            log.Fatalf("socketio listen error: %s\n", err)
        }
    }()
    defer server.Close()

    corsHandler := handlers.CORS(
        handlers.AllowedOrigins([]string{"*"}), // In production, replace "*" with your actual origins
        handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS"}),
        handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
    )

    http.Handle("/socket.io/", corsHandler(server))
    http.Handle("/", http.FileServer(http.Dir("asset")))

    log.Println("Serving at localhost:8000...")
    port := "8000" // Define the port you want to use
    log.Fatal(http.ListenAndServe(":"+port, nil)) // Use nil as the handler to use the default mux
}
