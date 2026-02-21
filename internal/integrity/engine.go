package integrity

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/SoNdA11/argus-recon/internal/app"
)

func Start() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		recompute()
	}
}

func recompute() {
	app.State.Lock()
	defer app.State.Unlock()

	latMean, latJitter := stats(app.State.ResponseLatencies)
	powerHz := notificationRate(app.State.PowerTimestamps)
	drift := cadencePowerDrift(app.State.RealPower, app.State.RealCadence)
	stressDropRate := stressProxy(powerHz)
	mtuVar := mtuProxyVariance(latJitter)

	if powerHz > 0 && latMean == 0 {
		latMean = (1000.0 / powerHz) * 0.35
	}
	if powerHz > 0 && latJitter == 0 {
		latJitter = math.Max(0.9, latMean*0.08)
	}

	if app.State.IntegrityReports == nil {
		app.State.IntegrityReports = map[string]app.IntegrityReport{}
	}

	now := time.Now().Unix()
	for addr, dev := range app.State.DiscoveredDevices {
		report := buildReportForDevice(dev, addr == app.State.TrainerAddress, addr == app.State.LocalVirtualAddr, app.State.ConnectedReal,
			latMean, latJitter, powerHz, drift, stressDropRate, mtuVar)
		report.LastUpdatedUnix = now
		app.State.IntegrityReports[addr] = report
	}

	selected := app.State.TrainerAddress
	if selected == "" {
		for addr := range app.State.DiscoveredDevices {
			selected = addr
			break
		}
	}
	if report, ok := app.State.IntegrityReports[selected]; ok {
		app.State.Integrity = report
	}
}

func buildReportForDevice(dev app.DiscoveredDevice, isSelectedTrainer, isLocalVirtual, connectedReal bool, latMean, latJitter, powerHz, drift, stressDropRate, mtuVar float64) app.IntegrityReport {
	reasons := []string{}
	score := 70
	class := app.ClassificationSuspect
	confidence := 0.55

	oui := ""
	if len(dev.Address) >= 8 {
		oui = strings.ToUpper(dev.Address[:8])
	}
	vendor := vendorFromMAC(dev.Address)

	signals := app.IntegritySignals{}
	if isSelectedTrainer {
		signals = app.IntegritySignals{
			LatencyMeanMs:       round(latMean),
			LatencyJitterMs:     round(latJitter),
			PowerNotifyHz:       round(powerHz),
			PowerCadenceDrift:   round(drift),
			StressDropRate:      round(stressDropRate),
			MTUBehaviorVariance: round(mtuVar),
		}
	} else {
		signals = app.IntegritySignals{
			LatencyMeanMs:       0,
			LatencyJitterMs:     0,
			PowerNotifyHz:       0,
			PowerCadenceDrift:   0,
			StressDropRate:      1,
			MTUBehaviorVariance: 0,
		}
	}

	if isLocalVirtual || strings.Contains(strings.ToLower(dev.Name), "argus") {
		score = 20
		class = app.ClassificationEmulator
		confidence = 0.95
		reasons = append(reasons, "Local emulated device detected (Argus X-Link).")
		reasons = append(reasons, "Host adapter signature detected; not dedicated embedded hardware.")
	} else if isSelectedTrainer && connectedReal && dev.HasCyclingPower {
		score = 88
		class = app.ClassificationGenuine
		confidence = 0.86
		if signals.PowerNotifyHz < 0.8 || signals.PowerNotifyHz > 2.5 {
			score = 74
			class = app.ClassificationSuspect
			confidence = 0.66
			reasons = append(reasons, "Notification rate outside expected range for real trainer.")
		} else {
			reasons = append(reasons, "Real trainer connected with coherent notification profile.")
		}
		if vendor == "Intel" || vendor == "Realtek" {
			reasons = append(reasons, "Host adapter OUI detected; verify no software relay is present.")
			score -= 8
		}
	} else {
		reasons = append(reasons, "No active ATT telemetry for strong classification.")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Awaiting additional data to increase confidence.")
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return app.IntegrityReport{
		TargetAddress:   dev.Address,
		TargetName:      dev.Name,
		Score:           score,
		Classification:  class,
		Confidence:      confidence,
		Reasons:         reasons,
		ObservedPHY:     "1M",
		ObservedBLEVers: "4.2+",
		ObservedOUI:     oui,
		VendorGuess:     vendor,
		Signals:         signals,
	}
}

func stats(values []float64) (mean, stddev float64) {
	if len(values) == 0 {
		return 0, 0
	}
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	for _, v := range values {
		stddev += math.Pow(v-mean, 2)
	}
	stddev = math.Sqrt(stddev / float64(len(values)))
	return mean, stddev
}

func notificationRate(ts []time.Time) float64 {
	if len(ts) < 3 {
		return 0
	}
	sorted := append([]time.Time{}, ts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Before(sorted[j]) })
	window := sorted[len(sorted)-1].Sub(sorted[0]).Seconds()
	if window <= 0 {
		return 0
	}
	return float64(len(sorted)-1) / window
}

func cadencePowerDrift(power, cadence int) float64 {
	if power == 0 || cadence == 0 {
		return 0
	}
	ratio := float64(power) / float64(cadence)
	baseline := 2.9
	return math.Abs(ratio-baseline) * 20
}

func stressProxy(powerHz float64) float64 {
	if powerHz == 0 {
		return 1
	}
	if powerHz >= 1.0 {
		return 0.1
	}
	return 1 - powerHz
}

func mtuProxyVariance(jitter float64) float64 {
	if jitter == 0 {
		return 0
	}
	if jitter > 5 {
		return 1
	}
	return jitter / 5
}

func round(v float64) float64 {
	return math.Round(v*100) / 100
}