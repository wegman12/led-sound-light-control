# Driver Modification Plan: Add clk_set_rate() Support

## Objective
Modify `davinci-mcasp.c` to dynamically configure DPLL_PER clock rate like the Raspberry Pi BCM2835 driver does, enabling internal clock generation for audio without external oscillator.

## Analysis

### Current Behavior
1. Driver gets system clock frequency from device tree (`system-clock-frequency` property)
2. Calculates integer dividers to generate BCLK from fixed system clock
3. Never changes the system clock rate
4. Limited to frequencies that divide evenly from 24 MHz

### Target Behavior (Like BCM2835)
1. Driver calculates required system clock rate based on sample rate
2. Calls `clk_set_rate()` to program DPLL_PER to desired frequency
3. Verifies achieved rate with `clk_get_rate()`
4. Calculates dividers from actual achieved rate
5. Works with any sample rate

### Code Locations

**davinci-mcasp.c:1279-1289** - Clock divider calculation in `hw_params()`
```c
if (mcasp->bclk_master && mcasp->bclk_div == 0 && mcasp->sysclk_freq) {
    int slots = mcasp->tdm_slots;
    int rate = params_rate(params);
    int sbits = params_width(params);
    davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                               rate * sbits * slots, true);
}
```

**davinci-mcasp.c:77-126** - Structure definition (needs clock pointer)

**davinci-mcasp.c:1755-1795** - `mcasp_reparent_fck()` (handles clock setup)

## Implementation Steps

### Step 1: Add Clock Pointer to Structure

Add to `struct davinci_mcasp`:
```c
struct clk *fck;  /* Functional clock for rate setting */
```

### Step 2: Get Clock Reference in Probe

In probe function (~line 2100), after device initialization:
```c
/* Get functional clock for dynamic rate adjustment */
mcasp->fck = devm_clk_get(&pdev->dev, "fck");
if (IS_ERR(mcasp->fck)) {
    /* Not fatal - will use fixed clock */
    dev_info(&pdev->dev, "Cannot get fck, using fixed clock\n");
    mcasp->fck = NULL;
}
```

### Step 3: Add Clock Rate Setting in hw_params()

Replace lines 1279-1289 with:
```c
if (mcasp->bclk_master && mcasp->bclk_div == 0) {
    int slots = mcasp->tdm_slots;
    int rate = params_rate(params);
    int sbits = params_width(params);
    unsigned int bclk_freq = rate * sbits * slots;
    unsigned int target_sysclk;

    if (mcasp->slot_width)
        sbits = mcasp->slot_width;

    /* Try to set optimal system clock rate if we have clock control */
    if (mcasp->fck && !IS_ERR(mcasp->fck)) {
        /* Calculate ideal system clock (BCLK * reasonable divider) */
        /* Use 32 as divider for good margin */
        target_sysclk = bclk_freq * 32;

        ret = clk_set_rate(mcasp->fck, target_sysclk);
        if (ret) {
            dev_warn(mcasp->dev,
                "Failed to set clock rate to %u Hz: %d\n",
                target_sysclk, ret);
            /* Continue with fixed clock */
        } else {
            /* Update sysclk_freq with actual achieved rate */
            mcasp->sysclk_freq = clk_get_rate(mcasp->fck);
            dev_info(mcasp->dev,
                "Set system clock to %u Hz (requested %u Hz)\n",
                mcasp->sysclk_freq, target_sysclk);
        }
    }

    /* Calculate dividers from actual system clock */
    if (mcasp->sysclk_freq) {
        davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                                   bclk_freq, true);
    }
}
```

### Step 4: Enable Clock Before Use

In `hw_params()`, before setting rate:
```c
if (mcasp->fck && !IS_ERR(mcasp->fck)) {
    ret = clk_prepare_enable(mcasp->fck);
    if (ret) {
        dev_err(mcasp->dev, "Failed to enable fck: %d\n", ret);
        return ret;
    }
}
```

And in cleanup path or `hw_free()`:
```c
if (mcasp->fck && !IS_ERR(mcasp->fck))
    clk_disable_unprepare(mcasp->fck);
```

## Device Tree Changes

Update device tree to allow clock rate changes:

