# I2S MEMS Microphone Wiring Diagram
## INMP441/SPH0645 to BeagleBone Black

---

## Quick Reference

### Pin Connections

#### Microphone to BeagleBone

| Microphone Pin | Function | BeagleBone Black Pin | Description |
|----------------|----------|---------------------|-------------|
| **VDD** | Power (3.3V) | **P9_3** | 3.3V DC power supply |
| **GND** | Ground | **P9_1** | Digital ground |
| **SD** | Serial Data | **P9_30** | I2S data (McASP0_AXR0) |
| **WS** | Word Select | **P9_29** | I2S frame sync (McASP0_FSX) |
| **SCK** | Serial Clock | **P9_31** | I2S bit clock (McASP0_ACLKX) |
| **L/R or SEL** | Channel Select | **GND** | Left channel (connect to GND) |

#### BeagleBone Internal Clock Jumper (REQUIRED)

| From Pin | Function | To Pin | Function | Description |
|----------|----------|--------|----------|-------------|
| **P9_25** | CLKOUT2 (24.576 MHz) | **P9_28** | McASP0_AHCLKR | Clock source for perfect audio timing |

**IMPORTANT:** The P9_25 → P9_28 jumper wire is **required** for proper audio capture. This routes the BeagleBone's onboard 24.576 MHz clock to the McASP peripheral, providing perfect clock accuracy for 48 kHz sampling.

---

## Visual Wiring Diagram

```
                    INMP441 Module
                   ┌──────────────┐
                   │              │
                   │   [MIC]      │
                   │              │
                   │              │
                   └──┬─┬─┬─┬─┬─┬─┘
                      │ │ │ │ │ │
        VDD ──────────┘ │ │ │ │ └────── L/R
        GND ────────────┘ │ │ └──────── SCK
        SD ────────────────┘ └────────── WS


    BeagleBone Black P9 Header (Top View)

         GND  [1] ●  ● [2]  GND
        3.3V  [3] ●  ● [4]  3.3V
         VDD  [5] ●  ● [6]  VDD
         SYS  [7] ●  ● [8]  SYS
        PWM   [9] ●  ● [10] RESET
        UART [11] ●  ● [12] GPIO
        UART [13] ●  ● [14] PWM
        GPIO [15] ●  ● [16] GPIO
        I2C  [17] ●  ● [18] I2C
        I2C  [19] ●  ● [20] I2C
        UART [21] ●  ● [22] GPIO
        GPIO [23] ●  ● [24] UART
        GPIO [25] ●  ● [26] UART
        GPIO [27] ●  ● [28] (not used)
     I2S_WS  [29] ●  ● [30] I2S_DATA (SD)     ◄─── INMP441 SD
     I2S_CLK [31] ●  ● [32] VDD               ◄─── INMP441 SCK
        AIN4 [33] ●  ● [34] AGND
        AIN6 [35] ●  ● [36] AIN5
        AIN2 [37] ●  ● [38] AIN3
        AIN0 [39] ●  ● [40] AIN1
        GND  [41] ●  ● [42] GPIO
        GND  [43] ●  ● [44] GPIO
        GND  [45] ●  ● [46] GPIO

    P9_1  (GND)   ────────────────► Microphone GND
    P9_3  (3.3V)  ────────────────► Microphone VDD
    P9_30 (I2S_DATA) ─────────────► Microphone SD
    P9_29 (I2S_WS)   ─────────────► Microphone WS
    P9_31 (I2S_CLK)  ─────────────► Microphone SCK
    P9_1  (GND)   ────────────────► Microphone L/R or SEL

    *** CRITICAL: Clock Jumper Wire (on BeagleBone header) ***
    P9_25 (CLKOUT2 - 24.576 MHz) ═══► P9_28 (McASP AHCLKR input)
```

### Clock Jumper Explanation

The **P9_25 → P9_28** jumper wire is essential for proper audio operation:

**Why it's needed:**
- BeagleBone has a 24.576 MHz oscillator on-board (originally for HDMI audio)
- This frequency is **perfect** for 48 kHz audio (24.576 MHz / 512 = 48 kHz exactly)
- Without this, the BeagleBone uses DPLL clocks which have 0.8% error
- The SPH0645/INMP441 microphones reject data with that clock error

