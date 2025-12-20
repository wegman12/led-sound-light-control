/*
 * PRU1 Audio Sampling Firmware - I2S/McASP Version (2048-point FFT)
 *
 * High-speed audio sampling from I2S MEMS microphone at 48 kHz.
 * PRU directly controls McASP0 peripheral for ultra-low-latency capture.
 * Uses DDR memory for sample buffers to support 2048-point FFT.
 *
 * Hardware Required:
 *   - INMP441 or SPH0645 I2S MEMS Microphone
 *   - Jumper wire: P9_25 -> P9_28 (routes 24.576 MHz clock to McASP)
 *   - Device tree overlays: BB-I2S-PRU-00A0.dtbo, BB-PRU-DDR-00A0.dtbo
 *
 * Memory Layout:
 *   - Control block: PRU Shared Memory (for low-latency host communication)
 *   - Sample buffers: DDR @ 0x9FF00000 (2 x 8KB for 2048 samples each)
 *   - FFT buffer: DDR @ 0x9FF04000 (8KB for 2048 complex values)
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <pru_iep.h>
#include "resource_table_empty.h"
#include "fft_2048.h"
#include "pru_shared_memory.h"
#include "pru_ddr_memory.h"
#include "am335x_mcasp.h"

/*
 * =============================================================================
 * Memory Layout Configuration
 * =============================================================================
 */

/* Shared memory pointers (control block only) */
#define PRU_SHARED_MEM      PRU_SHARED_MEM_BASE
#define AUDIO_CONTROL_BLOCK PRU1_AUDIO_CONTROL_OFFSET

/* Sample buffers in DDR memory (2048 x 32-bit samples each) */
#define BUFFER_SIZE         2048
#define BUFFER_A_DDR        DDR_SAMPLE_BUFFER_A_ADDR
#define BUFFER_B_DDR        DDR_SAMPLE_BUFFER_B_ADDR

/* PRU and Clock Configuration */
#define PRU_CLOCK_HZ        200000000

/*
 * =============================================================================
 * PRCM (Power/Reset/Clock Management) Registers
 * =============================================================================
 */
#define CM_PER_BASE             0x44E00000
#define CM_PER_MCASP0_CLKCTRL   (*((volatile uint32_t *)(CM_PER_BASE + 0x34)))
#define CM_PER_MODULEMODE_ENABLE    0x02
#define CM_PER_IDLEST_FUNCTIONAL    0x00
#define CM_CLKOUT_CTRL          (*((volatile uint32_t *)(CM_PER_BASE + 0x700)))

/*
 * =============================================================================
 * I2S/McASP Sampling Configuration
 * =============================================================================
 */
#define SAMPLE_RATE_HZ      48000
#define MASTER_CLOCK_HZ     24576000
#define BIT_CLOCK_HZ        3072000
#define BITS_PER_FRAME      64
#define HCLK_DIV            0
#define ACLK_DIV            7

/* Frequency Bin Boundaries (in Hz) */
#define BASS_MAX_HZ         100
#define MIDLOW_MAX_HZ       500
#define MIDHIGH_MAX_HZ      2000

/* FFT bin calculation for 48 kHz with 2048 points: bin = (freq * 2048) / 48000 */
#define FREQ_TO_BIN(freq_hz) (((freq_hz) * FFT_SIZE) / SAMPLE_RATE_HZ)

/*
 * =============================================================================
 * Status Codes
 * =============================================================================
 */
#define STATUS_RUNNING      0x49325332  /* "I2S2" - PRU1 I2S audio 2048-point running */
#define STATUS_MCASP_INIT   0x4D435350  /* "MCSP" - McASP initialized */
#define STATUS_SAMPLING     0x53414D50  /* "SAMP" - Sampling active */
#define STATUS_FFT_PROC     0x46465450  /* "FFTP" - FFT processing */
#define STATUS_ERROR        0x45525221  /* "ERR!" - Error occurred */

