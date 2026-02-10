package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"github.com/SoNdA11/argus-recon/internal/ble"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func Start() {
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/template/index.html")
	})

	http.HandleFunc("/ws", handleWebSocket)

	fmt.Println("[HTTP] Online interface at: http://localhost:8080")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("[FATAL] Error starting server: %v\n", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	go func() {
		for {
			app.State.Lock()
			msg := map[string]interface{}{
				"mode":          app.State.Mode,
				"boostValue":    app.State.BoostValue,
				"boostType":     app.State.BoostType,
				"simBasePower":  app.State.SimBasePower,
				"realPower":     app.State.RealPower,
				"outputPower":   app.State.OutputPower,
				"connected":     app.State.ConnectedReal,
				"clientConn":    app.State.ClientConnected,
				"outputHR":      app.State.OutputHR,
			}
			app.State.Unlock()
			
			if err := conn.WriteJSON(msg); err != nil {
				return 
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	for {
		var cmd map[string]interface{}
		if err := conn.ReadJSON(&cmd); err != nil {
			break
		}

		app.State.Lock()
		if m, ok := cmd["mode"].(string); ok { app.State.Mode = m }
		if b, ok := cmd["boost"].(float64); ok { app.State.BoostValue = int(b) }
		if s, ok := cmd["sim"].(float64); ok { app.State.SimBasePower = int(s) }
		if t, ok := cmd["boostType"].(string); ok { 
			app.State.BoostType = t 
			app.State.BoostValue = 0 
		}
		app.State.Unlock()

		if d, ok := cmd["disconnect"].(bool); ok && d {
			go ble.DisconnectTrainer()
		}
	}
}