package ble

import (
	"fmt"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"tinygo.org/x/bluetooth"
)

// Standard BLE UUIDs
var (
	// Service UUIDs
	ServiceCyclingPower = bluetooth.ServiceUUIDCyclingPower
	ServiceHeartRate    = bluetooth.ServiceUUIDHeartRate // 0x180D

	// Characteristic UUIDs
	CharCyclingPowerMeasure = bluetooth.CharacteristicUUIDCyclingPowerMeasurement
	CharHeartRateMeasure    = bluetooth.CharacteristicUUIDHeartRateMeasurement // 0x2A37
)

var (
	trainerDevice *bluetooth.Device
	hrDevice      *bluetooth.Device
)

// StartScanner starts the BLE discovery process for both Trainer and HRM.
func StartScanner() {
	fmt.Println("[BLE] Scanner Initialized. Looking for Power Meters and Heart Rate Monitors...")

	err := Adapter.Enable()
	if err != nil {
		fmt.Printf("[BLE] FATAL: Failed to enable BLE adapter: %v\n", err)
		return
	}

	// Scan loop
	for {
		// Only scan if we are missing at least one device
		if trainerDevice != nil && hrDevice != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Println("[BLE] Scanning...")
		
		// Blocking scan (runs until a device is found or manually stopped)
		err := Adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			
			// 1. Check for Trainer (Power Meter)
			if trainerDevice == nil && result.HasServiceUUID(ServiceCyclingPower) {
				fmt.Printf("[BLE] Found Power Meter: %s (%s)\n", result.LocalName(), result.Address.String())
				adapter.StopScan()
				connectTrainer(result.Address)
			}

			// 2. Check for Heart Rate Monitor
			if hrDevice == nil && result.HasServiceUUID(ServiceHeartRate) {
				fmt.Printf("[BLE] Found HR Monitor: %s (%s)\n", result.LocalName(), result.Address.String())
				adapter.StopScan()
				connectHR(result.Address)
			}
		})

		if err != nil {
			fmt.Printf("[BLE] Scan error: %v. Retrying in 2s...\n", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func connectTrainer(addr bluetooth.Address) {
	fmt.Println("[BLE] Connecting to Trainer...")
	device, err := Adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("[BLE] Connection failed: %v\n", err)
		return
	}

	trainerDevice = &device
	fmt.Println("[BLE] Trainer Connected! Discovering Services...")

	// Discover and Subscribe
	services, _ := device.DiscoverServices([]bluetooth.UUID{ServiceCyclingPower})
	for _, service := range services {
		chars, _ := service.DiscoverCharacteristics([]bluetooth.UUID{CharCyclingPowerMeasure})
		for _, char := range chars {
			fmt.Println("[BLE] Subscribing to Power Measurement...")
			char.EnableNotifications(func(buf []byte) {
				// Parse Standard Cycling Power
				power, cadence := parsePowerCadence(buf)

				app.State.Lock()
				app.State.RealPower = power
				app.State.RealCadence = cadence
				app.State.ConnectedReal = true
				app.State.Unlock()
			})
		}
	}
}

func connectHR(addr bluetooth.Address) {
	fmt.Println("[BLE] Connecting to HR Monitor...")
	device, err := Adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("[BLE] HR Connection failed: %v\n", err)
		return
	}

	hrDevice = &device
	fmt.Println("[BLE] HR Monitor Connected! Discovering Services...")

	// Discover and Subscribe
	services, _ := device.DiscoverServices([]bluetooth.UUID{ServiceHeartRate})
	for _, service := range services {
		chars, _ := service.DiscoverCharacteristics([]bluetooth.UUID{CharHeartRateMeasure})
		for _, char := range chars {
			fmt.Println("[BLE] Subscribing to Heart Rate...")
			char.EnableNotifications(func(buf []byte) {
				// Parse Standard Heart Rate
				hr := parseHR(buf)

				app.State.Lock()
				app.State.RealHR = hr
				app.State.ConnectedHR = true
				app.State.Unlock()
			})
		}
	}
}

// Helper: Parse Standard Cycling Power (0x2A63)
func parsePowerCadence(buf []byte) (int, int) {
	if len(buf) < 4 {
		return 0, 0
	}

	flags := uint16(buf[0]) | uint16(buf[1])<<8
	power := int(int16(uint16(buf[2]) | uint16(buf[3])<<8))

	// Calculate offset based on flags
	offset := 4
	if flags&0x01 != 0 { offset += 1 } // Pedal Power Balance present
	if flags&0x04 != 0 { offset += 2 } // Accumulated Torque present
	if flags&0x10 != 0 { offset += 6 } // Wheel Revolution Data present

	cadence := 0
	// Check if Crank Revolution Data is present (Bit 5)
	if flags&0x20 != 0 && len(buf) >= offset+4 {
		// We calculate RPM in the logic loop or here. 
		// For simplicity, let's assume the trainer sends instantaneous cadence 
		// or we just capture the raw bytes. 
		// NOTE: Calculating cadence from Cumulative Crank Revs requires storing previous state.
		// For this snippet, we will trust the main loop to handle zeroing, 
		// or implement a simple calculator if needed.
		// Usually, trainers send estimated instantaneous cadence in a separate field or via FTMS.
		// If this is raw CPP, we strictly need previous revs.
		// Let's return 0 here and rely on the logic loop fix for now, 
		// or use a simplified assumption if your trainer supports it.
		
		// To properly fix "Ghost Cadence", we act in the Logic Loop.
	}
	
	// FIX: Ensure no negative power
	if power < 0 { power = 0 }
	
	return power, cadence
}

// Helper: Parse Standard Heart Rate (0x2A37)
func parseHR(buf []byte) int {
	if len(buf) < 2 {
		return 0
	}
	
	flags := buf[0]
	hr := 0
	
	// Bit 0: Value Format (0 = UINT8, 1 = UINT16)
	if flags&0x01 == 0 {
		hr = int(buf[1])
	} else {
		if len(buf) >= 3 {
			hr = int(uint16(buf[1]) | uint16(buf[2])<<8)
		}
	}

	return hr
}

// DisconnectTrainer gracefully closes connections
func DisconnectTrainer() {
	if trainerDevice != nil {
		(*trainerDevice).Disconnect()
		trainerDevice = nil
	}
	if hrDevice != nil {
		(*hrDevice).Disconnect()
		hrDevice = nil
	}
	
	app.State.Lock()
	app.State.ConnectedReal = false
	app.State.ConnectedHR = false
	app.State.Unlock()
	
	fmt.Println("[BLE] Devices Disconnected.")
}