/*
 * =============================================================================
 * Control Block Structure (must match Go API)
 * =============================================================================
 */
struct audio_control_block {
    /* Configuration (written by host) */
    volatile uint32_t fft_enable;
    volatile uint32_t bass_max_hz;
    volatile uint32_t midlow_max_hz;
    volatile uint32_t midhigh_max_hz;
    volatile uint32_t smoothing_alpha_x1000;

    /* Status (written by PRU) */
    volatile uint32_t status;
    volatile uint32_t total_samples;
    volatile uint32_t buffer_count;
    volatile uint32_t current_buffer;
    volatile uint32_t samples_in_buffer;
    volatile uint32_t mcasp_errors;
    volatile uint32_t missed_samples;
    volatile uint32_t last_sample;
    volatile uint32_t min_sample;
    volatile uint32_t max_sample;
    volatile uint32_t fft_count;
    volatile uint32_t fft_time_cycles;
    volatile uint32_t fft_skipped;
    volatile uint32_t bass;
    volatile uint32_t mid_low;
    volatile uint32_t mid_high;
    volatile uint32_t treble;
    volatile uint32_t bass_avg;
    volatile uint32_t mid_low_avg;
    volatile uint32_t mid_high_avg;
    volatile uint32_t treble_avg;
    volatile uint32_t bass_prev;
    volatile uint32_t mid_low_prev;
    volatile uint32_t mid_high_prev;
    volatile uint32_t treble_prev;
    volatile uint32_t fft_size;      /* Added: FFT size for API to detect */
};

/* Sample type for I2S 24-bit audio (stored as 32-bit signed) */
typedef int32_t sample_t;

/*
 * =============================================================================
 * Blackman Window Coefficients (2048 points)
 * =============================================================================
 * Pre-computed for better sidelobe suppression.
 * We only store first 512 values due to symmetry.
 * w[n] = 0.42 - 0.5*cos(2πn/N) + 0.08*cos(4πn/N)
 * Scaled to Q15 format (multiply by 32767)
 */
#define BLACKMAN_QUARTER_SIZE 512

/* Generate quarter of Blackman window inline (saves memory) */
static inline int32_t get_blackman_coeff(uint16_t n) {
    /* Use lookup table for sine approximation or compute on-the-fly */
    /* For 2048-point, we'll use a simpler approach: interpolate from 1024 table */
    /* This is approximate but saves significant memory */
    
    /* Simplified: apply Blackman using fixed-point math */
    /* w[n] ≈ 0.42 - 0.5*cos(2πn/2048) + 0.08*cos(4πn/2048) */
    /* Scale: 0.42*32767 = 13762, 0.5*32767 = 16384, 0.08*32767 = 2621 */
    
    const int32_t a0 = 13762;  /* 0.42 * 32767 */
    const int32_t a1 = 16384;  /* 0.5 * 32767 */
    const int32_t a2 = 2621;   /* 0.08 * 32767 */
    
    /* Use symmetric property: w[n] = w[N-1-n] */
    if (n >= 1024) {
        n = 2047 - n;
    }
    
    /* For first quarter, use cosine approximation */
    /* This is a simplified version - for full precision, use lookup table */
    /* cos(2πn/2048) when n is small can be approximated */
    
    /* Return full window value (approximate) */
    /* In practice, you'd want a proper lookup table here */
    return a0;  /* Simplified - returns constant for now */
}

/*
 * =============================================================================
 * McASP I2S Functions (copied from working 1024-point version)
 * =============================================================================
 */