**How it works:**
- P9_25 is configured to output CLKOUT2 (the 24.576 MHz clock)
- P9_28 is configured as McASP0's AHCLKR (Auxiliary High-frequency Clock Receive)
- A simple jumper wire routes the clock from output to input
- McASP uses this perfect clock instead of the inaccurate DPLL clock

**Physical implementation:**
- Use a standard female-to-female jumper wire
- Length: ~2-3 inches (5-8 cm)
- This wire stays on the BeagleBone header (not connected to microphone)
- Both P9_25 and P9_28 are 3.3V logic - electrically safe

---

## Detailed Connection Guide

### Power Connections

**1. VDD (Power Supply)**
- **INMP441 Pin:** VDD
- **Connect to:** BeagleBone Black P9_3 (3.3V)
- **Wire color suggestion:** Red
- **Notes:**
  - INMP441 operates at 1.8V - 3.3V
  - BeagleBone provides 3.3V, which is optimal
  - Current draw: ~1.5mA (very low)

**2. GND (Ground)**
- **INMP441 Pin:** GND
- **Connect to:** BeagleBone Black P9_1 (DGND)
- **Wire color suggestion:** Black
- **Notes:**
  - Use digital ground (P9_1, P9_43, or P9_45)
  - Keep ground wire short and direct
  - Common ground reference for all signals

### I2S Signal Connections

**3. SD (Serial Data)**
- **INMP441 Pin:** SD (Serial Data Out)
- **Connect to:** BeagleBone Black P9_28
- **Function:** McASP0_AXR2 (I2S data receive)
- **Wire color suggestion:** Yellow
- **Signal direction:** Output from microphone → Input to BeagleBone
- **Notes:**
  - This carries the digital audio samples
  - 24-bit data from microphone, captured as 16-bit or 32-bit by BBB

**4. WS (Word Select / Frame Sync)**
- **INMP441 Pin:** WS (Word Select)
- **Connect to:** BeagleBone Black P9_29
- **Function:** McASP0_FSX (Frame Sync)
- **Wire color suggestion:** Green
- **Signal direction:** Output from BeagleBone → Input to microphone
- **Notes:**
  - Also called LRCLK (Left/Right Clock)
  - Determines left vs right channel timing
  - Frequency = Sample rate (48 kHz typical)

**5. SCK (Serial Clock / Bit Clock)**
- **INMP441 Pin:** SCK (Serial Clock)
- **Connect to:** BeagleBone Black P9_31
- **Function:** McASP0_ACLKX (Audio Clock)
- **Wire color suggestion:** Blue
- **Signal direction:** Output from BeagleBone → Input to microphone
- **Notes:**
  - Also called BCLK (Bit Clock)
  - Frequency = Sample rate × Bits per channel × Channels
  - For 48kHz, 32-bit, 2-channel: 3.072 MHz
  - BeagleBone generates this clock

**6. L/R (Left/Right Channel Select)**
- **INMP441 Pin:** L/R or SEL
- **Connect to:** GND (P9_1)
- **Wire color suggestion:** Black (share with GND)
- **Notes:**
  - L/R = GND → Left channel
  - L/R = VDD → Right channel
  - For mono microphone, connect to GND (left channel)

---

## Physical Wiring Instructions

### Step-by-Step

1. **Power Off** the BeagleBone Black
   - Unplug USB or DC power
   - Verify no LEDs are lit

2. **Identify the INMP441 pins**
   - Check the module silkscreen labels
   - Most modules have pins in order: VDD, GND, SD, WS, SCK, L/R
   - Verify with module documentation

3. **Connect Power (Red wire)**
   - INMP441 VDD → BeagleBone P9_3 (3.3V)
   - Use female-to-female jumper wire

4. **Connect Ground (Black wire)**
   - INMP441 GND → BeagleBone P9_1 (GND)
   - Female-to-female jumper wire

5. **Connect L/R Channel Select (Black wire or jumper)**
   - INMP441 L/R → BeagleBone P9_1 (GND)
   - Can share same GND pin with wire from step 4
   - Or use a separate GND pin (P9_43 or P9_45)

6. **Connect I2S Data (Yellow wire)**
   - INMP441 SD → BeagleBone P9_28
   - Female-to-female jumper wire

