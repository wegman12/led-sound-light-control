/*
 * PRU1 Audio Sampling Firmware
 *
 * High-speed audio sampling from AIN1 at 40 kHz using IEP timer.
 * Uses double buffering for continuous sampling while processing.
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <pru_iep.h>
#include <sys_tscAdcSs.h>
#include "resource_table_empty.h"
#include "fft.h"

/* Memory Layout */
#define PRU_SHARED_MEM      0x00010000
#define AUDIO_CONTROL_BLOCK 0x00002000  /* Offset 8KB in shared memory (after PRU0's 4KB) */

/* PRU1 Local DRAM - Double Buffer Layout */
#define BUFFER_SIZE         1024        /* Samples per buffer (power of 2) */
#define BUFFER_A_OFFSET     0x0000      /* Buffer A at start of PRU1 DRAM */
#define BUFFER_B_OFFSET     0x0800      /* Buffer B at 2KB offset (1024 samples × 2 bytes) */

/* PRU and IEP Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define IEP_CLOCK_HZ        200000000

/* Sampling Configuration */
#define SAMPLE_RATE_HZ      40000       /* 40 kHz sampling rate */
#define SAMPLE_PERIOD_NS    25000       /* 25 μs = 25,000 ns */
#define IEP_CMP_VALUE       5000        /* 5,000 cycles @ 200 MHz = 25 μs */

/* Frequency Bin Boundaries (in Hz) - Configurable */
#define BASS_MAX_HZ         150         /* Bass: 0-150 Hz */
#define MIDLOW_MAX_HZ       1000        /* Mid-Low: 150-1000 Hz */
#define MIDHIGH_MAX_HZ      2000        /* Mid-High: 1000-2000 Hz */
/* Treble: 2000 Hz to Nyquist (SAMPLE_RATE_HZ / 2) */

/* Calculate FFT bin index from frequency: bin = (freq * FFT_SIZE) / SAMPLE_RATE_HZ
 * FFT_SIZE is defined in fft.h as 1024
 * For 40 kHz sampling: bin resolution = 40000 / 1024 = 39.0625 Hz/bin
 */
#define FREQ_TO_BIN(freq_hz) (((freq_hz) * FFT_SIZE) / SAMPLE_RATE_HZ)

/* Control Module registers to enable the ADC peripheral */
#define CM_WKUP_CLKSTCTRL  (*((volatile uint32_t *)0x44E00400))
#define CM_WKUP_ADC_TSC_CLKCTRL  (*((volatile uint32_t *)0x44E004BC))

/* Status Codes */
#define STATUS_RUNNING      0x41554431  /* "AUD1" in ASCII - indicates PRU1 audio is running */
#define STATUS_ADC_INIT     0x41444349  /* "ADCI" in ASCII - ADC initialized */
#define STATUS_IEP_INIT     0x49455049  /* "IEPI" in ASCII - IEP initialized */
#define STATUS_SAMPLING     0x53414D50  /* "SAMP" in ASCII - High-speed sampling active */
#define STATUS_FFT_PROC     0x46465450  /* "FFTP" in ASCII - FFT processing */

/* Audio Control Block Structure (in shared memory)
 * Layout: Configuration (written by host) + Status (written by PRU)
 */
struct audio_control_block {
    /* === Configuration Section (written by host, read by PRU) === */
    volatile uint32_t fft_enable;           /* 1 = FFT enabled, 0 = disabled */
    volatile uint32_t bass_max_hz;          /* Bass upper frequency boundary (Hz) */
    volatile uint32_t midlow_max_hz;        /* Mid-low upper frequency boundary (Hz) */
    volatile uint32_t midhigh_max_hz;       /* Mid-high upper frequency boundary (Hz) */
    volatile uint32_t smoothing_alpha_x1000;/* Temporal smoothing factor (0-1000, where 1000 = 1.0) */

