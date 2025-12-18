# Current Status: Driver Modification Implementation

**Date**: 2025-12-18
**Status**: Modified driver built and installed, but not seeing expected behavior

---

## What We've Accomplished

### ✅ Code Modifications Complete

Successfully applied all 4 modifications to `/tmp/davinci-mcasp-fresh.c`:

1. **Struct fields added** (line 81-82):
   ```c
   struct clk *fck;		/* Functional clock for rate setting */
   bool clk_active;		/* Track if clock is enabled */
   ```

2. **Clock rate function added** (after line 751):
   ```c
   static int davinci_mcasp_set_sysclk_rate(struct davinci_mcasp *mcasp,
                                             unsigned int bclk_freq)
   {
       // Calls clk_set_rate() to dynamically adjust DPLL_PER
       // Target = bclk_freq * 32
   }
   ```

3. **hw_params modified** (line 1321):
   - Removed `&& mcasp->sysclk_freq` condition
   - Added call to `davinci_mcasp_set_sysclk_rate()`
   - Wrapped divider calculation in `if (mcasp->sysclk_freq)`

4. **Probe function modified** (line 2471):
   ```c
   mcasp->fck = devm_clk_get(&pdev->dev, "fck");
   if (IS_ERR(mcasp->fck)) {
       dev_info(&pdev->dev, "Functional clock not available, using fixed rate\n");
       mcasp->fck = NULL;
   }
   mcasp->clk_active = false;
   ```

### ✅ Build and Installation

```bash
# Built on BeagleBone
cd ~/mcasp-clkrate-build
make clean && make
# Result: snd-soc-davinci-mcasp.ko (41KB)

# Installed to system location
sudo cp ~/mcasp-clkrate-build/snd-soc-davinci-mcasp.ko \
  /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/
sudo depmod -a

# Verified our code is present:
strings /lib/modules/.../snd-soc-davinci-mcasp.ko | grep "Set system clock"
# Output: "Set system clock to %lu Hz (target %u Hz)"
```

### ✅ System State After Reboot

- Device tree overlay loaded: `BB-I2S-MIC-00A0.dtbo` (configured in /boot/uEnv.txt)
- Audio device present: `arecord -l` shows "I2S-Microphone"
- Module loaded: `lsmod | grep mcasp` shows `snd_soc_davinci_mcasp`
- Recording works: `arecord` runs without errors

---

## Current Problem

### ❌ Audio Data is All Zeros

Recording produces valid WAV file but contains only zeros:
```bash
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 3 /tmp/test.wav
hexdump -C /tmp/test.wav | head -30
# Shows: 00 00 00 00 00 00 00 00 ...
```

### ❓ No Debug Messages in dmesg

Expected messages NOT appearing:
- "DEBUG: About to get functional clock"
- "Functional clock not available, using fixed rate"
- "Functional clock acquired successfully"
- "Set system clock to X Hz (target Y Hz)"
- "Using fixed system clock"

Only seeing:
```
[   68.319070] davinci-mcasp 48038000.mcasp: IRQ common not found
```

---

## Hypotheses

### Hypothesis 1: Module Not Actually Running
- Possible the stock kernel module is being loaded instead of our modified one
- Even though we installed to `/lib/modules/.../ti/`, there might be compression or caching

### Hypothesis 2: Clock Lookup Name Wrong
- `devm_clk_get(&pdev->dev, "fck")` might be failing silently
- Device tree might not expose clock with name "fck"
- AM335x might use different clock name

### Hypothesis 3: Code Path Not Executing
- Maybe `mcasp->bclk_master` is false
- Maybe `mcasp->bclk_div != 0` (already set)
- hw_params might not be called if device is pre-configured

### Hypothesis 4: Original 24 MHz Problem Still Exists
- Even if clock code runs, 24 MHz oscillator fundamentally incompatible
- Would need to see clock rate change messages to confirm

---

## Next Debugging Steps

### Step 1: Verify Correct Module is Running

```bash
ssh bbb2.wegman

# Check if compressed module is being used
ls -lh /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.*
# Might need to remove .ko.xz compressed version

# Force uncompressed module
sudo rm /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz
sudo depmod -a
sudo reboot
```

### Step 2: Check Device Tree Clock Configuration

```bash
# On BeagleBone
cat /sys/firmware/devicetree/base/ocp/interconnect@48000000/segment@0/target-module@38000/mcasp@0/clock-names
# Or check what clocks are available
ls -l /sys/class/clk/ | grep -i mcasp
```

### Step 3: Add More Debug Output

Add debug messages at more strategic points:
- Beginning of probe function: "DEBUG: davinci_mcasp_probe started"
- Beginning of hw_params: "DEBUG: hw_params called"
- Before checking bclk_master condition: "DEBUG: bclk_master=%d bclk_div=%d"

### Step 4: Check Clock Tree

```bash
# On BeagleBone
cat /sys/kernel/debug/clk/clk_summary | grep -i mcasp
# Or
cat /sys/kernel/debug/clk/clk_summary | grep -i dpll_per
```

