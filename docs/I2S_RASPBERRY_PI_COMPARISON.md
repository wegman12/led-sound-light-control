# Raspberry Pi vs BeagleBone I2S Clock Generation Comparison

## Executive Summary

**KEY FINDING**: The AM335x hardware DOES support dynamic clock generation for 48kHz audio from internal sources. The missing piece is that the McASP driver doesn't utilize the Linux clock framework's `clk_set_rate()` API like the Raspberry Pi driver does.

## Hardware Capabilities

### Raspberry Pi BCM2835
- **Clock Source**: 19.2 MHz crystal → Programmable PLL with fractional dividers
- **Driver Implementation**: `/sound/soc/bcm/bcm2835-i2s.c`
  - Calls `clk_set_rate(dev->clk, bclk_rate)` at line 429
  - Dynamically programs clock generator for any requested rate
  - Clock framework handles PLL programming automatically

### BeagleBone AM335x
- **Clock Sources Available**:
  - `sys_clkin_ck`: 24 MHz fixed oscillator
  - `dpll_per_ck`: Programmable DPLL (192-960 MHz typical)
  - `dpll_per_m2_ck`: Divided output from DPLL_PER

- **Current mcasp0_fck Definition** (am33xx-clocks.dtsi:40-46):
  ```dts
  mcasp0_fck: mcasp0_fck {
      #clock-cells = <0>;
      compatible = "fixed-factor-clock";
      clocks = <&sys_clkin_ck>;  /* Hardcoded to 24 MHz */
      clock-mult = <1>;
      clock-div = <1>;
  };
  ```

- **DPLL_PER Capabilities** (drivers/clk/ti/dpll.c:71-81):
  ```c
  static const struct clk_ops dpll_no_gate_ck_ops = {
      .recalc_rate = &omap3_dpll_recalc,
      .round_rate = &omap2_dpll_round_rate,
      .set_rate = &omap3_noncore_dpll_set_rate,  /* ← SUPPORTS RATE CHANGES */
      .set_parent = &omap3_noncore_dpll_set_parent,
      .determine_rate = &omap3_noncore_dpll_determine_rate,
  };
  ```

- **Driver Implementation**: `/sound/soc/ti/davinci-mcasp.c`
  - Does NOT call `clk_set_rate()`
  - Only calculates dividers for fixed input clock
  - Line 1769 warning: "Update the bindings to use assigned-clocks!"

## The Problem

1. **mcasp0_fck is a fixed-factor-clock**: Cannot be reparented or rate-adjusted
2. **Driver doesn't call clk_set_rate()**: Even if we had a variable clock source
3. **Clock hierarchy is hardcoded**: Device tree binds mcasp0_fck directly to 24 MHz source

## Why Raspberry Pi Works and BeagleBone Doesn't

| Aspect | Raspberry Pi | BeagleBone (Current) |
|--------|-------------|----------------------|
| Clock source type | Programmable PLL | Fixed 24 MHz |
| Driver uses clk_set_rate() | ✅ Yes (line 429) | ❌ No |
| Can generate arbitrary rates | ✅ Yes | ❌ No (only divides 24 MHz) |
| Device tree clock definition | Flexible mux | Fixed-factor passthrough |

## Solution Path

### Option 1: Full Driver Modification (Preferred for Upstream)

This would make BeagleBone work like Raspberry Pi:

**Step 1: Modify Device Tree to Use DPLL_PER**

Create a new clock definition in device tree overlay:
```dts
/* Create a mux clock that can select DPLL_PER output */
mcasp0_fck_mux: mcasp0_fck_mux {
    #clock-cells = <0>;
    compatible = "ti,mux-clock";
    clocks = <&sys_clkin_ck>, <&dpll_per_m2_ck>;
    ti,set-rate-parent;  /* Allow rate changes to propagate to DPLL */
};
```

**Step 2: Modify McASP Driver to Call clk_set_rate()**