    /* === Status Section (written by PRU, read by host) === */
    volatile uint32_t status;               /* PRU running status */
    volatile uint32_t total_samples;        /* Total samples collected */
    volatile uint32_t buffer_count;         /* Number of completed buffers */
    volatile uint32_t current_buffer;       /* 0 = Buffer A, 1 = Buffer B */
    volatile uint32_t samples_in_buffer;    /* Current sample count in active buffer */
    volatile uint32_t adc_timeouts;         /* ADC timeout errors */
    volatile uint32_t missed_samples;       /* Missed samples (overruns) */
    volatile uint32_t last_sample;          /* Most recent sample value */
    volatile uint32_t min_sample;           /* Minimum sample value */
    volatile uint32_t max_sample;           /* Maximum sample value */
    volatile uint32_t fft_count;            /* Number of FFTs computed */
    volatile uint32_t fft_time_cycles;      /* Last FFT processing time (PRU cycles) */
    volatile uint32_t fft_skipped;          /* FFTs skipped due to timing overrun */
    volatile uint32_t bass;                 /* Bass magnitude sum (0-bass_max_hz) */
    volatile uint32_t mid_low;              /* Mid-low magnitude sum (bass_max_hz-midlow_max_hz) */
    volatile uint32_t mid_high;             /* Mid-high magnitude sum (midlow_max_hz-midhigh_max_hz) */
    volatile uint32_t treble;               /* Treble magnitude sum (midhigh_max_hz-Nyquist) */
    volatile uint32_t bass_avg;             /* Bass average magnitude per bin */
    volatile uint32_t mid_low_avg;          /* Mid-low average magnitude per bin */
    volatile uint32_t mid_high_avg;         /* Mid-high average magnitude per bin */
    volatile uint32_t treble_avg;           /* Treble average magnitude per bin */

    /* Temporal smoothing state (previous values) */
    volatile uint32_t bass_prev;            /* Previous bass value for smoothing */
    volatile uint32_t mid_low_prev;         /* Previous mid_low value for smoothing */
    volatile uint32_t mid_high_prev;        /* Previous mid_high value for smoothing */
    volatile uint32_t treble_prev;          /* Previous treble value for smoothing */
};

/* Sample buffer type (16-bit samples for 12-bit ADC) */
typedef uint16_t sample_t;

/* Pre-computed Hann window coefficients using int32_t with 16-bit scaling
 * w(n) = 0.5 * (1 - cos(2πn / (N-1))) * 65536
 * Range: 0 to 65536 (representing 0.0 to 1.0)
 * Only storing first 512 values; second half is symmetric
 */
