package main

import (
"context"
"encoding/json"
"flag"
"fmt"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/google/uuid"
"github.com/gorilla/websocket"
"go.uber.org/zap"

"messenger-backend/internal/database"
"messenger-backend/internal/websocket"
)

var upgrader = websocket.Upgrader{
ReadBufferSize:  1024,
WriteBufferSize: 1024,
CheckOrigin: func(r *http.Request) bool {
return true
},
}

func main() {
port := flag.Int("port", 8080, "Server port")
flag.Parse()

logger, _ := zap.NewProduction()
defer logger.Sync()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

dbConfig := database.Config{
Host:            getEnv("DB_HOST", "localhost"),
Port:            getIntEnv("DB_PORT", 5432),
User:            getEnv("DB_USER", "messenger"),
Password:        getEnv("DB_PASSWORD", ""),
DBName:          getEnv("DB_NAME", "messenger"),
SSLMode:         getEnv("DB_SSLMODE", "disable"),
MaxOpenConns:    25,
MaxIdleConns:    5,
ConnMaxLifetime: 5 * time.Minute,
}

db, err := database.NewDatabase(ctx, dbConfig, logger)
if err != nil {
logger.Fatal("DB init failed", zap.Error(err))
}
defer db.Close()

hub := ws.NewHub(logger)
go hub.Run(ctx)

mux := http.NewServeMux()
mux.HandleFunc("/health", healthHandler)
mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
handleWS(w, r, hub, logger)
})

server := &http.Server{
Addr:         fmt.Sprintf(":%d", *port),
Handler:      mux,
ReadTimeout:  15 * time.Second,
WriteTimeout: 15 * time.Second,
}

go server.ListenAndServe()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

server.Shutdown(context.Background())
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
w.WriteHeader(http.StatusOK)
w.Write([]byte("OK"))
}

func handleWS(w http.ResponseWriter, r *http.Request, hub *ws.Hub, logger *zap.Logger) {
conn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
return
}

client := &ws.Client{
ID:     uuid.New().String(),
UserID: r.URL.Query().Get("user_id"),
Conn:   conn,
Send:   make(chan []byte, 256),
}

hub.Register <- client
go client.WritePump()
go client.ReadPump(hub, func(c *ws.Client, m ws.WSMessage) error {
return c.SendJSON(ws.WSMessage{Type: ws.MessageTypeAck, Timestamp: time.Now().UnixMilli()})
})
}

func getEnv(key, def string) string {
if v := os.Getenv(key); v != "" {
return v
}
return def
}

func getIntEnv(key string, def int) int {
if v := os.Getenv(key); v != "" {
fmt.Sscanf(v, "%d", &def)
}
return def
}
