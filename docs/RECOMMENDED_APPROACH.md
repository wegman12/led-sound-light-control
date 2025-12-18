# Recommended Approach: Internal Clock Generation

## Summary of Findings

After analyzing the Raspberry Pi driver and AM335x clock framework, I've confirmed:

✅ **AM335x CAN generate 48kHz audio clocks from internal sources**
✅ **DPLL_PER supports runtime rate changes** (has `.set_rate` callback)
✅ **The hardware capability exists** - it's just not being used

The difference between Raspberry Pi (working) and BeagleBone (not working):

| Component | Raspberry Pi | BeagleBone |
|-----------|-------------|------------|
| Hardware capability | Programmable PLL | Programmable DPLL ✅ |
| Device tree clock type | Flexible | Fixed 24 MHz ❌ |
| Driver calls clk_set_rate() | Yes ✅ | No ❌ |

## Quick Test: Option 2 (Assigned-Clocks)

**Time: ~30 minutes**
**Risk: Low (easily reversible)**

This tests if we can use DPLL_PER without modifying the driver.

### Implementation

Modify `hardware/device-tree/BB-I2S-MIC-00A0.dts`:

```dts
/* McASP0 configuration */
fragment@1 {
    target = <&mcasp0>;
    __overlay__ {
        #sound-dai-cells = <0>;
        pinctrl-names = "default";
        pinctrl-0 = <&i2s_pins>;
        status = "okay";

        /* USE DPLL_PER DIVIDED OUTPUT INSTEAD OF FIXED 24 MHz */
        assigned-clocks = <&dpll_per_m2_ck>;
        assigned-clock-rates = <98304000>;  /* 48000 Hz * 2048 = 98.304 MHz */

        op-mode = <0>;  /* MCASP_IIS_MODE */
        tdm-slots = <2>;
        slot-width = <32>;
        num-serializer = <16>;

        serial-dir = <
            2 0 0 0  /* AXR0 = RX */
            0 0 0 0
            0 0 0 0
            0 0 0 0
        >;

        rx-num-evt = <32>;
    };
};

/* Audio card configuration */
fragment@2 {
    target-path = "/";
    __overlay__ {
        sound {
            compatible = "simple-audio-card";
            simple-audio-card,name = "I2S-Microphone";
            simple-audio-card,format = "i2s";
            simple-audio-card,bitclock-master = <&cpu_master>;
            simple-audio-card,frame-master = <&cpu_master>;

            cpu_master: simple-audio-card,cpu {
                sound-dai = <&mcasp0>;
                /* Tell driver to use 98.304 MHz as system clock */
                system-clock-frequency = <98304000>;
            };

            simple-audio-card,codec {
                sound-dai = <&codec>;
            };
        };

        codec: dmic_codec {
            #sound-dai-cells = <0>;
            compatible = "dmic-codec";
            num-channels = <1>;
            wakeup-delay-ms = <50>;
            status = "okay";
        };
    };
};
```

### Testing Procedure

1. **Backup current overlay**:
   ```bash
   cp hardware/device-tree/BB-I2S-MIC-00A0.dts hardware/device-tree/BB-I2S-MIC-00A0.dts.backup
   ```

2. **Apply changes and compile**:
   ```bash
   # Edit the file with changes above
   dtc -O dtb -o /lib/firmware/BB-I2S-MIC-00A0.dtbo -b 0 -@ hardware/device-tree/BB-I2S-MIC-00A0.dts
   ```

3. **Load and test**:
   ```bash
   # Unload old overlay
   sudo rmdir /sys/kernel/config/device-tree/overlays/BB-I2S-MIC 2>/dev/null

   # Load new overlay
   sudo mkdir /sys/kernel/config/device-tree/overlays/BB-I2S-MIC
   sudo su -c 'echo BB-I2S-MIC-00A0.dtbo > /sys/kernel/config/device-tree/overlays/BB-I2S-MIC/path'

   # Check dmesg for clock messages
   dmesg | grep -i "mcasp\|clock\|dpll"

   # Try recording
   arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 test.wav
   ```

4. **Check with oscilloscope** (if available):
   - P9_31 (BCLK): Should show ~3.072 MHz (48000 Hz * 64)
   - P9_29 (WS): Should show 48 kHz square wave

### Expected Outcomes

**Success Case**:
- dmesg shows: "assigned clock rate 98304000 Hz"
- BCLK/WS signals present on pins
- arecord successfully captures audio

**Failure Case**:
- dmesg shows: "failed to set assigned clock rate"
- OR clocks still not generated
- **Next step**: Proceed to Option 1 (driver modification)

## Full Solution: Option 1 (Driver Modification)

**Time: 2-3 days**
**Risk: Medium (requires kernel development)**

If Option 2 doesn't work, we need to modify the driver to call `clk_set_rate()`.

### Why This Might Be Necessary

The current driver calculates clock dividers but never asks the clock framework to actually change the rate:

```c
// Current code (davinci-mcasp.c:1287)
davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                           rate * sbits * slots, true);
// Calculates dividers for FIXED sysclk_freq, doesn't change the source clock
```

Raspberry Pi driver actively programs the clock:

```c
// BCM2835 code (bcm2835-i2s.c:429)
ret = clk_set_rate(dev->clk, bclk_rate);
// Actually changes the PLL frequency
```

### Implementation Steps

1. Clone and modify davinci-mcasp.c
2. Add clk_set_rate() call in hw_params()
3. Build custom kernel module
4. Test with our hardware
5. Submit patch to upstream Linux kernel

**Detailed implementation in I2S_RASPBERRY_PI_COMPARISON.md**

## Fallback: Option 3 (External Oscillator)

**Time: 1-2 hours**
**Cost: $1-5**
**Risk: None (proven solution)**

If software approaches prove too complex:
- Follow I2S_SOLUTIONS_GUIDE.md Section 1
- Purchase 24.576 MHz oscillator
- Connect to P9_12 (AHCLKX)
- Works with existing driver, no modifications needed

## My Recommendation

**Start with Option 2** (assigned-clocks):
- Takes 30 minutes to test
- No kernel modification required
- Will immediately show if DPLL_PER can be used
- If it works, we get dynamic clocking without driver changes
- If it doesn't work, we learn what's blocking it

**If Option 2 fails or only works partially**:
- Proceed to Option 1 (driver modification)
- We'll have learned exactly what the driver needs
- Could become an upstream contribution

**External oscillator remains available**:
- If time-constrained, go with proven hardware solution
- Can always revisit software solution later

## Files Modified

- `hardware/device-tree/BB-I2S-MIC-00A0.dts` - Device tree overlay
- Documentation:
  - `docs/I2S_RASPBERRY_PI_COMPARISON.md` - Technical analysis
  - `docs/RECOMMENDED_APPROACH.md` - This file

## Next Steps

Ready to proceed with Option 2 test when you are. Let me know if you want to:
1. Try the assigned-clocks approach
2. Go straight to driver modification
3. Use external oscillator instead
