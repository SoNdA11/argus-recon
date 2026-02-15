package app

import "sync"

type AppState struct {
	sync.Mutex
	Mode string		// "sim" or "bridge"

	// Configuration
	SimBasePower int
	BoostValue   int
	BoostType    string

	// Real Telemetry (Input)
	RealPower     int
	RealCadence   int
	RealHR		  int

	// Final Telemetry (Output to App)
	OutputPower   int
	OutputCadence int
	OutputHR      int

	// Connectivity Status
	ConnectedReal   bool
	ConnectedHR		bool
	ClientConnected bool
}

var State = &AppState{
	Mode:         "sim",
	SimBasePower: 150,
	BoostValue:   0,
	BoostType:    "fix",
}