#!/usr/bin/env python3
"""
Apply modifications to davinci-mcasp.c to add dynamic clock rate support.
This script programmatically modifies the source file rather than using a patch.
"""

import sys
import re

def add_clock_to_struct(content):
    """Add fck and clk_active fields to struct davinci_mcasp"""
    # Find the struct definition
    struct_pattern = r'(struct davinci_mcasp \{.*?)(void __iomem \*base;)'
    replacement = r'\1\2\n\tstruct clk *fck;\t\t/* Functional clock for rate setting */\n\tbool clk_active;\t\t/* Track if clock is enabled */'

    content = re.sub(struct_pattern, replacement, content, count=1, flags=re.DOTALL)
    return content

def add_set_sysclk_rate_function(content):
    """Add the davinci_mcasp_set_sysclk_rate function"""
    # Find the location after davinci_mcasp_set_sysclk function
    pattern = r'(static int davinci_mcasp_set_sysclk.*?\n\treturn 0;\n\})\n'

    new_function = '''

/**
 * davinci_mcasp_set_sysclk_rate - Set system clock to optimal rate for sample rate
 * @mcasp: McASP device
 * @bclk_freq: Target bit clock frequency
 *
 * Attempts to program the system clock (typically DPLL_PER) to an optimal
 * rate for the target bit clock frequency. Falls back to fixed clock if
 * rate setting is not supported.
 *
 * Returns: 0 on success, negative error code on failure
 */
static int davinci_mcasp_set_sysclk_rate(struct davinci_mcasp *mcasp,
					  unsigned int bclk_freq)
{
\tunsigned int target_sysclk;
\tunsigned long achieved_rate;
\tint ret;

\t/* Skip if we don't have clock control */
\tif (!mcasp->fck || IS_ERR(mcasp->fck))
\t\treturn 0;

\t/* Calculate target system clock (BCLK * divider)
\t * Use 32 as a reasonable divider for good jitter margin
\t */
\ttarget_sysclk = bclk_freq * 32;

\tret = clk_set_rate(mcasp->fck, target_sysclk);
\tif (ret < 0) {
\t\tdev_warn(mcasp->dev,
\t\t\t "Failed to set clock rate to %%u Hz: %%d\\n",
\t\t\t target_sysclk, ret);
\t\treturn ret;
\t}

\tachieved_rate = clk_get_rate(mcasp->fck);
\tmcasp->sysclk_freq = achieved_rate;
\tdev_info(mcasp->dev, "Set system clock to %%lu Hz (target %%u Hz)\\n",
\t\t achieved_rate, target_sysclk);

\treturn 0;
}
'''

    content = re.sub(pattern, r'\1' + new_function + '\n', content, count=1, flags=re.DOTALL)
    return content

def modify_hw_params(content):
    """Modify davinci_mcasp_hw_params to add clock management"""

    # Add clock enable after ret = davinci_mcasp_set_dai_fmt
    pattern1 = r'(\tret = davinci_mcasp_set_dai_fmt\(cpu_dai, mcasp->dai_fmt\);\n\tif \(ret\)\n\t\treturn ret;\n)'

    clock_enable = '''\t/* Enable functional clock if we have it */
\tif (mcasp->fck && !IS_ERR(mcasp->fck) && !mcasp->clk_active) {
\t\tret = clk_prepare_enable(mcasp->fck);
\t\tif (ret) {
\t\t\tdev_err(mcasp->dev, "Failed to enable fck: %%d\\n", ret);
\t\t\treturn ret;
\t\t}
\t\tmcasp->clk_active = true;
\t}

'''

    content = re.sub(pattern1, r'\1\n' + clock_enable, content, count=1)

    # Replace the clock divider calculation section
    pattern2 = r'(\t/\*\n\t \* If mcasp is BCLK master.*?\n\t \*/\n)\tif \(mcasp->bclk_master && mcasp->bclk_div == 0 && mcasp->sysclk_freq\) \{\n\t\tint slots = mcasp->tdm_slots;\n\t\tint rate = params_rate\(params\);\n\t\tint sbits = params_width\(params\);\n\n\t\tif \(mcasp->slot_width\)\n\t\t\tsbits = mcasp->slot_width;\n\n\t\tdavinci_mcasp_calc_clk_div\(mcasp, mcasp->sysclk_freq,\n\t\t\t\t\t   rate \* sbits \* slots, true\);\n\t\}'

    new_section = r'''\1\tif (mcasp->bclk_master && mcasp->bclk_div == 0) {
\t\tint slots = mcasp->tdm_slots;
\t\tint rate = params_rate(params);
\t\tint sbits = params_width(params);
\t\tunsigned int bclk_freq;

\t\tif (mcasp->slot_width)
\t\t\tsbits = mcasp->slot_width;

\t\tbclk_freq = rate * sbits * slots;

\t\t/* Try to set optimal system clock rate */
\t\tret = davinci_mcasp_set_sysclk_rate(mcasp, bclk_freq);
\t\tif (ret < 0) {
\t\t\tdev_info(mcasp->dev, "Using fixed system clock\\n");
\t\t}

\t\t/* Calculate dividers from actual system clock */
\t\tif (mcasp->sysclk_freq) {
\t\t\tdavinci_mcasp_calc_clk_div(mcasp, mcasp->sysclk_freq,
\t\t\t\t\t\t   bclk_freq, true);
\t\t}
\t}'''

    content = re.sub(pattern2, new_section, content, count=1, flags=re.DOTALL)

    return content

