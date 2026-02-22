package ble

import (
	"fmt"
	"strings"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"github.com/SoNdA11/argus-recon/internal/integrity"
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
	trainerDevice     *bluetooth.Device
	hrDevice          *bluetooth.Device
	connectingTrainer bool
	connectingHR      bool
)

// StartScanner starts the BLE discovery process for both Trainer and HRM.
func StartScanner() {
	fmt.Println("[BLE] Scanner Initialized. Looking for Power Meters and Heart Rate Monitors...")

	err := Adapter.Enable()
	if err != nil {
		fmt.Printf("[BLE] FATAL: Failed to enable BLE adapter: %v\n", err)
		return
	}

	for {
		if shouldPauseScan() {
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Println("[BLE] Scanning...")
		err := Adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			registerScanResult(result)

			if trainerDevice == nil && !connectingTrainer && result.HasServiceUUID(ServiceCyclingPower) {
				connectingTrainer = true
				addr := result.Address
				name := result.LocalName()
				go func() {
					fmt.Printf("[BLE] Found Power Meter: %s (%s)\n", name, addr.String())
					adapter.StopScan()
					connectTrainer(addr)
					connectingTrainer = false
				}()
			}

			if hrDevice == nil && !connectingHR && result.HasServiceUUID(ServiceHeartRate) {
				connectingHR = true
				addr := result.Address
				name := result.LocalName()
				go func() {
					fmt.Printf("[BLE] Found HR Monitor: %s (%s)\n", name, addr.String())
					adapter.StopScan()
					connectHR(addr)
					connectingHR = false
				}()
			}
		})

		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already in progress") {
				time.Sleep(1200 * time.Millisecond)
				continue
			}
			fmt.Printf("[BLE] Scan error: %v. Retrying in 2s...\n", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func shouldPauseScan() bool {
	if connectingTrainer || connectingHR {
		return true
	}
	if trainerDevice != nil || hrDevice != nil {
		return true
	}
	return false
}

func connectTrainer(addr bluetooth.Address) {
	fmt.Println("[BLE] Connecting to Trainer...")
	device, err := Adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		fmt.Printf("[BLE] Connection failed: %v\n", err)
		return
	}

	trainerDevice = &device
	fmt.Println("[BLE] Trainer Connected! Initiating Deep GATT Discovery...")

	app.State.Lock()
	app.State.TrainerAddress = addr.String()
	app.State.Unlock()

	services, err := device.DiscoverServices(nil)
	if err != nil {
		fmt.Printf("[BLE] GATT Discovery Failed: %v\n", err)
		return
	}

	var deviceServices []integrity.GATTService

	for _, service := range services {
		gattSrv := integrity.GATTService{UUID: service.UUID().String()}

		chars, _ := service.DiscoverCharacteristics(nil)
		for _, char := range chars {
			props := "N/A"

			gattSrv.Characteristics = append(gattSrv.Characteristics, integrity.GATTCharacteristic{
				UUID:       char.UUID().String(),
				Properties: props,
			})

			if char.UUID() == CharCyclingPowerMeasure {
				fmt.Println("[BLE] Subscribing to Power Measurement...")

				var lastNotify time.Time

				char.EnableNotifications(func(buf []byte) {
					now := time.Now()
					var intervalMs float64

					if !lastNotify.IsZero() {
						intervalMs = float64(now.Sub(lastNotify).Milliseconds())
					} else {
						intervalMs = 1000.0
					}
					lastNotify = now

					power, cadence := parsePowerCadence(buf)

					app.State.Lock()
					app.State.RealPower = power
					app.State.RealCadence = cadence
					app.State.ConnectedReal = true
					app.State.PowerTimestamps = appendTrimTimes(app.State.PowerTimestamps, now, 80)
					app.State.ResponseLatencies = appendTrimFloat(app.State.ResponseLatencies, intervalMs, 80)
					app.State.Unlock()
				})
			}
		}
		deviceServices = append(deviceServices, gattSrv)
	}

	gattHash := integrity.GenerateGATTHash(deviceServices)
	fmt.Printf("[INTEGRITY] GATT Fingerprint Hash: %s\n", gattHash)

	app.State.Lock()
	if dev, ok := app.State.DiscoveredDevices[addr.String()]; ok {
		dev.GATTHash = gattHash
		app.State.DiscoveredDevices[addr.String()] = dev
	}
	app.State.Unlock()
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

	services, _ := device.DiscoverServices([]bluetooth.UUID{ServiceHeartRate})
	for _, service := range services {
		chars, _ := service.DiscoverCharacteristics([]bluetooth.UUID{CharHeartRateMeasure})
		for _, char := range chars {
			fmt.Println("[BLE] Subscribing to Heart Rate...")
			char.EnableNotifications(func(buf []byte) {
				t0 := time.Now()
				hr := parseHR(buf)
				latencyMs := time.Since(t0).Seconds() * 1000

				app.State.Lock()
				app.State.RealHR = hr
				app.State.ConnectedHR = true
				app.State.ResponseLatencies = appendTrimFloat(app.State.ResponseLatencies, latencyMs, 80)
				app.State.Unlock()
			})
		}
	}
}

