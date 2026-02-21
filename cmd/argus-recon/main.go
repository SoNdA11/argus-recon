package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"github.com/SoNdA11/argus-recon/internal/ble"
	"github.com/SoNdA11/argus-recon/internal/integrity"
	"github.com/SoNdA11/argus-recon/internal/server"
)

func main() {
	fmt.Println("\n▒▒▒▒▒▒▒ ARGUS RECON ▒▒▒▒▒▒▒")

	ble.InitAdapter()
	ble.SetupServices()

	go ble.StartScanner()
	go integrity.Start()

	go logicLoop()

	fmt.Println("[SYSTEM] Core Logic Activated.")
	fmt.Println("[SYSTEM] Starting Web Server...")

	server.Start()
}

func logicLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		app.State.Lock()

		// Noise generator
		noise := rand.Intn(6) - 3

		if app.State.Mode == "sim" {
			// --- SIMULATOR MODE ---
			base := app.State.SimBasePower
			if base == 0 {
				base = 150
			}

			app.State.OutputPower = base + noise

			// Cadence Logic for Sim
			if app.State.OutputPower > 0 {
				app.State.OutputCadence = 90 + (noise / 2)
			} else {
				app.State.OutputCadence = 0
			}

			// HR Logic for Sim (Fallback if no Real HR)
			if app.State.RealHR > 0 {
				app.State.OutputHR = app.State.RealHR
			} else {
				app.State.OutputHR = 70 + (app.State.OutputPower / 3)
			}

		} else {
			// --- BRIDGE MODE (INTERCEPTOR) ---

			// 1. Calculate Boost
			var boostWatts int
			if app.State.BoostType == "pct" {
				fReal := float64(app.State.RealPower)
				fPct := float64(app.State.BoostValue)
				boostWatts = int(fReal * (fPct / 100.0))
			} else {
				boostWatts = app.State.BoostValue
			}

			finalPower := app.State.RealPower + boostWatts

			// Safety: If real trainer stops, zero the output
			if app.State.RealPower == 0 {
				finalPower = 0
			}

			app.State.OutputPower = finalPower

			if app.State.RealPower == 0 {
				app.State.OutputCadence = 0
			} else {
				if app.State.RealCadence > 0 {
					app.State.OutputCadence = app.State.RealCadence
				} else {
					// Fallback estimation: 60rpm + (Watts/5) capped at 100
					estimated := 60 + (app.State.RealPower / 5)
					if estimated > 100 {
						estimated = 100
					}
					app.State.OutputCadence = estimated
				}
			}

			// Use Real HR from sensor if available.
			if app.State.RealHR > 0 {
				app.State.OutputHR = app.State.RealHR
			} else {
				// Fallback to simulation ONLY if sensor is missing
				app.State.OutputHR = 65 + (app.State.OutputPower / 3)
			}
		}

		// Security Limits
		if app.State.OutputPower < 0 {
			app.State.OutputPower = 0
		}
		if app.State.OutputHR > 190 {
			app.State.OutputHR = 190
		}

		// Send to BLE
		ble.UpdateOutputs(app.State.OutputPower, app.State.OutputCadence, app.State.OutputHR)

		app.State.Unlock()
	}
}