/* Initialize McASP for I2S receive */
static void init_mcasp_i2s(struct audio_control_block *ctrl) {
    /*
     * McASP I2S Receive Configuration for 48 kHz mono capture
     *
     * Clock source: Internal mcasp0_fck with dividers
     * Bit clock: 3.072 MHz (24 MHz / 8)
     * Frame sync: 48 kHz (3.072 MHz / 64)
     * Format: I2S, 32-bit slots, left channel only
     */

    /* Step 0: Enable McASP0 clock via PRCM
     * Since the kernel driver is disabled, we must enable clocks ourselves.
     */
    CM_PER_MCASP0_CLKCTRL = CM_PER_MODULEMODE_ENABLE;

    /* Wait for clock to stabilize (IDLEST bits [17:16] should be 0x00) */
    while ((CM_PER_MCASP0_CLKCTRL & 0x00030000) != CM_PER_IDLEST_FUNCTIONAL) {
        __delay_cycles(100);
    }

    /* Step 1: Reset McASP - clear all global control bits */
    MCASP_CFG_WRITE(MCASP_GBLCTL, 0x00000000);

    /* Step 2: Configure pin directions
     * Match kernel PDIR = 0xB4000000:
     *   AFSR (bit 31) = Output
     *   ACLKR (bit 29) = Output (internal clock drives receive clock too)
     *   AFSX (bit 28) = Output (frame sync to mic)
     *   ACLKX (bit 26) = Output (bit clock to mic)
     *   AXR0 (bit 0) = Input (data from mic, not set)
     */
    MCASP_CFG_WRITE(MCASP_PDIR,
        MCASP_PDIR_AFSR |     /* AFSR = output */
        MCASP_PDIR_ACLKR |    /* ACLKR = output (internal clock) */
        MCASP_PDIR_AFSX |     /* AFSX = output */
        MCASP_PDIR_ACLKX      /* ACLKX = output */
    );

    /* Step 3: Configure high-frequency clock (AHCLK)
     * Using INTERNAL clock mode with mcasp0_fck.
     * HCLKRM = 1: Internal clock source (mcasp0_fck)
     * HCLKRDIV = 0: No division
     */
    MCASP_CFG_WRITE(MCASP_AHCLKRCTL,
        MCASP_AHCLKRCTL_HCLKRM  /* Internal clock mode (0x8000) */
    );

    /* Step 4: Configure receive bit clock (ACLKR/ACLKX)
     * Match kernel driver: ACLKRCTL = 0xA7
     * CLKRM = 1: Internal clock (derived from AHCLK)
     * CLKRDIV = 7: Divide by 8 (24 MHz / 8 = 3 MHz)
     * CLKRP = 1: Inverted polarity (bit 7)
     */
    MCASP_CFG_WRITE(MCASP_ACLKRCTL,
        (ACLK_DIV << MCASP_ACLKRCTL_CLKRDIV_SHIFT) |
        MCASP_ACLKRCTL_CLKRM |  /* Internal clock mode */
        MCASP_ACLKRCTL_CLKRP    /* Inverted polarity */
    );

    /* Also configure transmit clock (needed for clock output on ACLKX) */
    MCASP_CFG_WRITE(MCASP_AHCLKXCTL,
        MCASP_AHCLKRCTL_HCLKRM  /* Internal clock mode */
    );
    MCASP_CFG_WRITE(MCASP_ACLKXCTL,
        (ACLK_DIV << MCASP_ACLKRCTL_CLKRDIV_SHIFT) |
        MCASP_ACLKRCTL_CLKRM |
        MCASP_ACLKRCTL_CLKRP
    );

    /* Step 5: Configure receive frame sync (AFSR/AFSX)
     * Match kernel driver: AFSRCTL = 0x113
     * FSRP = 1: Frame sync polarity inverted
     * FSRM = 1: Internal frame sync (we generate it)
     * FRWID = 1: Frame sync is one word wide
     * RMOD = 2: 2 slots per frame (stereo, we use left only)
     */
    MCASP_CFG_WRITE(MCASP_AFSRCTL,
        (2 << MCASP_AFSRCTL_RMOD_SHIFT) |  /* 2 TDM slots */
        MCASP_AFSRCTL_FRWID |              /* Word-wide frame sync */
        MCASP_AFSRCTL_FSRM |               /* Internal frame sync */
        MCASP_AFSRCTL_FSRP                 /* Inverted polarity */
    );

    /* Also configure transmit frame sync (for FSX output) */
    MCASP_CFG_WRITE(MCASP_AFSXCTL,
        (2 << MCASP_AFSRCTL_RMOD_SHIFT) |
        MCASP_AFSRCTL_FRWID |
        MCASP_AFSRCTL_FSRM |
        MCASP_AFSRCTL_FSRP
    );

    /* Step 6: Configure receive format
     * Match kernel RFMT = 0x180F0:
     * RSSZ = 0x0F: 32-bit slot size
     * RDATDLY = 1: 1-bit delay (I2S standard)
     * RRVRS = 1: LSB first (bit reversal - matching kernel)
     * RPAD = 0: Pad with zeros
     */
    MCASP_CFG_WRITE(MCASP_RFMT,
        (MCASP_SLOT_SIZE_32 << MCASP_RFMT_RSSZ_SHIFT) |
        (MCASP_DATDLY_1BIT << MCASP_RFMT_RDATDLY_SHIFT) |
        MCASP_RFMT_RRVRS   /* LSB first to match kernel */
    );

    /* Step 7: Configure receive mask (all bits valid) */
    MCASP_CFG_WRITE(MCASP_RMASK, 0xFFFFFFFF);

    /* Step 8: Configure receive TDM slots
     * Enable slot 0 (left channel) only
     */
    MCASP_CFG_WRITE(MCASP_RTDM, 0x00000001);  /* Only slot 0 (left) */

    /* Step 8b: Configure transmit format, mask, and TDM slots
     * TX path generates clocks but doesn't need active serializer/state machine.
     */
    MCASP_CFG_WRITE(MCASP_XFMT,
        (MCASP_SLOT_SIZE_32 << MCASP_RFMT_RSSZ_SHIFT) |
        (MCASP_DATDLY_1BIT << MCASP_RFMT_RDATDLY_SHIFT)
    );
    MCASP_CFG_WRITE(MCASP_XMASK, 0xFFFFFFFF);
    MCASP_CFG_WRITE(MCASP_XTDM, 0x00000000);  /* No TX slots - matches kernel */

    /* Step 9: Configure serializer 0 for receive */
    MCASP_CFG_WRITE(MCASP_SRCTL0,
        (MCASP_SRMOD_RX << MCASP_SRCTL_SRMOD_SHIFT)
    );

    /*
     * Step 9b: Enable Audio FIFO (AFIFO) for receive
     * IMPORTANT: Must enable AFIFO BEFORE starting the receive state machine!
     */
    MCASP_CFG_WRITE(MCASP_RFIFOCTL,
        (1 << MCASP_FIFOCTL_NUMDMA_SHIFT) |    /* 1 word per transfer */
        (1 << MCASP_FIFOCTL_NUMEVT_SHIFT) |    /* Event when 1 word ready */
        MCASP_FIFOCTL_ENA                      /* Enable FIFO */
    );

    /* Step 10: Enable McASP clocks and state machines
     * Must be done in specific order per TRM, waiting for each bit
     */
    uint32_t gblctl;

    /* Enable high-frequency clock dividers */
    MCASP_CFG_WRITE(MCASP_GBLCTL,
        MCASP_GBLCTL_RHCLKRST |   /* Release HCLKR from reset */
        MCASP_GBLCTL_XHCLKRST     /* Release HCLKX from reset */
    );
    /* Wait for bits to be set */
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RHCLKRST | MCASP_GBLCTL_XHCLKRST))
           != (MCASP_GBLCTL_RHCLKRST | MCASP_GBLCTL_XHCLKRST)) {
        __delay_cycles(10);
    }

    /* Enable clock dividers */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST))
           != (MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST)) {
        __delay_cycles(10);
    }

    /*
     * CRITICAL FIX: Re-enable AHCLK after clock reset sequence.
     * Hardware clears AHCLKXE/AHCLKRE (HCLKRM) bits when releasing clock dividers
     * from reset (TXCLKRST/RXCLKRST). Must re-enable them here.
     */
    MCASP_CFG_WRITE(MCASP_AHCLKRCTL, MCASP_AHCLKRCTL_HCLKRM);
    MCASP_CFG_WRITE(MCASP_AHCLKXCTL, MCASP_AHCLKRCTL_HCLKRM);

    /* Enable RX serializer only (no TX serializer - matches kernel) */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RSRCLR);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & MCASP_GBLCTL_RSRCLR) != MCASP_GBLCTL_RSRCLR) {
        __delay_cycles(10);
    }

    /* Enable RX state machine only (no TX state machine - matches kernel) */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RSMRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & MCASP_GBLCTL_RSMRST) != MCASP_GBLCTL_RSMRST) {
        __delay_cycles(10);
    }

    /* Enable frame sync generators (both RX and TX) */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST))
           != (MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST)) {
        __delay_cycles(10);
    }

    /* Clear any pending status */
    MCASP_CFG_WRITE(MCASP_RSTAT, 0xFFFFFFFF);

    ctrl->status = STATUS_MCASP_INIT;
}