func registerScanResult(result bluetooth.ScanResult) {
	address := result.Address.String()
	now := time.Now()

	mdHex := ""
	for _, md := range result.ManufacturerData() {
		mdHex += fmt.Sprintf("%04X:%X ", md.CompanyID, md.Data)
	}
	parsedMD := strings.TrimSpace(mdHex)

	app.State.Lock()
	defer app.State.Unlock()

	dev := app.State.DiscoveredDevices[address]
	if dev.Address == "" {
		dev = app.DiscoveredDevice{Address: address, FirstSeenMs: now.UnixMilli()}
		dev.Order = app.State.NextDeviceOrder
		app.State.NextDeviceOrder++
	}
	if dev.LastSeenMs > 0 {
		interval := float64(now.UnixMilli() - dev.LastSeenMs)
		if dev.ObservedAdvInterval == 0 {
			dev.ObservedAdvInterval = interval
		} else {
			dev.ObservedAdvInterval = (interval + dev.ObservedAdvInterval) / 2
		}
	}

	dev.Name = result.LocalName()
	dev.RSSI = result.RSSI
	dev.HasCyclingPower = result.HasServiceUUID(ServiceCyclingPower)
	dev.HasHeartRate = result.HasServiceUUID(ServiceHeartRate)
	dev.LastSeenUnix = now.Unix()
	dev.LastSeenMs = now.UnixMilli()

	if parsedMD != "" {
		dev.ManufacturerData = parsedMD
	}

	app.State.DiscoveredDevices[address] = dev
}

func appendTrimTimes(v []time.Time, value time.Time, max int) []time.Time {
	v = append(v, value)
	if len(v) > max {
		v = v[len(v)-max:]
	}
	return v
}

func appendTrimFloat(v []float64, value float64, max int) []float64 {
	v = append(v, value)
	if len(v) > max {
		v = v[len(v)-max:]
	}
	return v
}

// Helper: Parse Standard Cycling Power (0x2A63)
func parsePowerCadence(buf []byte) (int, int) {
	if len(buf) < 4 {
		return 0, 0
	}

	flags := uint16(buf[0]) | uint16(buf[1])<<8
	power := int(int16(uint16(buf[2]) | uint16(buf[3])<<8))

	offset := 4
	if flags&0x01 != 0 {
		offset += 1
	}
	if flags&0x04 != 0 {
		offset += 2
	}
	if flags&0x10 != 0 {
		offset += 6
	}

	cadence := 0
	if flags&0x20 != 0 && len(buf) >= offset+4 {
		// Placeholder: cadence logic
	}

	if power < 0 {
		power = 0
	}

	return power, cadence
}

// Helper: Parse Standard Heart Rate (0x2A37)
func parseHR(buf []byte) int {
	if len(buf) < 2 {
		return 0
	}

	flags := buf[0]
	hr := 0

	if flags&0x01 == 0 {
		hr = int(buf[1])
	} else if len(buf) >= 3 {
		hr = int(uint16(buf[1]) | uint16(buf[2])<<8)
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