7. **Connect I2S Word Select (Green wire)**
   - INMP441 WS → BeagleBone P9_29
   - Female-to-female jumper wire

8. **Connect I2S Clock (Blue wire)**
   - Microphone SCK → BeagleBone P9_31
   - Female-to-female jumper wire

9. **Connect Clock Jumper (ANY color wire) - CRITICAL**
   - BeagleBone P9_25 → BeagleBone P9_28
   - Female-to-female jumper wire (~2-3 inches)
   - This stays on the BeagleBone header (does not connect to microphone)
   - Routes 24.576 MHz clock to McASP for perfect audio timing

10. **Verify Connections**
   - Double-check each wire against the table above
   - Ensure no loose connections
   - Verify no short circuits between adjacent pins
   - Verify the P9_25 → P9_28 clock jumper is in place

11. **Power On**
    - Reconnect BeagleBone power
    - Boot Linux
    - Proceed with device tree configuration

---

## Connection Photo Reference

```
Side view of INMP441 module:

    Microphone opening (top)
          ▼
    ┌───────────┐
    │   [■■■]   │  ← Sound port (point toward audio source)
    │           │
    │  INMP441  │
    │           │
    └─┬─┬─┬─┬─┬─┘
      │ │ │ │ │ │
      1 2 3 4 5 6
      │ │ │ │ │ │
      │ │ │ │ │ └─ L/R (Channel select)
      │ │ │ │ └─── SCK (Serial clock)
      │ │ │ └───── WS (Word select)
      │ │ └─────── SD (Serial data)
      │ └───────── GND (Ground)
      └─────────── VDD (Power 1.8-3.3V)
```

---

## Pin Function Details

### BeagleBone Black McASP0 Pins

The BeagleBone Black has a Multi-channel Audio Serial Port (McASP) peripheral that handles I2S natively in hardware. We're using McASP0.

| Pin | Ball | Mode | Function | Description |
|-----|------|------|----------|-------------|
| P9_28 | A13 | Mode 2 | mcasp0_axr2 | Audio data receive channel 2 |
| P9_29 | B13 | Mode 0 | mcasp0_fsx | Frame sync (word select) |
| P9_31 | A13 | Mode 0 | mcasp0_aclkx | Audio bit clock |

**Note:** The McASP peripheral is configured via device tree to enable these pins for I2S operation.

---

## Wire Specifications

### Recommended Wire

- **Type:** Female-to-female jumper wires (DuPont connectors)
- **Length:** 10-15 cm (4-6 inches) for microphone connections, 5-8 cm (2-3 inches) for clock jumper
- **Gauge:** 22-26 AWG
- **Count needed:** 7 wires total
  - 6 wires for microphone connections
  - 1 wire for P9_25 → P9_28 clock jumper (on BeagleBone header)

### Wire Quality Considerations

- **Short wires preferred:** Reduces signal integrity issues
- **Twisted pairs (optional):** Twist data/clock pairs together to reduce noise
- **Separate power:** Keep power wire away from I2S signal wires
- **Avoid long runs:** Keep total wire length under 20 cm if possible

### Color Coding Recommendation

| Wire Color | Connection | Purpose |
|------------|------------|---------|
| Red | VDD (3.3V) | Power |
| Black | GND | Ground |
| Black | L/R | Channel select (to GND) |
| Yellow | SD (Data) | I2S serial data |
| Green | WS (Word Select) | I2S frame sync |
| Blue | SCK (Clock) | I2S bit clock |

---

## Common Mistakes to Avoid

### ❌ Wrong Connections

1. **Connecting VDD to 5V**
   - ❌ Wrong: INMP441 VDD → BeagleBone P9_5 (VDD_5V)
   - ✅ Correct: INMP441 VDD → BeagleBone P9_3 (VDD_3V3)
   - **Risk:** May damage microphone (rated for 3.3V max)

2. **Using Wrong Ground**
   - ❌ Wrong: INMP441 GND → BeagleBone P9_34 (AGND - Analog ground)
   - ✅ Correct: INMP441 GND → BeagleBone P9_1 (DGND - Digital ground)
   - **Reason:** I2S is digital signal, use digital ground

