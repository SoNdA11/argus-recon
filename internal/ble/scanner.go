package ble

import (
	"encoding/binary"
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"

	"github.com/SoNdA11/argus-recon/internal/app"
)

var currentDevice *bluetooth.Device

func StartScanner() {
	for {
		app.State.Lock()

		needScan := (app.State.Mode == "bridge") && !app.State.ConnectedReal
		app.State.Unlock()

		if needScan {
			go Adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
				if device.LocalName() != "Argus Recon" && device.LocalName() != "" {
					if device.HasServiceUUID(bluetooth.ServiceUUIDCyclingPower) {
						fmt.Printf("🎯 [SCANNER] Found Target: %s. Connecting...\n", device.LocalName())
						adapter.StopScan()
						connectToTrainer(device.Address)
					}
				}
			})
			time.Sleep(10 * time.Second)
			Adapter.StopScan()
		}
		time.Sleep(2 * time.Second)
	}
}

func connectToTrainer(addr bluetooth.Address) {
	device, err := Adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("❌ [ERROR] Connection Failed: %v\n", err)
		return
	}

	currentDevice = &device

	fmt.Println("🔗 [SYSTEM] BRIDGE ESTABLISHED: Real Trainer Linked.")

	app.State.Lock()
	app.State.ConnectedReal = true
	app.State.Unlock()

	go func() {
	}()

	srvs, _ := device.DiscoverServices([]bluetooth.UUID{bluetooth.ServiceUUIDCyclingPower})
	for _, srv := range srvs {
		chars, _ := srv.DiscoverCharacteristics([]bluetooth.UUID{bluetooth.CharacteristicUUIDCyclingPowerMeasurement})
		for _, char := range chars {
			char.EnableNotifications(func(buf []byte) {
				if len(buf) >= 4 {
					rawPower := int(binary.LittleEndian.Uint16(buf[2:4]))
					app.State.Lock()
					app.State.RealPower = rawPower
					if app.State.RealCadence == 0 {
						app.State.RealCadence = 85
					}
					app.State.Unlock()
				}
			})
		}
	}
}

func DisconnectTrainer() {
	if currentDevice != nil {
		fmt.Println("🔌 [SYSTEM] Disconnecting Real Trainer...")
		err := currentDevice.Disconnect()
		if err != nil {
			fmt.Printf("⚠️ Error while disconnecting (may already be closed): %v\n", err)
		}
		currentDevice = nil
	}

	app.State.Lock()
	app.State.ConnectedReal = false
	app.State.RealPower = 0
	app.State.Mode = "sim"
	app.State.Unlock()

	fmt.Println("✅ [SYSTEM] System returned to Simulator mode.")
}