def add_hw_free_function(content):
    """Add davinci_mcasp_hw_free function"""
    # Find location after hw_params function
    pattern = r'(static int davinci_mcasp_hw_params.*?\n\treturn 0;\n\})\n\n(static int davinci_mcasp_trigger)'

    hw_free = '''

static int davinci_mcasp_hw_free(struct snd_pcm_substream *substream,
\t\t\t\t  struct snd_soc_dai *cpu_dai)
{
\tstruct davinci_mcasp *mcasp = snd_soc_dai_get_drvdata(cpu_dai);

\t/* Disable functional clock when not in use */
\tif (mcasp->fck && !IS_ERR(mcasp->fck) && mcasp->clk_active) {
\t\tclk_disable_unprepare(mcasp->fck);
\t\tmcasp->clk_active = false;
\t\tdev_dbg(mcasp->dev, "Disabled functional clock\\n");
\t}

\treturn 0;
}
'''

    content = re.sub(pattern, r'\1' + hw_free + '\n\n\2', content, count=1, flags=re.DOTALL)
    return content

def add_hw_free_to_ops(content):
    """Add hw_free to dai_ops structure"""
    pattern = r'(static struct snd_soc_dai_ops davinci_mcasp_dai_ops = \{[^\}]*\.set_sysclk\s*=\s*davinci_mcasp_set_sysclk,)'
    replacement = r'\1\n\t.hw_free\t= davinci_mcasp_hw_free,'

    content = re.sub(pattern, replacement, content, count=1)
    return content

def add_clock_init_in_probe(content):
    """Add clock initialization in probe function"""
    # Find the probe function and add clock initialization
    pattern = r'(static int davinci_mcasp_probe.*?)([\t]ret = devm_snd_soc_register_component)'

    clock_init = '''\n\t/* Get functional clock reference for dynamic rate adjustment */
\tmcasp->fck = devm_clk_get(&pdev->dev, "fck");
\tif (IS_ERR(mcasp->fck)) {
\t\tdev_info(&pdev->dev,
\t\t\t "Functional clock not available, using fixed rate\\n");
\t\tmcasp->fck = NULL;
\t}
\tmcasp->clk_active = false;

\t'''

    content = re.sub(pattern, r'\1' + clock_init + r'\2', content, count=1, flags=re.DOTALL)
    return content

def main():
    if len(sys.argv) != 3:
        print("Usage: apply-driver-modifications.py <input_file> <output_file>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = sys.argv[2]

    print(f"Reading {input_file}...")
    with open(input_file, 'r') as f:
        content = f.read()

    print("Applying modifications...")
    print("  - Adding clock fields to struct...")
    content = add_clock_to_struct(content)

    print("  - Adding davinci_mcasp_set_sysclk_rate function...")
    content = add_set_sysclk_rate_function(content)

    print("  - Modifying hw_params function...")
    content = modify_hw_params(content)

    print("  - Adding hw_free function...")
    content = add_hw_free_function(content)

    print("  - Adding hw_free to ops structure...")
    content = add_hw_free_to_ops(content)

    print("  - Adding clock initialization in probe...")
    content = add_clock_init_in_probe(content)

    print(f"Writing {output_file}...")
    with open(output_file, 'w') as f:
        f.write(content)

    print("Done! Modified driver written successfully.")

if __name__ == '__main__':
    main()