const int32_t hann_window_half[BUFFER_SIZE / 2] = {
    0,0,2,5,9,15,22,30,39,50,61,74,88,104,121,138,
    158,178,200,222,246,272,298,326,355,385,416,449,483,518,554,592,
    630,670,711,754,797,842,888,935,983,1033,1084,1136,1189,1243,1299,1355,
    1413,1472,1533,1594,1657,1720,1785,1851,1919,1987,2057,2128,2199,2273,2347,2422,
    2499,2576,2655,2735,2816,2898,2982,3066,3152,3238,3326,3415,3505,3596,3688,3782,
    3876,3972,4068,4166,4265,4364,4465,4567,4670,4774,4880,4986,5093,5201,5311,5421,
    5532,5645,5758,5873,5988,6105,6222,6341,6460,6581,6702,6825,6948,7072,7198,7324,
    7451,7580,7709,7839,7970,8102,8235,8369,8504,8640,8776,8914,9052,9192,9332,9473,
    9615,9758,9901,10046,10191,10338,10485,10633,10782,10931,11082,11233,11385,11538,11692,11846,
    12002,12158,12315,12472,12631,12790,12950,13110,13272,13434,13597,13760,13924,14090,14255,14422,
    14589,14757,14925,15094,15264,15434,15606,15777,15950,16123,16296,16471,16646,16821,16997,17174,
    17351,17529,17708,17887,18066,18246,18427,18608,18790,18972,19155,19339,19522,19707,19891,20077,
    20263,20449,20636,20823,21010,21198,21387,21576,21765,21955,22145,22336,22527,22718,22910,23102,
    23295,23487,23681,23874,24068,24262,24457,24652,24847,25042,25238,25434,25630,25827,26024,26221,
    26418,26616,26813,27011,27210,27408,27607,27806,28005,28204,28403,28603,28802,29002,29202,29402,
    29603,29803,30003,30204,30405,30606,30806,31007,31208,31409,31611,31812,32013,32214,32415,32617,
    32818,33019,33220,33422,33623,33824,34025,34226,34427,34628,34829,35030,35231,35431,35632,35832,
    36033,36233,36433,36633,36832,37032,37232,37431,37630,37829,38028,38226,38425,38623,38821,39018,
    39216,39413,39610,39807,40003,40199,40395,40591,40786,40981,41176,41370,41564,41758,41951,42144,
    42337,42529,42721,42913,43104,43294,43485,43675,43864,44054,44242,44431,44618,44806,44993,45179,
    45365,45551,45736,45921,46105,46288,46471,46654,46836,47017,47198,47379,47559,47738,47917,48095,
    48272,48449,48626,48802,48977,49151,49325,49499,49672,49844,50015,50186,50356,50526,50694,50862,
    51030,51197,51363,51528,51693,51857,52020,52182,52344,52505,52665,52825,52984,53142,53299,53455,
    53611,53766,53920,54073,54226,54378,54529,54679,54828,54976,55124,55271,55416,55561,55706,55849,
    55991,56133,56273,56413,56552,56690,56827,56963,57099,57233,57366,57499,57630,57761,57891,58020,
    58147,58274,58400,58525,58649,58772,58894,59015,59135,59254,59372,59489,59605,59720,59834,59947,
    60058,60169,60279,60388,60496,60602,60708,60813,60916,61019,61120,61221,61320,61418,61515,61611,
    61706,61800,61893,61985,62075,62165,62253,62340,62426,62511,62595,62678,62760,62840,62919,62998,
    63075,63151,63226,63299,63372,63443,63513,63582,63650,63717,63782,63847,63910,63972,64033,64092,
    64151,64208,64264,64319,64373,64425,64477,64527,64576,64624,64670,64716,64760,64803,64844,64885,
    64924,64962,64999,65035,65069,65102,65134,65165,65195,65223,65250,65276,65301,65324,65346,65367,
    65387,65406,65423,65439,65454,65467,65480,65491,65501,65509,65517,65523,65528,65532,65534,65535
};

/* Apply DC offset removal using 32-bit arithmetic
 * Converts unsigned ADC samples to signed 32-bit values centered at zero
 * Input: uint16_t samples (0-4095 ADC range)
 * Output: int32_t samples centered at zero
 */
static void remove_dc_offset_32bit(const sample_t *input, int32_t *output, uint16_t size) {
    uint32_t sum = 0;
    int32_t mean;
    uint16_t i;

    /* Calculate mean of buffer */
    for (i = 0; i < size; i++) {
        sum += input[i];
    }
    mean = (int32_t)(sum / size);

    /* Convert to signed 32-bit and subtract mean (no overflow possible) */
    for (i = 0; i < size; i++) {
        output[i] = (int32_t)input[i] - mean;
    }
}

/* Apply Hann window to 32-bit buffer samples (in-place)
 * Multiplies each sample by the corresponding window coefficient
 * Window is symmetric, so we only store half and mirror it
 * Window coefficients are scaled by 65536, representing 0.0 to 1.0
 * Works on int32_t to avoid any overflow during multiplication
 */
static void apply_hann_window_32bit(int32_t *buffer, uint16_t size) {
    uint16_t i;

    /* Apply window - using symmetric property */
    for (i = 0; i < size; i++) {
        int32_t window_coef;

        /* Use first half or mirror from second half */
        if (i < size / 2) {
            window_coef = hann_window_half[i];
        } else {
            window_coef = hann_window_half[size - 1 - i];
        }

        /* Multiply sample by window coefficient and scale back
         * Coefficients are scaled by 65536, so divide by 65536 (shift right by 16)
         * Using 64-bit intermediate to avoid overflow during multiplication
         */
        int64_t windowed = ((int64_t)buffer[i] * (int64_t)window_coef) >> 16;
        buffer[i] = (int32_t)windowed;
    }
}