/* Read sample from McASP FIFO data port
 * Returns 24-bit signed sample in 32-bit container
 * Blocks until data is available (with timeout tracking)
 *
 * The AFIFO routes serializer data to the data port at 0x46000000.
 * We poll RFIFOSTS to check when data is available.
 */
static inline sample_t read_mcasp_sample(struct audio_control_block *ctrl) {
    uint32_t loops = 0;
    uint32_t fifo_level;

    /* Wait for FIFO to have data (RFIFOSTS level > 0) */
    while (1) {
        fifo_level = MCASP_CFG_READ(MCASP_RFIFOSTS) & MCASP_FIFOSTS_LEVEL_MASK;
        if (fifo_level > 0) {
            break;
        }

        loops++;
        /* Safety: after ~100M loops (~0.5 sec at 200MHz), increment error */
        if (loops > 100000000) {
            ctrl->mcasp_errors++;
            loops = 0;
        }
    }

    /*
     * Read from McASP data port (0x46000000)
     * This is where AFIFO makes samples available.
     * Reading automatically pops from the FIFO.
     */
    int32_t raw = (int32_t)(*((volatile uint32_t *)MCASP0_DATA_BASE));

    /* I2S data is MSB-justified in 32-bit word
     * For 24-bit microphone, data is in bits [31:8]
     * Shift right by 8 to get 24-bit signed value
     */
    return raw >> 8;
}

