package ble

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
	"tinygo.org/x/bluetooth"
)

// UUIDs
var (
	uuidServicePower   = bluetooth.ServiceUUIDCyclingPower
	uuidServiceCadence = bluetooth.ServiceUUIDCyclingSpeedAndCadence
	uuidServiceHR      = bluetooth.ServiceUUIDHeartRate
	uuidCharPower      = bluetooth.CharacteristicUUIDCyclingPowerMeasurement
	uuidCharCadence    = bluetooth.CharacteristicUUIDCSCMeasurement
	uuidCharHR         = bluetooth.CharacteristicUUIDHeartRateMeasurement
)

var (
	charPower   bluetooth.Characteristic
	charCadence bluetooth.Characteristic
	charHR      bluetooth.Characteristic
)

var (
	cumCrankRevs  uint16 = 0
	lastCrankTime uint16 = 0
)

func SetupServices() {
	fmt.Println("[BLE] Provisioning GATT Services...")

	Adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.New16BitUUID(0x1800),
		Characteristics: []bluetooth.CharacteristicConfig{
			{UUID: bluetooth.New16BitUUID(0x2A01), Flags: bluetooth.CharacteristicReadPermission, Value: []byte{0x82, 0x04}},
		},
	})

	Adapter.AddService(&bluetooth.Service{
		UUID: uuidServicePower,
		Characteristics: []bluetooth.CharacteristicConfig{
			{Handle: &charPower, UUID: uuidCharPower, Flags: bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission, Value: []byte{0x00, 0x00, 0x00, 0x00}},
			{UUID: bluetooth.New16BitUUID(0x2A65), Flags: bluetooth.CharacteristicReadPermission, Value: []byte{0x00, 0x00, 0x00, 0x00}},
		},
	})

	Adapter.AddService(&bluetooth.Service{
		UUID: uuidServiceCadence,
		Characteristics: []bluetooth.CharacteristicConfig{
			{Handle: &charCadence, UUID: uuidCharCadence, Flags: bluetooth.CharacteristicNotifyPermission, Value: []byte{0x02, 0x00, 0x00, 0x00, 0x00}},
		},
	})

	Adapter.AddService(&bluetooth.Service{
		UUID: uuidServiceHR,
		Characteristics: []bluetooth.CharacteristicConfig{
			{Handle: &charHR, UUID: uuidCharHR, Flags: bluetooth.CharacteristicNotifyPermission, Value: []byte{0x00, 0x00}},
		},
	})

	adv := Adapter.DefaultAdvertisement()
	err := adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    "Argus X-Link",
		ServiceUUIDs: []bluetooth.UUID{uuidServicePower},
	})
	if err != nil {
		log.Fatal(err)
	}
	adv.Start()

	registerLocalVirtualDevice("Argus X-Link")
}

func registerLocalVirtualDevice(name string) {
	address := "LOCAL-VIRTUAL"
	if mac, err := Adapter.Address(); err == nil {
		address = mac.String()
	}

	app.State.Lock()
	defer app.State.Unlock()

	now := time.Now()
	app.State.AdapterAddress = address
	app.State.LocalVirtualAddr = address
	order := app.State.NextDeviceOrder
	if existing, ok := app.State.DiscoveredDevices[address]; ok && existing.Order > 0 {
		order = existing.Order
	} else {
		app.State.NextDeviceOrder++
	}

	delete(app.State.DiscoveredDevices, "LOCAL-VIRTUAL")
	app.State.DiscoveredDevices[address] = app.DiscoveredDevice{
		Address:             address,
		Name:                name,
		RSSI:                0,
		HasCyclingPower:     true,
		HasHeartRate:        true,
		LastSeenUnix:        now.Unix(),
		LastSeenMs:          now.UnixMilli(),
		FirstSeenMs:         now.UnixMilli(),
		Order:               order,
		ObservedAdvInterval: 1000,
	}
}

func UpdateOutputs(watts, rpm, hr int) {
	pPayload := make([]byte, 4)
	binary.LittleEndian.PutUint16(pPayload[0:2], 0)
	binary.LittleEndian.PutUint16(pPayload[2:4], uint16(watts))
	charPower.Write(pPayload)

	if rpm > 0 {
		cumCrankRevs++
		timePerRev := (60.0 / float64(rpm)) * 1024.0
		lastCrankTime += uint16(timePerRev)
	}

	cPayload := make([]byte, 5)
	cPayload[0] = 0x02 // Flags: Crank Data Present
	binary.LittleEndian.PutUint16(cPayload[1:3], cumCrankRevs)
	binary.LittleEndian.PutUint16(cPayload[3:5], lastCrankTime)
	charCadence.Write(cPayload)

	if hr > 190 {
		hr = 190
	}
	charHR.Write([]byte{0x00, uint8(hr)})
}