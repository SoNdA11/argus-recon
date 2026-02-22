package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/SoNdA11/argus-recon/internal/app"
)

// GATTCharacteristic represents the signature of a characteristic
type GATTCharacteristic struct {
	UUID       string
	Properties string
}

// GATTService represents a service and its internal characteristics
type GATTService struct {
	UUID            string
	Characteristics []GATTCharacteristic
}

// DeviceFingerprint stores the structural identity of the device
type DeviceFingerprint struct {
	MAC              string
	Name             string
	ManufacturerData string
	Services         []GATTService
	GATTHash         string
}

// GenerateGATTHash creates a deterministic signature of the GATT table
// Real devices (ThinkRider) will have different hashes compared to software-based emulators
func GenerateGATTHash(services []GATTService) string {
	var builder strings.Builder

	// Sort services to guarantee determinism
	sort.Slice(services, func(i, j int) bool {
		return services[i].UUID < services[j].UUID
	})

	for _, srv := range services {
		builder.WriteString(srv.UUID + "|")

		sort.Slice(srv.Characteristics, func(i, j int) bool {
			return srv.Characteristics[i].UUID < srv.Characteristics[j].UUID
		})

		for _, char := range srv.Characteristics {
			builder.WriteString(char.UUID + ":" + char.Properties + ";")
		}
	}

	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// EvaluateDeviceIdentity merges the structural fingerprint with behavioral signals
func EvaluateDeviceIdentity(fp DeviceFingerprint, signals app.IntegritySignals) (int, app.DeviceClassification, []string) {
	score := 100
	var reasons []string
	class := app.ClassificationGenuine

	// 1. OUI (MAC) Verification
	vendor := vendorFromMAC(fp.MAC)
	switch vendor {
	case "Thinkrider":
		reasons = append(reasons, "[+] Physical Hardware OUI detected (ThinkRider).")
	case "Intel", "Realtek":
		score -= 30
		reasons = append(reasons, "[-] Computer Adapter OUI detected. Possible Emulator/Relay.")
	}

	// 2. Behavioral Analysis (Critical Latency)
	if signals.LatencyMeanMs > 80.0 || signals.LatencyJitterMs > 15.0 {
		score -= 25
		reasons = append(reasons, "[-] ATT response latency incompatible with embedded RTOS.")
	}

	// 3. Manufacturer Data Verification
	if fp.ManufacturerData == "" {
		score -= 10
		reasons = append(reasons, "[!] Missing Manufacturer Specific Data in Advertising.")
	}

	// 4. Final Classification
	if score >= 80 {
		class = app.ClassificationGenuine
	} else if score >= 50 {
		class = app.ClassificationSuspect
	} else {
		class = app.ClassificationEmulator
	}

	// If the device name contains the system's own signature
	if strings.Contains(strings.ToLower(fp.Name), "argus") {
		score = 10
		class = app.ClassificationEmulator
		reasons = append(reasons, "[!] VIRTUAL identity confirmed (Argus Recon).")
	}

	return score, class, reasons
}