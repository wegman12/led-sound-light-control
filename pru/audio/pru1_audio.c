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
    volatile uint32_t bass;                 /* Bass magnitude (0-bass_max_hz) */
    volatile uint32_t mid_low;              /* Mid-low magnitude (bass_max_hz-midlow_max_hz) */
    volatile uint32_t mid_high;             /* Mid-high magnitude (midlow_max_hz-midhigh_max_hz) */
    volatile uint32_t treble;               /* Treble magnitude (midhigh_max_hz-Nyquist) */
};

/* Sample buffer type (16-bit samples for 12-bit ADC) */
typedef uint16_t sample_t;

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

                /* Initialize FFT with completed buffer samples */
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
                    uint32_t mag_sq;
                    uint32_t bass_sum = 0;
                    uint32_t midlow_sum = 0;
                    uint32_t midhigh_sum = 0;
                    uint32_t treble_sum = 0;

                    /* Calculate bin boundaries based on configurable frequency ranges */
                    const uint16_t bass_end = FREQ_TO_BIN(ctrl->bass_max_hz);
                    const uint16_t midlow_end = FREQ_TO_BIN(ctrl->midlow_max_hz);
                    const uint16_t midhigh_end = FREQ_TO_BIN(ctrl->midhigh_max_hz);
                    const uint16_t nyquist_bin = FFT_SIZE / 2;  /* Only first half has unique data */

                    /* Accumulate magnitudes for each frequency band */
                    for (bin = 0; bin < nyquist_bin; bin++) {
                        mag_sq = fft_magnitude_squared(fft_buf->data[bin]);

                        if (bin <= bass_end) {
                            bass_sum += mag_sq;
                        } else if (bin <= midlow_end) {
                            midlow_sum += mag_sq;
                        } else if (bin <= midhigh_end) {
                            midhigh_sum += mag_sq;
                        } else {
                            treble_sum += mag_sq;
                        }
                    }

                    /* Write results to control block */
                    ctrl->bass = bass_sum;
                    ctrl->mid_low = midlow_sum;
                    ctrl->mid_high = midhigh_sum;
                    ctrl->treble = treble_sum;
                }

                /* Return to sampling status */
                ctrl->status = STATUS_SAMPLING;
            }
        }

        /* Update current position */
        ctrl->samples_in_buffer = buffer_index;
    }
}