### Option A: Use fck_parent (legacy but supported)
```dts
&mcasp0 {
    fck_parent = "dpll_per_m2_ck";
    /* Driver will issue warning but will work */
};
```

### Option B: Use assigned-clocks (modern approach)
This requires changes to base device tree (am33xx.dtsi), not overlay:
```dts
&mcasp0 {
    clocks = <&dpll_per_m2_ck>;
    assigned-clocks = <&dpll_per_m2_ck>;
    assigned-clock-parents = <&dpll_per_ck>;
};
```

### Option C: Create new clock definition (most flexible)
In overlay:
```dts
clk_mcasp0: clk_mcasp0 {
    #clock-cells = <0>;
    compatible = "fixed-factor-clock";
    clocks = <&dpll_per_m2_ck>;
    clock-mult = <1>;
    clock-div = <1>;
};

&mcasp0 {
    clocks = <&clk_mcasp0>;
};
```

## Testing Strategy

### Phase 1: Verify Clock Framework Integration
1. Build modified driver
2. Add debug printk() to show:
   - Clock pointer obtained successfully
   - Requested vs achieved rates
   - Divider calculations
3. Load module and check dmesg

### Phase 2: Test Sample Rates
Test multiple sample rates to verify flexibility:
- 48000 Hz (most common)
- 44100 Hz (CD quality)
- 96000 Hz (high-res)
- 8000 Hz (telephony)

Each should result in appropriate DPLL programming.

### Phase 3: Verify Hardware Output
- Use oscilloscope to measure BCLK frequency
- Or record audio and verify non-zero data
- Check McASP registers show correct dividers

## Expected Clock Rates

| Sample Rate | BCLK (64×) | Optimal DPLL (÷32) | Actual DPLL |
|-------------|------------|-------------------|-------------|
| 8 kHz | 512 kHz | 16.384 MHz | ~16 MHz |
| 44.1 kHz | 2.8224 MHz | 90.3168 MHz | ~90 MHz |
| 48 kHz | 3.072 MHz | 98.304 MHz | ~98 MHz |
| 96 kHz | 6.144 MHz | 196.608 MHz | ~196 MHz |

DPLL_PER supports 192-960 MHz range, so all these are achievable.

## Error Handling

### If clk_set_rate() fails:
- Log warning but continue
- Fall back to fixed clock behavior
- User sees message: "using fixed clock"

### If clk_get() fails:
- Store NULL in mcasp->fck
- Skip all clock rate setting
- Behave exactly like current driver

### If achieved rate is too far from target:
- Calculate actual BCLK frequency
- Check if divider produces acceptable PPM error
- Warn if audio quality may be affected

## Benefits

1. **No external hardware required** - Works with internal clocks
2. **Flexible sample rates** - Not limited to 24 MHz divisions
3. **Backward compatible** - Falls back to fixed clock if needed
4. **Upstream potential** - Benefits all AM335x audio users
5. **Educational** - Learn kernel clock framework

## Risks and Mitigation

### Risk: DPLL_PER shared with other peripherals
**Mitigation**: Check if other devices use DPLL_PER, may need separate DPLL output

### Risk: Clock changes affect power consumption
**Mitigation**: Only change clock when audio active, restore on close

### Risk: Some clock rates not achievable
**Mitigation**: Use `clk_round_rate()` to check before setting

### Risk: Breaks existing users
**Mitigation**: Make feature opt-in via device tree property

## Files to Modify

1. **sound/soc/ti/davinci-mcasp.c** - Main driver file
2. **hardware/device-tree/BB-I2S-MIC-00A0.dts** - Device tree overlay
3. **docs/** - Documentation for changes

## Next Steps

1. Create patch file with modifications
2. Build on BeagleBone
3. Test with debug output
4. Verify clock rates
5. Test audio recording
6. Document results
7. Prepare for upstream submission

## References

- BCM2835 I2S driver: `sound/soc/bcm/bcm2835-i2s.c:419-436`
- Linux Clock Framework: `Documentation/driver-api/clk.rst`
- TI DPLL driver: `drivers/clk/ti/dpll.c`
- AM335x TRM: Section 22 (McASP), Section 8 (PRCM)