### Step 5: Verify Device Tree Overlay

```bash
# Check if overlay actually applied
cat /proc/device-tree/chosen/overlays/0
# Check McASP device tree node
find /sys/firmware/devicetree/base -name "*mcasp*" -type d
```

---

## Key File Locations

### On Development Machine
- Modified source: `/tmp/davinci-mcasp-fresh.c` (2577 lines)
- Quick reference: `/tmp/quick-edits.txt`
- Manual guide: `docs/MANUAL_EDIT_GUIDE.md`
- Original source: `/home/kevin/repositories/external/linux_kernel/beagleboard/sound/soc/ti/davinci-mcasp.c`

### On BeagleBone (bbb2.wegman)
- Build directory: `~/mcasp-clkrate-build/`
- Modified module: `~/mcasp-clkrate-build/snd-soc-davinci-mcasp.ko`
- Installed module: `/lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko`
- Original backup: `/lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.orig`
- Compressed original: `/lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz`
- Device tree overlay: `/lib/firmware/BB-I2S-MIC-00A0.dtbo`
- Boot config: `/boot/uEnv.txt` (has `uboot_overlay_addr4=BB-I2S-MIC-00A0.dtbo`)

---

## Quick Commands Reference

### Rebuild and Install
```bash
# On development machine
scp /tmp/davinci-mcasp-fresh.c bbb2.wegman:~/mcasp-clkrate-build/davinci-mcasp.c

# On BeagleBone
cd ~/mcasp-clkrate-build
make clean && make
sudo cp snd-soc-davinci-mcasp.ko /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/
sudo rm /lib/modules/5.10.168-ti-r83/kernel/sound/soc/ti/snd-soc-davinci-mcasp.ko.xz  # Remove compressed
sudo depmod -a
sudo reboot
```

### Test After Reboot
```bash
# Wait for boot
sleep 45 && ssh bbb2.wegman "uptime"

# Check messages
ssh bbb2.wegman "dmesg | grep -i 'mcasp\|functional\|clock rate'"

# Test recording
ssh bbb2.wegman "arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 3 /tmp/test.wav"

# Check data
ssh bbb2.wegman "hexdump -C /tmp/test.wav | head -30"
```

---

## Alternative Approaches if Debugging Fails

### Option 1: Try Different Clock Name
Instead of "fck", try:
- "ahclkx" (McASP transmit clock)
- "ahclkr" (McASP receive clock)
- NULL (get default clock)

### Option 2: Direct Clock Manipulation
Bypass devm_clk_get and directly access DPLL_PER registers (less portable)

### Option 3: External Oscillator
Order 24.576 MHz oscillator as originally discussed:
- Digikey: Abracon ASFLMB-24.576MHZ-LC-T
- Should be $1-5, arrives in 1-2 days
- Guaranteed to work, no software changes needed

### Option 4: Check if Simple-Card Driver Issues
The simple-card driver might be overriding our settings. Check:
```bash
cat /sys/kernel/debug/asoc/simple-card/dapm/davinci-mcasp.0-dir-hifi
```

---

## Technical Notes

### Why We Expect This to Work

1. **BCM2835 comparison**: Raspberry Pi's I2S driver does exactly this - uses `clk_set_rate()` to dynamically program PLLs
2. **AM335x capability**: DPLL_PER supports `clk_set_rate()` according to clock tree documentation
3. **No hardware changes**: This is purely software, driver just needs to request the rate
4. **Backward compatible**: If clock not available, falls back to fixed rate

### Why Audio Might Still Be Zeros

Even with correct clock programming:
1. **I2S mode mismatch**: MEMS mic might expect different I2S format
2. **Pin configuration**: BCLK/WS not actually outputting
3. **Clock dividers**: Even with right sysclk, dividers might be wrong
4. **Power/enable**: MEMS mic might need power-enable signal

### Success Criteria

We'll know it's working when we see:
1. ✅ "Set system clock to X Hz" in dmesg (proves our code runs)
2. ✅ Non-zero audio data in recording
3. ✅ Correct clock frequency on oscilloscope (if available)

---

## Current Environment

- **Board**: BeagleBone Black (AM3358, rev A6)
- **Kernel**: 5.10.168-ti-r83
- **MEMS Mic**: SPH0645 (I2S output)
- **Original Clock**: 24 MHz (wrong - need 24.576 MHz or DPLL-derived)
- **Target Sample Rate**: 48 kHz
- **Format**: S32_LE (32-bit signed little-endian)

---

## Resume Instructions

When resuming this work:

1. **Read this document** to understand current state
2. **Follow Step 1** under "Next Debugging Steps" first
3. **Check if compressed .ko.xz is problem** - very likely culprit
4. **Add more debug output** if still no messages
5. **Consider external oscillator** if debugging becomes too time-consuming

The modified driver source is ready at `/tmp/davinci-mcasp-fresh.c` and can be rebuilt/reinstalled anytime.
