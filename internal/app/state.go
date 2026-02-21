package app

import (
	"sync"
	"time"
)

type DeviceClassification string

const (
	ClassificationGenuine  DeviceClassification = "genuine"
	ClassificationSuspect  DeviceClassification = "suspect"
	ClassificationEmulator DeviceClassification = "emulator"
)

type DiscoveredDevice struct {
	Address             string  `json:"address"`
	Name                string  `json:"name"`
	RSSI                int16   `json:"rssi"`
	HasCyclingPower     bool    `json:"hasCyclingPower"`
	HasHeartRate        bool    `json:"hasHeartRate"`
	LastSeenUnix        int64   `json:"lastSeenUnix"`
	LastSeenMs          int64   `json:"-"`
	FirstSeenMs         int64   `json:"-"`
	Order               int64   `json:"-"`
	ObservedAdvInterval float64 `json:"observedAdvIntervalMs"`
}

type IntegritySignals struct {
	LatencyMeanMs       float64 `json:"latencyMeanMs"`
	LatencyJitterMs     float64 `json:"latencyJitterMs"`
	PowerNotifyHz       float64 `json:"powerNotifyHz"`
	PowerCadenceDrift   float64 `json:"powerCadenceDrift"`
	StressDropRate      float64 `json:"stressDropRate"`
	MTUBehaviorVariance float64 `json:"mtuBehaviorVariance"`
}

type IntegrityReport struct {
	TargetAddress   string               `json:"targetAddress"`
	TargetName      string               `json:"targetName"`
	Score           int                  `json:"score"`
	Classification  DeviceClassification `json:"classification"`
	Confidence      float64              `json:"confidence"`
	Reasons         []string             `json:"reasons"`
	ObservedPHY     string               `json:"observedPhy"`
	ObservedBLEVers string               `json:"observedBleVersion"`
	ObservedOUI     string               `json:"observedOui"`
	VendorGuess     string               `json:"vendorGuess"`
	LastUpdatedUnix int64                `json:"lastUpdatedUnix"`
	Signals         IntegritySignals     `json:"signals"`
}

type AppState struct {
	sync.Mutex
	Mode string // "sim" or "bridge"

	// Configuration
	SimBasePower int
	BoostValue   int
	BoostType    string

	// Real Telemetry (Input)
	RealPower   int
	RealCadence int
	RealHR      int

	// Final Telemetry (Output to App)
	OutputPower   int
	OutputCadence int
	OutputHR      int

	// Connectivity Status
	ConnectedReal   bool
	ConnectedHR     bool
	ClientConnected bool

	// Integrity Scanner
	DiscoveredDevices map[string]DiscoveredDevice
	NextDeviceOrder   int64
	AdapterAddress    string
	LocalVirtualAddr  string
	TrainerAddress    string
	Integrity         IntegrityReport
	IntegrityReports  map[string]IntegrityReport
	ResponseLatencies []float64
	PowerTimestamps   []time.Time
}

var State = &AppState{
	Mode:              "sim",
	SimBasePower:      150,
	BoostValue:        0,
	BoostType:         "fix",
	DiscoveredDevices: map[string]DiscoveredDevice{},
	Integrity: IntegrityReport{
		Score:           50,
		Classification:  ClassificationSuspect,
		Confidence:      0.35,
		Reasons:         []string{"Waiting for BLE telemetry baseline."},
		ObservedPHY:     "1M",
		ObservedBLEVers: "4.2+",
	},
}