Add to `davinci_mcasp_hw_params()` (around line 1280):
```c
static int davinci_mcasp_hw_params(struct snd_pcm_substream *substream,
                                     struct snd_pcm_hw_params *params,
                                     struct snd_soc_dai *cpu_dai)
{
    struct davinci_mcasp *mcasp = snd_soc_dai_get_drvdata(cpu_dai);

    if (mcasp->bclk_master && mcasp->bclk_div == 0 && mcasp->sysclk_freq) {
        int slots = mcasp->tdm_slots;
        int rate = params_rate(params);
        int sbits = params_width(params);
        unsigned int bclk_freq = rate * sbits * slots;
        struct clk *fck;

        /* Get functional clock */
        fck = clk_get(mcasp->dev, "fck");
        if (!IS_ERR(fck)) {
            /* Try to set clock rate */
            int ret = clk_set_rate(fck, bclk_freq * mcasp->bclk_div);
            if (ret) {
                dev_warn(mcasp->dev,
                    "Failed to set clock rate to %d Hz, using fixed rate\n",
                    bclk_freq * mcasp->bclk_div);
            } else {
                /* Update sysclk_freq to actual achieved rate */
                mcasp->sysclk_freq = clk_get_rate(fck);
                dev_info(mcasp->dev, "Set clock rate to %d Hz\n",
                         mcasp->sysclk_freq);
            }
            clk_put(fck);
        }

        /* Continue with existing divider calculation */
        davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                                   rate * sbits * slots, true);
    }
    /* ... rest of function ... */
}
```

### Option 2: Device Tree + assigned-clocks (Quick Test)

Use the existing `assigned-clocks` mechanism the driver already mentions:

```dts
&mcasp0 {
    assigned-clocks = <&dpll_per_m2_ck>;
    assigned-clock-rates = <98304000>;  /* 48kHz * 2048 = good for audio */
    clocks = <&dpll_per_m2_ck>;
};
```

**Limitation**: This sets a fixed rate at boot, doesn't allow dynamic changes per sample rate.

### Option 3: External Oscillator (Fallback)

If software solution proves too complex:
- Add 24.576 MHz external oscillator (as documented in I2S_SOLUTIONS_GUIDE.md)
- Cost: $1-5, Time: 1-2 hours
- Guaranteed to work with existing driver

## Technical Feasibility Assessment

### Can AM335x Generate 48kHz from Internal Clocks?

**YES** - TI E2E Forum confirms: "24MHz/500 = 48KHz"

But more importantly:
- DPLL_PER can generate 192 MHz (192000000 / 4000 = 48000 Hz)
- Or 245.76 MHz (optimal for audio: 48000 * 5120 = 245.76 MHz)
- Or 98.304 MHz (48000 * 2048 = 98.304 MHz)

The `dpll_no_gate_ck_ops` struct proves the hardware supports `.set_rate`.

### Why Hasn't This Been Done Before?

1. **Different Use Cases**: Most AM335x audio applications use codec chips that provide their own clocks (CBM_CFM mode)
2. **Legacy Hardware**: Existing designs (like BeagleBone HDMI audio) use external 24.576 MHz oscillators
3. **Driver Complexity**: Adding clock framework integration requires understanding both ALSA and TI clock subsystems
4. **The 24 MHz Limitation Myth**: TI documentation emphasizes "cannot generate 44.1kHz from 24MHz", leading developers to assume external oscillators are required

### Estimated Development Effort

| Option | Effort | Risk | Upstream Potential |
|--------|--------|------|-------------------|
| Option 1: Full driver mod | 2-3 days | Medium | High - benefits all AM335x users |
| Option 2: assigned-clocks test | 1-2 hours | Low | Low - just a workaround |
| Option 3: External oscillator | 1-2 hours | None | N/A - hardware solution |

## References

### Key Source Files
- **AM335x Clock Definitions**: `/arch/arm/boot/dts/am33xx-clocks.dtsi:40,260-274`
- **DPLL Operations**: `/drivers/clk/ti/dpll.c:71-81,639-641`
- **McASP Driver**: `/sound/soc/ti/davinci-mcasp.c:708-749,1279-1289`
- **BCM2835 Reference**: `/sound/soc/bcm/bcm2835-i2s.c:419-436` (Raspberry Pi)

### Documentation
- TI AM335x Technical Reference Manual: Section 22 (McASP), Section 8 (Clock Management)
- Linux Clock Framework: `Documentation/driver-api/clk.rst`
- TI E2E Forum: "AM335x Audio Clock Generation"

## Next Steps

To implement Option 1 (recommended):
1. Create device tree overlay with mux clock definition
2. Patch davinci-mcasp.c to add clk_set_rate() call
3. Build and test modified kernel module
4. Verify clock generation with oscilloscope
5. Test audio recording functionality
6. Submit patch to upstream Linux kernel

To test Option 2 (quick validation):
1. Add assigned-clocks properties to device tree
2. Recompile and load overlay
3. Check dmesg for clock rate
4. Test if audio capture works