/* Initialize IEP timer for 40 kHz periodic events */
static void init_iep_timer(void) {
    /* Disable IEP timer */
    CT_IEP.TMR_GLB_CFG_bit.CNT_EN = 0;

    /* Clear counter */
    CT_IEP.TMR_CNT = 0;

    /* Clear overflow status */
    CT_IEP.TMR_GLB_STS_bit.CNT_OVF = 1;

    /* Set compare value for 40 kHz (25 μs = 5,000 cycles @ 200 MHz) */
    CT_IEP.TMR_CMP0 = IEP_CMP_VALUE;

    /* Configure CMP0 to reset counter on match (for continuous periodic events) */
    CT_IEP.TMR_CMP_CFG_bit.CMP0_RST_CNT_EN = 1;

    /* Enable CMP0 */
    CT_IEP.TMR_CMP_CFG_bit.CMP_EN = 0x01;  /* Enable CMP0 (bit 0) */

    /* Set default increment to 1 (count every clock cycle) */
    CT_IEP.TMR_GLB_CFG_bit.DEFAULT_INC = 1;

    /* Enable IEP timer */
    CT_IEP.TMR_GLB_CFG_bit.CNT_EN = 1;
}

/* Initialize ADC for sampling AIN1 */
static void init_adc(struct audio_control_block *ctrl) {
    /* Enable ADC clock in Control Module
     * Set the always-on clock domain to NO_SLEEP and enable ADC_TSC clock
     */
    while (!(CM_WKUP_ADC_TSC_CLKCTRL == 0x02)) {
        CM_WKUP_CLKSTCTRL = 0;
        CM_WKUP_ADC_TSC_CLKCTRL = 0x02;
    }

    /* Disable ADC module so we can configure it */
    ADC_TSC.CTRL_bit.ENABLE = 0;

    /* Enable step configuration writes (write protect bit = 1 means writable) */
    ADC_TSC.CTRL_bit.STEPCONFIG_WRITEPROTECT_N_ACTIVE_LOW = 1;

    /* Configure ADC clock divider (0 = divide by 1, highest speed) */
    ADC_TSC.ADC_CLKDIV_bit.ADC_CLKDIV = 0;

    /* Configure Step 1 for AIN1 (Channel 1) sampling
     * MODE = 0: Software enabled, one-shot
     * AVERAGING = 0: No averaging (fastest)
     * SEL_INP_SWC_3_0 = 1: Channel 1 (AIN1)
     * FIFO_SELECT = 0: Use FIFO0
     */
    ADC_TSC.STEPCONFIG1_bit.MODE = 0;  /* One-shot mode */
    ADC_TSC.STEPCONFIG1_bit.AVERAGING = 0;  /* No averaging */
    ADC_TSC.STEPCONFIG1_bit.SEL_INP_SWC_3_0 = 1;  /* AIN1 */
    ADC_TSC.STEPCONFIG1_bit.FIFO_SELECT = 0;  /* FIFO0 */

    /* Set step delays to 0 (fastest) */
    ADC_TSC.STEPDELAY1 = 0;

    /* Disable step configuration writes (protect configuration) */
    ADC_TSC.CTRL_bit.STEPCONFIG_WRITEPROTECT_N_ACTIVE_LOW = 0;

    /* Enable channel ID tagging in FIFO data */
    ADC_TSC.CTRL_bit.STEP_ID_TAG = 1;

    /* Enable ADC module */
    ADC_TSC.CTRL_bit.ENABLE = 1;

    ctrl->status = STATUS_ADC_INIT;
}

