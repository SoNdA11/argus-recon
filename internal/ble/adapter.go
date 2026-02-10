package ble

import (
	"github.com/SoNdA11/argus-recon/internal/app"
	"fmt"
	"log"
	"tinygo.org/x/bluetooth"
)

var Adapter = bluetooth.DefaultAdapter

func InitAdapter() {
	fmt.Println("[BLE] Initializing Hardware Stack...")

	if err := Adapter.Enable(); err != nil {
		fmt.Printf("\n[FATAL ERROR] Bluetooth Adapter Fault: %v\n", err)
		log.Fatal("System Halted.")
	}

	Adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		app.State.Lock()
		wasConnected := app.State.ClientConnected
		app.State.ClientConnected = connected
		app.State.Unlock()

		if connected {
			fmt.Println("\n✅ [BLE] UPLINK ESTABLISHED: Client Connected!")
		} else {
			if wasConnected {
				fmt.Println("🔻 [BLE] Uplink Lost: Client Disconnected.")
			}
		}
	})
}