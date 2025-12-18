# Manual Driver Edit Guide

## Step-by-Step Instructions

### Preparation

1. SSH to BeagleBone:
   ```bash
   ssh bbb2.wegman
   ```

2. Navigate to build directory:
   ```bash
   cd ~/mcasp-clkrate-build
   ```

3. Check if we need to get the original source:
   ```bash
   ls -l davinci-mcasp.c
   ```

   If not there, we'll need to find the kernel source.

### Change 1: Add Clock Fields to Struct (Line ~80)

**Location**: In `struct davinci_mcasp` definition

**Find this** (around line 77-80):
```c
struct davinci_mcasp {
	struct snd_dmaengine_dai_dma_data dma_data[2];
	void __iomem *base;
	u32 fifo_base;
	struct device *dev;
```

**Add these two lines after `u32 fifo_base;`**:
```c
	struct clk *fck;		/* Functional clock for rate setting */
	bool clk_active;		/* Track if clock is enabled */
```

**Result should look like**:
```c
struct davinci_mcasp {
	struct snd_dmaengine_dai_dma_data dma_data[2];
	void __iomem *base;
	u32 fifo_base;
	struct clk *fck;		/* Functional clock for rate setting */
	bool clk_active;		/* Track if clock is enabled */
	struct device *dev;
```

### Change 2: Add Clock Rate Function (After Line ~748)

**Location**: After the `davinci_mcasp_set_sysclk` function (ends around line 748)

**Add this entire function**:
```c
/**
 * davinci_mcasp_set_sysclk_rate - Set system clock to optimal rate
 * @mcasp: McASP device
 * @bclk_freq: Target bit clock frequency
 *
 * Attempts to program the system clock (typically DPLL_PER) to an optimal
 * rate for the target bit clock frequency.
 */
static int davinci_mcasp_set_sysclk_rate(struct davinci_mcasp *mcasp,
					  unsigned int bclk_freq)
{
	unsigned int target_sysclk;
	unsigned long achieved_rate;
	int ret;

	/* Skip if we don't have clock control */
	if (!mcasp->fck || IS_ERR(mcasp->fck))
		return 0;

	/* Calculate target system clock (BCLK * divider)
	 * Use 32 as a reasonable divider for good jitter margin
	 */
	target_sysclk = bclk_freq * 32;

	ret = clk_set_rate(mcasp->fck, target_sysclk);
	if (ret < 0) {
		dev_warn(mcasp->dev,
			 "Failed to set clock rate to %u Hz: %d\n",
			 target_sysclk, ret);
		return ret;
	}

	achieved_rate = clk_get_rate(mcasp->fck);
	mcasp->sysclk_freq = achieved_rate;
	dev_info(mcasp->dev, "Set system clock to %lu Hz (target %u Hz)\n",
		 achieved_rate, target_sysclk);

	return 0;
}
```

### Change 3: Modify hw_params Function (Line ~1279)

**Location**: In `davinci_mcasp_hw_params` function

**Find this** (around line 1275-1289):
```c
	/*
	 * If mcasp is BCLK master, and a BCLK divider was not provided by
	 * the machine driver, we need to calculate the ratio.
	 */
	if (mcasp->bclk_master && mcasp->bclk_div == 0 && mcasp->sysclk_freq) {
		int slots = mcasp->tdm_slots;
		int rate = params_rate(params);
		int sbits = params_width(params);

		if (mcasp->slot_width)
			sbits = mcasp->slot_width;

		davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
					   rate * sbits * slots, true);
	}
```

**Replace with**:
```c
	/*
	 * If mcasp is BCLK master, and a BCLK divider was not provided by
	 * the machine driver, set optimal system clock rate and calculate ratio.
	 */
	if (mcasp->bclk_master && mcasp->bclk_div == 0) {
		int slots = mcasp->tdm_slots;
		int rate = params_rate(params);
		int sbits = params_width(params);
		unsigned int bclk_freq;

		if (mcasp->slot_width)
			sbits = mcasp->slot_width;

		bclk_freq = rate * sbits * slots;

		/* Try to set optimal system clock rate */
		ret = davinci_mcasp_set_sysclk_rate(mcasp, bclk_freq);
		if (ret < 0) {
			dev_info(mcasp->dev, "Using fixed system clock\n");
		}

		/* Calculate dividers from actual system clock */
		if (mcasp->sysclk_freq) {
			davinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
						   bclk_freq, true);
		}
	}
```

**Key changes**:
- Remove `&& mcasp->sysclk_freq` condition
- Add `unsigned int bclk_freq;` variable
- Calculate `bclk_freq = rate * sbits * slots;`
- Call `davinci_mcasp_set_sysclk_rate(mcasp, bclk_freq);`
- Wrap divider calculation in `if (mcasp->sysclk_freq)`

### Change 4: Get Clock in Probe Function (Line ~2640)

**Location**: In `davinci_mcasp_probe` function, before `devm_snd_soc_register_component`

**Find this** (around line 2630-2640):
```c
	ret = devm_snd_soc_register_component(&pdev->dev,
					&davinci_mcasp_component,
					&davinci_mcasp_dai[pdata->op_mode], 1);
```

**Add BEFORE this block**:
```c
	/* Get functional clock reference for dynamic rate adjustment
	 * This is optional - if not available, driver will use fixed clock
	 */
	mcasp->fck = devm_clk_get(&pdev->dev, "fck");
	if (IS_ERR(mcasp->fck)) {
		dev_info(&pdev->dev,
			 "Functional clock not available, using fixed rate\n");
		mcasp->fck = NULL;
	}
	mcasp->clk_active = false;

```

## Building and Testing

After making all 4 changes:

```bash
# Still in ~/mcasp-clkrate-build
make clean
make
```

If successful, you'll see:
```
snd-soc-davinci-mcasp.ko
```

Load the module:
```bash
sudo rmmod snd_soc_davinci_mcasp 2>/dev/null
sudo insmod snd-soc-davinci-mcasp.ko
```

Check dmesg for our messages:
```bash
dmesg | tail -20
```

Look for:
- "Set system clock to X Hz (target Y Hz)"
- Or "Functional clock not available, using fixed rate"

Test recording:
```bash
arecord -D hw:0,0 -f S32_LE -r 48000 -c 1 -d 3 test.wav
hexdump -C test.wav | head -20
```

If successful, you should see **non-zero data** instead of all 00s!

## Vim Tips

If using vi/vim:

- Press `i` to enter insert mode
- Make your edits
- Press `Esc` to exit insert mode
- Type `:wq` to save and quit
- Type `:q!` to quit without saving

To jump to a line number:
- In command mode, type `:80` and press Enter (jumps to line 80)

To search for text:
- In command mode, type `/struct davinci_mcasp` and press Enter

## Common Issues

**Issue**: "missing terminating quote"
**Fix**: Make sure all string literals end with `\n"`

**Issue**: "undeclared variable"
**Fix**: Check that `unsigned int bclk_freq;` was added

**Issue**: Clock doesn't change
**Fix**: Check dmesg for "Failed to set clock rate" message

## Success Indicators

✅ Module compiles without errors
✅ Module loads without errors
✅ dmesg shows "Set system clock to X Hz"
✅ Recording captures non-zero audio data