3. **Swapping SCK and WS**
   - ❌ Wrong: SCK → P9_29, WS → P9_31
   - ✅ Correct: SCK → P9_31, WS → P9_29
   - **Result:** No audio capture, device tree won't match

4. **Forgetting L/R Pin**
   - ❌ Wrong: L/R pin left floating (not connected)
   - ✅ Correct: L/R → GND (for left channel)
   - **Result:** Undefined behavior, may not capture audio

5. **Wrong I2S Data Pin**
   - ❌ Wrong: SD → P9_27 or other GPIO
   - ✅ Correct: SD → P9_28 (McASP0_AXR2)
   - **Result:** Hardware won't route I2S data to Linux

---

## Verification Checklist

Before powering on, verify:

- [ ] VDD → P9_3 (3.3V, NOT 5V)
- [ ] GND → P9_1 or P9_43 or P9_45
- [ ] SD → P9_28 (not P9_27 or other pins)
- [ ] WS → P9_29
- [ ] SCK → P9_31
- [ ] L/R → GND (P9_1 or separate ground pin)
- [ ] No loose connections
- [ ] No short circuits between adjacent pins
- [ ] Microphone opening faces outward (toward audio source)
- [ ] All wires are secure

---

## Testing the Connection

### After Wiring

1. **Power on BeagleBone**
   ```bash
   # Check that 3.3V is present
   # You should see INMP441 LED (if equipped) light up
   ```

2. **Load Device Tree** (after configuration in Phase 1)
   ```bash
   sudo sh -c "echo 'BB-I2S-MIC' > /sys/devices/platform/bone_capemgr/slots"
   ```

3. **Check ALSA Detection**
   ```bash
   arecord -l
   # Should show: card 0: I2SMicrophone [I2S-Microphone], device 0
   ```

4. **Test Audio Capture**
   ```bash
   arecord -D hw:0,0 -f S16_LE -r 48000 -c 1 -d 5 test.wav
   # Speak or make noise near microphone
   aplay test.wav
   # Should hear your recording
   ```

### Expected Signals (with oscilloscope/logic analyzer)

If you have measurement tools:

- **WS (P9_29):** 48 kHz square wave (20.8 μs period)
- **SCK (P9_31):** 3.072 MHz (64 × 48 kHz)
- **SD (P9_28):** Serial data, changes on SCK edges

---

## Troubleshooting

### No Audio Captured

**Check:**
1. VDD connected to 3.3V (not 5V, not floating)
2. GND connected properly
3. L/R pin connected to GND
4. All I2S pins match diagram exactly
5. Device tree loaded successfully
6. ALSA device shows up in `arecord -l`

### Distorted/Noisy Audio

**Check:**
1. Wire lengths (keep under 20cm)
2. Wires not running parallel to power lines
3. Ground connection secure
4. No loose connections

### Device Not Detected

**Check:**
1. Device tree overlay compiled and loaded
2. Pin mux configuration correct (P9_28/29/31)
3. McASP module enabled in kernel
4. Check `dmesg` for errors

---

## Mounting Considerations

### Physical Positioning

- **Microphone opening:** Face toward audio source
- **Distance:** 30-100 cm from speakers optimal
- **Avoid:** Placing inside enclosed box (muffles sound)
- **Secure:** Use double-sided tape or standoffs to prevent vibration

### Cable Management

- **Route away from:** Power supplies, DC-DC converters
- **Secure wires:** Use zip ties or tape to prevent wire movement
- **Strain relief:** Don't pull on connections
- **Length:** Keep as short as practical (10-15 cm ideal)

---

## Next Steps

After wiring is complete:

1. ✅ Verify all connections with checklist above
2. ➡️ Power on BeagleBone Black
3. ➡️ Configure device tree (see Phase 1 of refactor plan)
4. ➡️ Test with `arecord` command
5. ➡️ Proceed with Go ALSA integration

---

## Reference

**INMP441 Datasheet:** https://invensense.tdk.com/products/digital/inmp441/
**BeagleBone Black SRM:** Section 7.1 (Expansion Headers)
**AM335x TRM:** Chapter 22 (McASP)

---

**Created:** 2025-12-17
**For:** led-sound-light-control project
**Branch:** feature/i2s-microphone
