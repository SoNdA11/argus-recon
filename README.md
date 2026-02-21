# ARGUS RECON
>
> BLE Signal Interception & Analysis Suite for IoT Fitness Devices

![Go](https://img.shields.io/badge/go-1.21+-blue)
![Architecture](https://img.shields.io/badge/arch-Bluetooth%20Low%20Energy%20(GATT)-blueviolet)
![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20MacOS%20%7C%20Linux-lightgrey)

## About The Project

**Argus Recon** is a cybersecurity Proof-of-Concept (PoC) and educational tool designed to audit and demonstrate vulnerabilities in **Bluetooth Low Energy (BLE)** implementations found in modern smart fitness equipment.

By exploiting the lack of encryption in standard GATT services (specifically `0x1818` Cycling Power), Argus Recon acts as a transparent **Man-in-the-Middle (MITM)** agent. It sits between the hardware (Smart Trainer) and the application (Zwift/MyWhoosh), allowing for real-time interception, analysis, and modification of telemetry data.

## Research Goals

* **Protocol Analysis:** Demonstrate how cleartext GATT transmission exposes user data.
* **Integrity Testing:** Prove the feasibility of real-time packet injection without hardware tampering.
* **Anti-Fraud Research:** Develop algorithms (like Pink Noise generation) to distinguish human bio-signatures from synthetic bots.

---

## Key Modules

### 1. Ghost Simulator (Synthetic Persona)

Generates a virtual cyclist indistinguishable from a human athlete.

* **Pink Noise Algorithm:** Uses 1/f noise to simulate natural muscular variability, bypassing entropy-based bot detection.
* **Bio-Mimicry:** Simulates physiological heart rate lag and cadence micro-jitters.
* **Use Case:** Testing platform anti-cheat systems and load testing.

### 2. Signal Interceptor (MITM Bridge)

The core "Attack" module. It links to a real Smart Trainer and acts as a proxy.

* **Active Interception:** Captures real BLE packets from the trainer.
* **Packet Injection:** Modifies power (Watts) values in real-time based on a "Boost" factor (Fixed Wattage or Percentage).
* **Re-Broadcasting:** Advertises as a standard Power Meter (`Argus X-Link`), making the modification invisible to the end application.

### 3. Dashboard (C2 Interface)

A modern, tactical web-based Command & Control interface.

* **Real-Time Telemetry:** Live visualization of Input vs. Output signals.
* **Control Vector:** Adjust injection parameters on the fly via WebSocket.
* **Cross-Platform:** Runs in any modern browser (Chrome/Edge/Firefox).

---

## Tech Stack

* **Core:** Go (Golang)
* **Bluetooth Stack:** TinyGo Bluetooth (HCI Abstraction)
* **Frontend:** HTML5, CSS3 (Glassmorphism UI), Vanilla JS
* **Communication:** WebSockets (Real-time duplex link)

---

## Quick Start Guide

### Prerequisites

* Go 1.21+ installed.
* A Bluetooth 4.0+ Adapter (USB Dongle or Internal).
  * Windows/Linux: Most generic CSR 4.0/5.0 dongles work.
  * macOS: Built-in Bluetooth is supported.

### Installation

```bash
# 1. Clone the repository
git clone https://github.com/YOUR_USERNAME/argus-recon.git
cd argus-recon

# 2. Install dependencies
go mod tidy

# 3. Run the Recon Agent
# (Note: Linux users may need 'sudo' for direct HCI access)
go run cmd/argus-recon/main.go
```

## Usage

* **Launch the Agent:** Run the command above.
* **Access Dashboard:** Open your browser and go to <http://localhost:8080>.
* **Select Mode:**
  * **Ghost Sim:** For virtual data generation.
  * **MITM Bridge:** To connect to a real trainer.
* **Connect App:** Open your cycling app (Zwift/MyWhoosh) and pair with the device named **"Argus Recon"**.

---

## Ethical Disclaimer

This tool was developed exclusively for educational and cybersecurity research purposes.

* **DO NOT** use this tool in competitive environments, online rankings, or sanctioned e-sports events.
* The use of packet injection tools violates the Terms of Service of most online platforms.
* The author assumes no responsibility for misuse. The goal is to advocate for better security standards (e.g., Bonding/Encryption) in IoT fitness devices.

## BLE Integrity Scanner

Argus Recon now includes an integrity scanner layer with:

* BLE target discovery list (MAC/name/RSSI/services)
* Real-time integrity score (0–100)
* Classification: genuine / suspect / emulator
* Behavioral signals: latency, jitter, notification rate, power-cadence drift

* Operator dashboard panel for target selection and signal reasons

See `docs/BLE_INTEGRITY_SCANNER.md` for architecture and deployment guidance.
