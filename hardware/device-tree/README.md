# I2S Microphone Device Tree Overlay

This directory contains the device tree overlay and tools for enabling I2S MEMS microphone (INMP441) support on BeagleBone Black.

## Files

- **BB-I2S-MIC-00A0.dts** - Device tree source for I2S microphone
- **Makefile** - Build and deployment automation
- **load-overlay.sh** - Script to load overlay on BeagleBone
- **test-audio.sh** - Audio capture test script

## Quick Start

### On Local Machine

1. **Compile the device tree overlay:**

   ```bash
   cd hardware/device-tree
   make compile
   ```

2. **Deploy to BeagleBone:**
   ```bash
   make deploy DEPLOY_HOST=bbb2.wegman
   ```

### On BeagleBone Black

3. **Load the overlay:**

   ```bash
   ssh bbb2.wegman
   cd ~/led-sound-light-control/hardware/device-tree
   ./load-overlay.sh
   ```

4. **Test audio capture:**
   ```bash
   ./test-audio.sh
   ```

## Detailed Setup

### Step 1: Compile Device Tree

The device tree overlay must be compiled from source (.dts) to binary (.dtbo):

```bash
make compile
```

This produces `BB-I2S-MIC-00A0.dtbo`.

### Step 2: Deploy to BeagleBone

Copy the compiled overlay to the BeagleBone:

```bash
# Deploy to bbb2.wegman (default)
make deploy

# Deploy to different host
make deploy DEPLOY_HOST=bbb1.wegman
```

This copies the `.dtbo` file to `/lib/firmware/` on the BeagleBone.

### Step 3: Load the Overlay

On the BeagleBone, load the device tree overlay:

```bash
sudo sh -c "echo 'BB-I2S-MIC' > /sys/devices/platform/bone_capemgr/slots"
```

Or use the helper script:

```bash
./load-overlay.sh
```

### Step 4: Verify ALSA Detection

Check that the I2S microphone is detected:

```bash
arecord -l
```

Expected output:

```
**** List of CAPTURE Hardware Devices ****
card 0: I2SMicrophone [I2S-Microphone], device 0: ...
```

### Step 5: Test Audio Capture

Record a 5-second test:

```bash
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 test.wav
```

Make noise near the microphone during recording.

Play back:

```bash
aplay test.wav
```

Or use the automated test:

```bash
./test-audio.sh
```

## Configuration Details

### Pin Assignments

| INMP441 Pin | BBB Pin | Function                 |
| ----------- | ------- | ------------------------ |
| SD          | P9_28   | McASP0_AXR2 (Data)       |
| WS          | P9_29   | McASP0_FSX (Frame Sync)  |
| SCK         | P9_31   | McASP0_ACLKX (Bit Clock) |

### Audio Format

- **Sample Rate:** 48000 Hz
- **Format:** S32_LE (32-bit signed little-endian)
- **Channels:** 1 (Mono, left channel)
- **Bit Clock:** 3.072 MHz (48kHz × 32bit × 2ch)

### Device Tree Configuration

The overlay configures:

- Pin multiplexing for McASP0 pins
- McASP0 peripheral in I2S mode
- Simple audio card driver
- SPDIF receiver codec (generic I2S codec)

## Load on Boot

To automatically load the overlay on boot, you can either manually edit uEnv.txt or deploy our pre-configured version.

### Option A: Deploy Managed uEnv.txt (Recommended)

We provide a uEnv.txt that's pre-configured with:

- I2S microphone overlay enabled
- Audio subsystem enabled (NOT disabled)
- PRU overlay enabled

**Deploy it:**

```bash
make deploy-uenv DEPLOY_HOST=bbb2.wegman
```

This will:

- Backup the existing /boot/uEnv.txt (to /boot/uEnv.txt.backup.TIMESTAMP)
- Deploy our uEnv.txt
- Prompt you to reboot

**After reboot**, the I2S overlay will be automatically loaded.

### Option B: Manual Edit

1. Edit `/boot/uEnv.txt`:

   ```bash
   ssh bbb2.wegman
   sudo nano /boot/uEnv.txt
   ```

2. Add this line (around line 20):

   ```
   uboot_overlay_addr4=/lib/firmware/BB-I2S-MIC-00A0.dtbo
   ```

3. Ensure audio is NOT disabled (line should be commented):

   ```
   #disable_uboot_overlay_audio=1
   ```

4. Reboot:
   ```bash
   sudo reboot
   ```

## Troubleshooting

### Overlay Won't Load

**Check slots:**

```bash
cat /sys/devices/platform/bone_capemgr/slots
```

**Check dmesg for errors:**

```bash
dmesg | grep -i "i2s\|mcasp\|audio"
```

**Common issues:**

- Overlay file not in `/lib/firmware/`
- Pin conflict with other overlays
- Syntax error in device tree

### No Audio Captured

**Check wiring:**

- VDD → P9_3 (3.3V)
- GND → P9_1
- SD → P9_28
- WS → P9_29
- SCK → P9_31
- L/R → GND

**Check ALSA device:**

```bash
arecord -l
```

**Test with verbose output:**

```bash
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 5 -vv test.wav
```

**Check McASP status:**

```bash
dmesg | tail -30
```

### Silent/Zero Audio

**Possible causes:**

- Microphone not powered (check VDD connection)
- L/R pin not connected to GND
- Wrong data pin (must be P9_28)
- Damaged microphone

**Verify power:**

```bash
# Check 3.3V rail
cat /sys/class/gpio/export
```

### Distorted Audio

**Possible causes:**

- Loose wiring connections
- Long wire runs (>20cm)
- EMI from power supply
- Sample rate mismatch

**Try:**

- Secure all connections
- Shorten wires
- Move away from power supplies
- Test different sample rates (32kHz, 44.1kHz)

## Advanced Configuration

### Change Sample Rate

Edit `BB-I2S-MIC-00A0.dts`:

```c
simple-audio-card,cpu {
    sound-dai = <&mcasp0>;
    system-clock-frequency = <24576000>;  // Change this
};
```

Clock frequencies:

- 48 kHz: 24576000 (48000 × 512)
- 44.1 kHz: 22579200 (44100 × 512)
- 32 kHz: 16384000 (32000 × 512)

Recompile and reload:

```bash
make compile deploy
```

### Stereo Configuration

For stereo (2 microphones on left and right channels):

1. Wire second INMP441 with L/R → VDD (right channel)
2. Edit device tree:

   ```c
   tdm-slots = <2>;  // Already configured
   ```

3. Capture with 2 channels:
   ```bash
   arecord -D hw:0,0 -f S32_LE -r 48000 -c 2 -d 5 stereo.wav
   ```

## References

- [BeagleBone Black SRM - Section 7.1](https://github.com/beagleboard/beaglebone-black/wiki/System-Reference-Manual)
- [AM335x TRM - Chapter 22 (McASP)](https://www.ti.com/lit/ug/spruh73q/spruh73q.pdf)
- [INMP441 Datasheet](https://invensense.tdk.com/products/digital/inmp441/)
- [Linux Simple Audio Card](https://www.kernel.org/doc/Documentation/devicetree/bindings/sound/simple-card.txt)

## Next Steps

After successfully testing audio capture:

1. ✅ Verify audio quality is good
2. → Proceed to **Phase 2: Go ALSA Integration**
3. → Create Go service to read from ALSA
4. → Write samples to PRU shared memory
5. → Update PRU firmware for I2S sample processing

See `docs/I2S_MICROPHONE_REFACTOR_PLAN.md` for complete implementation plan.
