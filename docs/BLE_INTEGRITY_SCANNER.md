# BLE Integrity Scanner — Argus Recon

## Overview

This module adds a multi-layer integrity analysis for BLE targets with a focus on Smart Trainers.

### Implemented Layers

1. **Device Identification**

   * MAC address
   * Local name
   * RSSI
   * Presence of Cycling Power and Heart Rate services
   * Observed advertising interval (estimated via passive scanner)

2. **Behavioral Analysis**

   * Average ATT processing latency (proxy)
   * Jitter (standard deviation)
   * Power notification frequency (Hz)
   * Power/cadence drift

3. **GATT Fingerprint (initial baseline)**

   * Validation of standard service presence (Cycling Power)
   * Stability indicators for future expansion of service order, handles, and descriptors

4. **Anti-Emulation Heuristics**

   * Low temporal entropy
   * Notification rate below expected threshold
   * Performance degradation under stress (proxy)
   * MTU/timing behavior variance (proxy)

## Integrity Score (0–100)

The score is composed of heuristic penalties with final classification:

* **🟢 Genuine**: score >= 80
* **🟡 Suspicious**: 55 <= score < 80
* **🔴 Emulator / Active attack**: score < 55

## Architecture with 3 BLE Adapters

Recommendation to reduce interference and improve reliability:

* **Dongle #1 (dedicated scanner):** passive discovery and fingerprinting
* **Dongle #2 (real trainer link):** connection and ATT/GATT data collection
* **Dongle #3 (Argus emulator):** advertising/peripheral for client app

## Offensive/Defensive Security

This repository maintains a focus on **defensive research and integrity assessment**.
For responsible use:

* do not use in competitive environments
* do not use for fraud
* validate tests only in authorized laboratory settings