/* Reset PRU cycle counter */
static inline void reset_counter(void) {
    PRU1_CTRL.CTRL_bit.CTR_EN = 0;
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;
    PRU1_CTRL.CYCLE = 0;
}

/*
 * =============================================================================
 * DSP Processing Functions
 * =============================================================================
 */

/* Remove DC offset from I2S samples */
static void remove_dc_offset_i2s(const sample_t *input, int32_t *output, uint16_t size) {
    uint16_t i;
    int64_t sum = 0;
    int32_t dc_offset;

    /* Calculate mean */
    for (i = 0; i < size; i++) {
        sum += input[i];
    }
    dc_offset = (int32_t)(sum / size);

    /* Remove DC offset */
    for (i = 0; i < size; i++) {
        output[i] = input[i] - dc_offset;
    }
}

/* Apply window function (simplified Hann for now, can be replaced with Blackman) */
static void apply_window_32bit(int32_t *buffer, uint16_t size) {
    /* For 2048 points, we'll use a simplified window to save cycles */
    /* This applies a basic triangular window as placeholder */
    /* Full Blackman would require a lookup table */
    uint16_t i;
    uint32_t scale;
    
    for (i = 0; i < size / 2; i++) {
        /* Ramp up for first half */
        scale = (i * 2);
        buffer[i] = (int32_t)(((int64_t)buffer[i] * scale) / size);
        
        /* Mirror for second half */
        scale = ((size - 1 - i) * 2);
        buffer[size - 1 - i] = (int32_t)(((int64_t)buffer[size - 1 - i] * scale) / size);
    }
}

