# Option B Implementation Status

## What We've Accomplished

### ✅ Complete Analysis
- Identified exactly where and how to add clk_set_rate() support
- Compared with BCM2835 driver implementation
- Understood AM335x clock framework capabilities
- Confirmed DPLL_PER supports dynamic rate changes

### ✅ Implementation Design
- Created detailed implementation plan (DRIVER_MODIFICATION_PLAN.md)
- Designed backward-compatible approach
- Identified all code locations requiring changes
- Created patch file documenting changes

### ✅ Build Infrastructure
- Created build scripts for BeagleBone
- Set up header file copying
- Created Makefile for module compilation

### ⚠️ Current Blocker
**Automated patching script has regex issues** causing syntax errors in generated code.

## The Changes Needed (Manual Reference)

### 1. Add Fields to struct davinci_mcasp (line ~80)
```c
struct davinci_mcasp {
    struct snd_dmaengine_dai_dma_data dma_data[2];
    void __iomem *base;
    u32 fifo_base;
    struct clk *fck;        /* ADD THIS */
    bool clk_active;        /* ADD THIS */
    struct device *dev;
    // ... rest of struct
};
```

### 2. Add Clock Rate Function (after davinci_mcasp_set_sysclk, line ~750)
```c
static int davinci_mcasp_set_sysclk_rate(struct davinci_mcasp *mcasp,
                                          unsigned int bclk_freq)
{
    unsigned int target_sysclk;
    unsigned long achieved_rate;
    int ret;

    if (!mcasp->fck || IS_ERR(mcasp->fck))
        return 0;

    target_sysclk = bclk_freq * 32;

    ret = clk_set_rate(mcasp->fck, target_sysclk);
    if (ret < 0) {
        dev_warn(mcasp->dev, "Failed to set clock: %d\n", ret);
        return ret;
    }

    achieved_rate = clk_get_rate(mcasp->fck);
    mcasp->sysclk_freq = achieved_rate;
    dev_info(mcasp->dev, "Set clock to %lu Hz\n", achieved_rate);

    return 0;
}
```

### 3. Modify hw_params Function (line ~1279)
BEFORE:
```c
if (mcasp->bclk_master && mcasp->bclk_div == 0 && mcasp->sysclk_freq) {
    davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                               rate * sbits * slots, true);
}
```

AFTER:
```c
if (mcasp->bclk_master && mcasp->bclk_div == 0) {
    unsigned int bclk_freq = rate * sbits * slots;

    /* Try to set optimal clock rate */
    davinci_mcasp_set_sysclk_rate(mcasp, bclk_freq);

    /* Calculate dividers from actual clock */
    if (mcasp->sysclk_freq) {
        davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
                                   bclk_freq, true);
    }
}
```

### 4. Get Clock in Probe Function (line ~2640)
Add before devm_snd_soc_register_component():
```c
/* Get functional clock for dynamic rate adjustment */
mcasp->fck = devm_clk_get(&pdev->dev, "fck");
if (IS_ERR(mcasp->fck)) {
    dev_info(&pdev->dev, "Using fixed clock\n");
    mcasp->fck = NULL;
}
mcasp->clk_active = false;
```

## Alternative: Quick Manual Edit on BeagleBone

Since automated patching has issues, you can:

1. SSH to BeagleBone
2. Copy original driver to build directory
3. Edit with vi/nano to add the 4 changes above
4. Build with our Makefile

```bash
ssh bbb2.wegman
cd ~/mcasp-clkrate-build
# Copy original if needed
cp ~/led-sound-light-control/external/linux_kernel/beagleboard/sound/soc/ti/davinci-mcasp.c .
# Edit the file
vi davinci-mcasp.c
# Make the 4 changes listed above
# Build
make
```

## Recommendation

Given the time invested and the patching complexity:

1. **For immediate results**: Order 24.576 MHz oscillator ($1-5)
   - Digikey: Abracon ASFLMB-24.576MHZ-LC-T
   - Mouser: ECS-2x6-S4
   - Amazon: Generic 24.576MHz oscillator modules

2. **For learning/contribution**: Continue with manual driver modification
   - The changes are well-documented above
   - All 4 modifications are straightforward
   - Can be done in 30 minutes with manual editing

3. **Best of both**: Order oscillator AND finish driver mod
   - Get working audio quickly
   - Still contribute driver improvements to community
   - Learn kernel development in the process

## Files in This Repository

- `docs/DRIVER_MODIFICATION_PLAN.md` - Detailed implementation guide
- `docs/I2S_RASPBERRY_PI_COMPARISON.md` - Technical analysis
- `patches/0001-davinci-mcasp-add-dynamic-clock-rate-support.patch` - Reference patch
- `scripts/build-modified-driver-v2.sh` - Build automation (needs working source)
- `hardware/device-tree/BB-I2S-MIC-00A0.dts` - Updated for dynamic clocks

## What's Tested and Working

✅ Device tree overlay compiles and loads
✅ Audio device appears in arecord -l
✅ Recording starts without errors
✅ Confirmed 24 MHz doesn't generate clocks (all zeros recorded)
✅ Clock framework supports clk_set_rate() on DPLL_PER

## What Still Needs Testing

⏳ Modified driver with clk_set_rate() compiled and loaded
⏳ Verify DPLL_PER actually changes rate when requested
⏳ Confirm I2S clocks appear on pins
⏳ Verify non-zero audio data captured

## Time Estimate to Complete

- **Manual driver edit approach**: 30-60 minutes
- **Debug Python script approach**: 2-3 hours
- **External oscillator approach**: 1-2 hours (after parts arrive)

The manual edit is actually fastest at this point.
