package app

import "sync"

type AppState struct {
	sync.Mutex
	Mode string

	SimBasePower int
	BoostValue   int
	BoostType    string

	RealPower     int
	RealCadence   int
	OutputPower   int
	OutputCadence int
	OutputHR      int

	ConnectedReal   bool
	ClientConnected bool
}

var State = &AppState{
	Mode:         "sim",
	SimBasePower: 150,
	BoostValue:   0,
	BoostType:    "fix",
}