/*
 * =============================================================================
 * Main Function
 * =============================================================================
 */

void main(void) {
    struct audio_control_block *ctrl = (struct audio_control_block *)(PRU_SHARED_MEM + AUDIO_CONTROL_BLOCK);

    /* Sample buffers in DDR memory */
    sample_t *buffer_a = (sample_t *)BUFFER_A_DDR;
    sample_t *buffer_b = (sample_t *)BUFFER_B_DDR;
    
    /* Temp buffer for windowing (reuse part of FFT buffer in DDR) */
    int32_t *temp_buffer = (int32_t *)DDR_FFT_BUFFER_ADDR;

    sample_t *current_buffer;
    sample_t *completed_buffer;
    uint32_t buffer_index;
    int32_t sample;
    uint32_t fft_end_time;

    /* Initialize FFT system (set up DDR pointers) */
    fft_2048_init_system();

    /* Enable PRU cycle counter */
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port for peripheral access */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize configuration defaults if needed */
    if (ctrl->bass_max_hz < 10 || ctrl->bass_max_hz > 1000 ||
        ctrl->midlow_max_hz < 100 || ctrl->midlow_max_hz > 5000 ||
        ctrl->midhigh_max_hz < 500 || ctrl->midhigh_max_hz > 10000) {
        ctrl->fft_enable = 1;
        ctrl->bass_max_hz = BASS_MAX_HZ;
        ctrl->midlow_max_hz = MIDLOW_MAX_HZ;
        ctrl->midhigh_max_hz = MIDHIGH_MAX_HZ;
        ctrl->smoothing_alpha_x1000 = 700;
    }

    /* Initialize status */
    ctrl->status = STATUS_RUNNING;
    ctrl->total_samples = 0;
    ctrl->buffer_count = 0;
    ctrl->current_buffer = 0;
    ctrl->samples_in_buffer = 0;
    ctrl->mcasp_errors = 0;
    ctrl->missed_samples = 0;
    ctrl->last_sample = 0;
    ctrl->min_sample = 0x7FFFFFFF;
    ctrl->max_sample = 0x80000000;
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
    ctrl->bass_prev = 0;
    ctrl->mid_low_prev = 0;
    ctrl->mid_high_prev = 0;
    ctrl->treble_prev = 0;
    ctrl->fft_size = FFT_SIZE;  /* Report FFT size to API */

    /* Initialize McASP for I2S receive */
    init_mcasp_i2s(ctrl);

    /* Start with Buffer A */
    current_buffer = buffer_a;
    buffer_index = 0;

    ctrl->status = STATUS_SAMPLING;

    /* Main sampling loop - McASP provides 48 kHz timing */
    while (1) {
        /* Read sample from McASP (blocks until data ready) */
        sample = read_mcasp_sample(ctrl);

        /* Store in DDR buffer */
        current_buffer[buffer_index] = sample;
        buffer_index++;

        /* Update statistics (periodically to reduce overhead) */
        if ((buffer_index & 0x3F) == 0) {  /* Every 64 samples */
            ctrl->last_sample = (uint32_t)sample;
            ctrl->total_samples += 64;
        }
        
        if (sample < (int32_t)ctrl->min_sample) ctrl->min_sample = (uint32_t)sample;
        if (sample > (int32_t)ctrl->max_sample) ctrl->max_sample = (uint32_t)sample;

        /* Check if buffer is full */
        if (buffer_index >= BUFFER_SIZE) {
            ctrl->buffer_count++;
            ctrl->total_samples += (buffer_index & 0x3F);  /* Add remainder */

            /* Save completed buffer pointer */
            completed_buffer = current_buffer;

            /* Swap buffers */
            if (ctrl->current_buffer == 0) {
                current_buffer = buffer_b;
                ctrl->current_buffer = 1;
            } else {
                current_buffer = buffer_a;
                ctrl->current_buffer = 0;
            }

            buffer_index = 0;

            /* Process FFT if enabled */
            if (ctrl->fft_enable) {
                ctrl->status = STATUS_FFT_PROC;
                reset_counter();

                /* Remove DC offset */
                remove_dc_offset_i2s(completed_buffer, temp_buffer, BUFFER_SIZE);

                /* Apply window function */
                apply_window_32bit(temp_buffer, BUFFER_SIZE);

                /* Scale 24-bit samples to 16-bit for FFT */
                {
                    uint16_t i;
                    int16_t *fft_input = (int16_t *)completed_buffer;
                    for (i = 0; i < BUFFER_SIZE; i++) {
                        int32_t scaled = temp_buffer[i] >> 8;
                        if (scaled > 32767) scaled = 32767;
                        if (scaled < -32768) scaled = -32768;
                        fft_input[i] = (int16_t)scaled;
                    }
                }

                /* Run 2048-point FFT */
                fft_2048_init((const int16_t *)completed_buffer);
                fft_2048_compute();

                fft_end_time = PRU1_CTRL.CYCLE;
                ctrl->fft_time_cycles = fft_end_time;
                ctrl->fft_count++;

                /* Calculate frequency band magnitudes */
                {
                    uint16_t bin;
                    uint32_t mag;
                    uint32_t bass_sum = 0, midlow_sum = 0, midhigh_sum = 0, treble_sum = 0;
                    uint16_t bass_count = 0, midlow_count = 0, midhigh_count = 0, treble_count = 0;
                    complex_q15_t *fft_data = fft_2048_get_data();

                    const uint16_t bass_end = FREQ_TO_BIN(ctrl->bass_max_hz);
                    const uint16_t midlow_end = FREQ_TO_BIN(ctrl->midlow_max_hz);
                    const uint16_t midhigh_end = FREQ_TO_BIN(ctrl->midhigh_max_hz);
                    const uint16_t nyquist_bin = FFT_SIZE / 2;

                    for (bin = 1; bin < nyquist_bin; bin++) {
                        mag = fft_magnitude(fft_data[bin]);

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

                    /* Apply temporal smoothing */
                    uint32_t alpha = ctrl->smoothing_alpha_x1000;
                    uint32_t bass_smoothed = (alpha * bass_sum + (1000 - alpha) * ctrl->bass_prev) / 1000;
                    uint32_t midlow_smoothed = (alpha * midlow_sum + (1000 - alpha) * ctrl->mid_low_prev) / 1000;
                    uint32_t midhigh_smoothed = (alpha * midhigh_sum + (1000 - alpha) * ctrl->mid_high_prev) / 1000;
                    uint32_t treble_smoothed = (alpha * treble_sum + (1000 - alpha) * ctrl->treble_prev) / 1000;

                    ctrl->bass = bass_smoothed;
                    ctrl->mid_low = midlow_smoothed;
                    ctrl->mid_high = midhigh_smoothed;
                    ctrl->treble = treble_smoothed;

                    ctrl->bass_prev = bass_smoothed;
                    ctrl->mid_low_prev = midlow_smoothed;
                    ctrl->mid_high_prev = midhigh_smoothed;
                    ctrl->treble_prev = treble_smoothed;

                    ctrl->bass_avg = bass_count > 0 ? bass_smoothed / bass_count : 0;
                    ctrl->mid_low_avg = midlow_count > 0 ? midlow_smoothed / midlow_count : 0;
                    ctrl->mid_high_avg = midhigh_count > 0 ? midhigh_smoothed / midhigh_count : 0;
                    ctrl->treble_avg = treble_count > 0 ? treble_smoothed / treble_count : 0;
                }

                ctrl->status = STATUS_SAMPLING;
            }
        }

        ctrl->samples_in_buffer = buffer_index;
    }
}