/* Fast ADC sample read (assumes data is ready in FIFO) */
static inline uint16_t read_adc_sample_fast(void) {
    uint32_t fifo_data;

    /* Check if data available */
    if (ADC_TSC.FIFO0COUNT > 0) {
        /* Read from FIFO */
        fifo_data = ADC_TSC.FIFO0DATA;
        /* Extract 12-bit sample value from bits [11:0] */
        return (uint16_t)(fifo_data & 0xFFF);
    }

    /* No data - return 0 (indicates error) */
    return 0;
}

/* Trigger ADC conversion */
static inline void trigger_adc(void) {
    /* Trigger step 1 for AIN1 capture */
    ADC_TSC.STEPENABLE = (1 << 1);
}

/* Reset PRU cycle counter to prevent overflow in timing measurements */
static inline void reset_counter(void) {
    /* Reset counter to 0 by disabling and re-enabling */
    PRU1_CTRL.CTRL_bit.CTR_EN = 0;
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Reset counter to 0 by writing directly to CYCLE register */
    PRU1_CTRL.CYCLE = 0;
}

/* Main function - High-speed 40 kHz ADC sampling with double buffering and FFT */
void main(void) {
    struct audio_control_block *ctrl = (struct audio_control_block *)(PRU_SHARED_MEM + AUDIO_CONTROL_BLOCK);
    sample_t *buffer_a = (sample_t *)BUFFER_A_OFFSET;  /* Buffer A in PRU1 DRAM */
    sample_t *buffer_b = (sample_t *)BUFFER_B_OFFSET;  /* Buffer B in PRU1 DRAM */
    fft_buffer_t *fft_buf = (fft_buffer_t *)FFT_BUFFER_OFFSET;  /* FFT working memory */
    sample_t *current_buffer;
    sample_t *completed_buffer;
    uint32_t buffer_index;
    uint16_t sample;
    uint32_t fft_end_time;

    /* Enable PRU cycle counter */
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port - allows PRU to access peripheral registers */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize configuration section with defaults (if not already set by host)
     * Check if configuration looks valid - all boundaries should be reasonable
     */
    if (ctrl->bass_max_hz < 10 || ctrl->bass_max_hz > 1000 ||
        ctrl->midlow_max_hz < 100 || ctrl->midlow_max_hz > 5000 ||
        ctrl->midhigh_max_hz < 500 || ctrl->midhigh_max_hz > 10000) {
        /* Configuration invalid or first boot - set defaults */
        ctrl->fft_enable = 1;              /* FFT enabled by default */
        ctrl->bass_max_hz = BASS_MAX_HZ;
        ctrl->midlow_max_hz = MIDLOW_MAX_HZ;
        ctrl->midhigh_max_hz = MIDHIGH_MAX_HZ;
        ctrl->smoothing_alpha_x1000 = 700; /* Default alpha = 0.7 (70% new, 30% old) */
    }

    /* Initialize status section */
    ctrl->status = STATUS_RUNNING;
    ctrl->total_samples = 0;
    ctrl->buffer_count = 0;
    ctrl->current_buffer = 0;  /* Start with Buffer A */
    ctrl->samples_in_buffer = 0;
    ctrl->adc_timeouts = 0;
    ctrl->missed_samples = 0;
    ctrl->last_sample = 0;
    ctrl->min_sample = 4095;   /* Start at max */
    ctrl->max_sample = 0;      /* Start at min */
    ctrl->fft_count = 0;
    ctrl->fft_time_cycles = 0;
    ctrl->fft_skipped = 0;
    ctrl->bass = 0;
    ctrl->mid_low = 0;
    ctrl->mid_high = 0;
    ctrl->treble = 0;
    ctrl->bass_avg = 0;
    ctrl->mid_low_avg = 0;
    ctrl->mid_high_avg = 0;
    ctrl->treble_avg = 0;

    /* Initialize temporal smoothing state */
    ctrl->bass_prev = 0;
    ctrl->mid_low_prev = 0;
    ctrl->mid_high_prev = 0;
    ctrl->treble_prev = 0;

    /* Initialize ADC */
    init_adc(ctrl);

    /* Initialize IEP timer for 40 kHz sampling */
    init_iep_timer();
    ctrl->status = STATUS_IEP_INIT;

    /* Set initial buffer to Buffer A */
    current_buffer = buffer_a;
    buffer_index = 0;

    /* Trigger first ADC conversion */
    trigger_adc();

    /* Update status to indicate high-speed sampling is active */
    ctrl->status = STATUS_SAMPLING;

    /* Main sampling loop - 40 kHz continuous sampling */
    while (1) {
        /* Wait for IEP CMP0 event (40 kHz tick) */
        while ((CT_IEP.TMR_CMP_STS & 0x01) == 0) {
            /* Busy wait for CMP0 event */
            /* This tight loop ensures minimal jitter */
        }

        /* Clear CMP0 status by writing 1 */
        CT_IEP.TMR_CMP_STS = 0x01;

        /* Read ADC sample (from previous trigger) */
        sample = read_adc_sample_fast();

        /* Check if we got a valid sample */
        if (sample == 0 && ADC_TSC.FIFO0COUNT == 0) {
            /* Timeout - ADC didn't complete in time */
            ctrl->adc_timeouts++;
        }

        /* Trigger next ADC conversion immediately */
        trigger_adc();

        /* Store sample in current buffer */
        current_buffer[buffer_index] = sample;
        buffer_index++;

        /* Update statistics */
        ctrl->last_sample = sample;
        if (sample < ctrl->min_sample && sample > 0) ctrl->min_sample = sample;
        if (sample > ctrl->max_sample) ctrl->max_sample = sample;
        ctrl->total_samples++;

        /* Check if buffer is full */
        if (buffer_index >= BUFFER_SIZE) {
            /* Buffer complete - process FFT on this buffer */
            ctrl->buffer_count++;

            /* Save pointer to completed buffer */
            completed_buffer = current_buffer;

            /* Swap buffers */
            if (ctrl->current_buffer == 0) {
                /* Switch to Buffer B */
                current_buffer = buffer_b;
                ctrl->current_buffer = 1;
            } else {
                /* Switch to Buffer A */
                current_buffer = buffer_a;
                ctrl->current_buffer = 0;
            }

            /* Reset buffer index */
            buffer_index = 0;

            /* Process FFT on completed buffer (if enabled) */
            if (ctrl->fft_enable) {
                ctrl->status = STATUS_FFT_PROC;

                /* Reset cycle counter to prevent overflow during timing */
                reset_counter();

                /* Use FFT buffer space as temporary 32-bit working buffer
                 * This avoids overflow issues during preprocessing
                 * FFT buffer is 4KB = 1024 complex samples = 1024 int32_t values
                 */
                int32_t *temp_buffer = (int32_t *)fft_buf;

                /* Pre-process Step 1: Remove DC offset using 32-bit arithmetic
                 * Input: uint16_t samples (0-4095), Output: int32_t centered at zero
                 */
                remove_dc_offset_32bit(completed_buffer, temp_buffer, BUFFER_SIZE);

                /* Pre-process Step 2: Apply Hann window using 32-bit arithmetic
                 * This reduces spectral leakage without overflow concerns
                 */
                apply_hann_window_32bit(temp_buffer, BUFFER_SIZE);

                /* Convert preprocessed 32-bit samples to 16-bit for FFT
                 * We can reuse the completed_buffer space since we're done with original data
                 */
                {
                    uint16_t i;
                    int16_t *output_buffer = (int16_t *)completed_buffer;
                    for (i = 0; i < BUFFER_SIZE; i++) {
                        /* Clamp to int16_t range to prevent overflow */
                        int32_t val = temp_buffer[i];
                        if (val > 32767) val = 32767;
                        if (val < -32768) val = -32768;
                        output_buffer[i] = (int16_t)val;
                    }
                }

                /* Initialize FFT with preprocessed samples */
                fft_init(fft_buf, (const int16_t *)completed_buffer);

                /* Compute FFT */
                fft_compute(fft_buf);

                /* Record FFT processing time (cycles elapsed since reset) */
                fft_end_time = PRU1_CTRL.CYCLE;
                ctrl->fft_time_cycles = fft_end_time;
                ctrl->fft_count++;

                /* Calculate magnitude and accumulate into frequency bins */
                {
                    uint16_t bin;
                    uint32_t mag;
                    uint32_t bass_sum = 0;
                    uint32_t midlow_sum = 0;
                    uint32_t midhigh_sum = 0;
                    uint32_t treble_sum = 0;
                    uint16_t bass_count = 0;
                    uint16_t midlow_count = 0;
                    uint16_t midhigh_count = 0;
                    uint16_t treble_count = 0;

                    /* Calculate bin boundaries based on configurable frequency ranges */
                    const uint16_t bass_end = FREQ_TO_BIN(ctrl->bass_max_hz);
                    const uint16_t midlow_end = FREQ_TO_BIN(ctrl->midlow_max_hz);
                    const uint16_t midhigh_end = FREQ_TO_BIN(ctrl->midhigh_max_hz);
                    const uint16_t nyquist_bin = FFT_SIZE / 2;  /* Only first half has unique data */

                    /* Accumulate magnitudes for each frequency band
                     * Skip bin 0 (DC component) - it represents DC offset, not audio frequency
                     * Using magnitude (not squared) for more linear response and less noise amplification
                     */
                    for (bin = 1; bin < nyquist_bin; bin++) {
                        mag = fft_magnitude(fft_buf->data[bin]);

                        if (bin <= bass_end) {
                            bass_sum += mag;
                            bass_count++;
                        } else if (bin <= midlow_end) {
                            midlow_sum += mag;
                            midlow_count++;
                        } else if (bin <= midhigh_end) {
                            midhigh_sum += mag;
                            midhigh_count++;
                        } else {
                            treble_sum += mag;
                            treble_count++;
                        }
                    }

                    /* Apply temporal smoothing: smoothed = alpha * new + (1 - alpha) * prev
                     * Using fixed-point arithmetic: alpha is in range [0, 1000]
                     * Formula: smoothed = (alpha * new + (1000 - alpha) * prev) / 1000
                     */
                    uint32_t alpha = ctrl->smoothing_alpha_x1000;
                    uint32_t bass_smoothed = (alpha * bass_sum + (1000 - alpha) * ctrl->bass_prev) / 1000;
                    uint32_t midlow_smoothed = (alpha * midlow_sum + (1000 - alpha) * ctrl->mid_low_prev) / 1000;
                    uint32_t midhigh_smoothed = (alpha * midhigh_sum + (1000 - alpha) * ctrl->mid_high_prev) / 1000;
                    uint32_t treble_smoothed = (alpha * treble_sum + (1000 - alpha) * ctrl->treble_prev) / 1000;

                    /* Write smoothed sum results to control block */
                    ctrl->bass = bass_smoothed;
                    ctrl->mid_low = midlow_smoothed;
                    ctrl->mid_high = midhigh_smoothed;
                    ctrl->treble = treble_smoothed;

                    /* Update previous values for next iteration */
                    ctrl->bass_prev = bass_smoothed;
                    ctrl->mid_low_prev = midlow_smoothed;
                    ctrl->mid_high_prev = midhigh_smoothed;
                    ctrl->treble_prev = treble_smoothed;

                    /* Calculate and write average results (avoid division by zero) */
                    ctrl->bass_avg = bass_count > 0 ? bass_smoothed / bass_count : 0;
                    ctrl->mid_low_avg = midlow_count > 0 ? midlow_smoothed / midlow_count : 0;
                    ctrl->mid_high_avg = midhigh_count > 0 ? midhigh_smoothed / midhigh_count : 0;
                    ctrl->treble_avg = treble_count > 0 ? treble_smoothed / treble_count : 0;
                }

                /* Return to sampling status */
                ctrl->status = STATUS_SAMPLING;
            }
        }

        /* Update current position */
        ctrl->samples_in_buffer = buffer_index;
    }
}
