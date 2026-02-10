package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"github.com/SoNdA11/argus-recon/internal/ble"
	"github.com/SoNdA11/argus-recon/internal/server"
)

func main() {
	fmt.Println("\n▒▒▒▒▒▒▒ ARGUS RECON ▒▒▒▒▒▒▒")

	ble.InitAdapter()
	ble.SetupServices()

	go ble.StartScanner()

	go logicLoop()

	fmt.Println("[SYSTEM] Core Logic Activated.")
	fmt.Println("[SYSTEM] Starting Web Server...")

	server.Start()
}

func logicLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		app.State.Lock()

		noise := rand.Intn(6) - 3

		if app.State.Mode == "sim" {
			base := app.State.SimBasePower
			if base == 0 {
				base = 150
			}
			
			app.State.OutputPower = base + noise
			app.State.OutputCadence = 90 + (noise / 2)
			app.State.OutputHR = 70 + (app.State.OutputPower / 3)

		} else {
			// --- BRIDGE MODE ---
			var boostWatts int

			if app.State.BoostType == "pct" {
				fReal := float64(app.State.RealPower)
				fPct := float64(app.State.BoostValue)
				boostWatts = int(fReal * (fPct / 100.0))
			} else {
				boostWatts = app.State.BoostValue
			}

			finalPower := app.State.RealPower + boostWatts

			if app.State.RealPower == 0 {
				finalPower = 0
			}

			app.State.OutputPower = finalPower
			app.State.OutputCadence = app.State.RealCadence
			app.State.OutputHR = 65 + (app.State.OutputPower / 3)
		}

		if app.State.OutputPower < 0 {
			app.State.OutputPower = 0
		}

		if app.State.OutputHR > 190 {
			app.State.OutputHR = 190
		}

		ble.UpdateOutputs(app.State.OutputPower, app.State.OutputCadence, app.State.OutputHR)

		app.State.Unlock()
	}
}