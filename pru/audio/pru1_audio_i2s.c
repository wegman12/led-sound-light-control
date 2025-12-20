/*
 * PRU1 Audio Sampling Firmware - I2S/McASP Version
 *
 * High-speed audio sampling from I2S MEMS microphone at 48 kHz.
 * PRU directly controls McASP0 peripheral for ultra-low-latency capture.
 * Uses double buffering in shared memory for 24-bit samples.
 *
 * Hardware Required:
 *   - INMP441 or SPH0645 I2S MEMS Microphone
 *   - Jumper wire: P9_25 -> P9_28 (routes 24.576 MHz clock to McASP)
 *   - Device tree overlay: BB-I2S-PRU-00A0.dtbo
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <pru_iep.h>
#include "resource_table_empty.h"
#include "fft.h"
#include "pru_shared_memory.h"
#include "am335x_mcasp.h"

/*
 * =============================================================================
 * Memory Layout Configuration
 * =============================================================================
 */

/* Shared memory pointers */
#define PRU_SHARED_MEM      PRU_SHARED_MEM_BASE
#define AUDIO_CONTROL_BLOCK PRU1_AUDIO_CONTROL_OFFSET

/* Sample buffers in shared memory (32-bit samples for 24-bit I2S data) */
#define BUFFER_SIZE         1024
#define BUFFER_A_SHARED     (PRU_SHARED_MEM + PRU1_SAMPLE_BUFFER_A_OFFSET)
#define BUFFER_B_SHARED     (PRU_SHARED_MEM + PRU1_SAMPLE_BUFFER_B_OFFSET)

/* PRU and Clock Configuration */
#define PRU_CLOCK_HZ        200000000

/*
 * =============================================================================
 * PRCM (Power/Reset/Clock Management) Registers
 * =============================================================================
 * Since the kernel McASP driver is disabled, the PRU must enable clocks.
 */
#define CM_PER_BASE             0x44E00000
#define CM_PER_MCASP0_CLKCTRL   (*((volatile uint32_t *)(CM_PER_BASE + 0x34)))
#define CM_PER_MODULEMODE_ENABLE    0x02
#define CM_PER_IDLEST_FUNCTIONAL    0x00

/* CLKOUT2 Control - outputs 24.576 MHz clock on P9_25 for McASP */
#define CM_CLKOUT_CTRL          (*((volatile uint32_t *)(CM_PER_BASE + 0x700)))
#define CLKOUT2_EN              (1 << 7)    /* Enable CLKOUT2 */
#define CLKOUT2_DIV_SHIFT       0           /* Divider bits 0-2 */
#define CLKOUT2_SOURCE_SHIFT    3           /* Source bits 3-5 */

/* CLKOUT2 source options:
 * 0 = CLK_32768 (32.768 kHz)
 * 1 = L3_GCLK
 * 2 = DPLL_DDR_M2
 * 3 = DPLL_PER_M2 (192 MHz on standard BBB)
 * 4 = LCD_CLK
 *
 * For 24.576 MHz, we need audio PLL which requires additional config.
 * As a workaround, we'll use the HDMI audio path configuration.
 */

/*
 * =============================================================================
 * I2S/McASP Sampling Configuration
 * =============================================================================
 *
 * Clock chain:
 *   24.576 MHz (CLKOUT2 via P9_25 jumper to P9_28)
 *   -> AHCLKR (master clock input)
 *   -> ACLKX = 24.576 MHz / 8 = 3.072 MHz (bit clock)
 *   -> FSX = 3.072 MHz / 64 = 48 kHz (frame sync / sample rate)
 *
 * I2S Frame Format:
 *   - 2 slots per frame (left + right channel)
 *   - 32 bits per slot
 *   - 64 bits total per frame
 *   - We only use left channel (microphone outputs on left)
 */
#define SAMPLE_RATE_HZ      48000
#define MASTER_CLOCK_HZ     24576000
#define BIT_CLOCK_HZ        3072000
#define BITS_PER_FRAME      64

/* Clock divider values */
#define HCLK_DIV            0   /* No division of AHCLK (use 24.576 MHz directly) */
#define ACLK_DIV            7   /* Divide by 8: 24.576 MHz / 8 = 3.072 MHz */

/* Frequency Bin Boundaries (in Hz) - Configurable via control block
 * Optimized based on I2S audio analysis (2024-12-19):
 * - Narrower bands improve SNR
 * - Bass: 0-100 Hz, Mid-Low: 100-500 Hz, Mid-High: 500-2000 Hz, Treble: 2000+ Hz
 */
#define BASS_MAX_HZ         100
#define MIDLOW_MAX_HZ       500
#define MIDHIGH_MAX_HZ      2000

/* FFT bin calculation for 48 kHz: bin = (freq * 1024) / 48000 */
#define FREQ_TO_BIN(freq_hz) (((freq_hz) * FFT_SIZE) / SAMPLE_RATE_HZ)

/*
 * =============================================================================
 * Status Codes
 * =============================================================================
 */
#define STATUS_RUNNING      0x49325331  /* "I2S1" - PRU1 I2S audio running */
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
    volatile uint32_t mcasp_errors;       /* Was adc_timeouts */
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

    /* Temporal smoothing state */
    volatile uint32_t bass_prev;
    volatile uint32_t mid_low_prev;
    volatile uint32_t mid_high_prev;
    volatile uint32_t treble_prev;

    /* Debug: McASP register values for diagnostics */
    volatile uint32_t debug_gblctl;      /* GBLCTL register after init */
    volatile uint32_t debug_rstat;       /* RSTAT register (updated in loop) */
    volatile uint32_t debug_pfunc;       /* PFUNC register */
    volatile uint32_t debug_pdir;        /* PDIR register */
    volatile uint32_t debug_aclkrctl;    /* ACLKRCTL register */
    volatile uint32_t debug_ahclkrctl;   /* AHCLKRCTL register */
    volatile uint32_t debug_srctl0;      /* SRCTL0 register */
    volatile uint32_t debug_loop_count;  /* Loop iterations waiting for data */
};

/* Sample type: 32-bit signed for 24-bit I2S data */
typedef int32_t sample_t;

/*
 * =============================================================================
 * Pre-computed Blackman Window Coefficients
 * =============================================================================
 * Blackman window provides better sidelobe suppression than Hann.
 * Optimized based on I2S audio analysis (2024-12-19).
 * w(n) = 0.42 - 0.5*cos(2πn/(N-1)) + 0.08*cos(4πn/(N-1))
 */
const int32_t blackman_window_half[BUFFER_SIZE / 2] = {
    0,0,1,2,4,6,8,11,14,18,22,27,32,38,44,50,
    57,65,72,81,89,99,108,119,129,140,152,164,176,189,203,217,
    231,246,261,277,293,310,328,345,364,382,402,422,442,463,484,506,
    528,551,575,599,623,648,674,700,727,754,782,810,839,869,899,929,
    961,992,1025,1058,1091,1126,1160,1196,1232,1268,1306,1343,1382,1421,1461,1501,
    1542,1584,1626,1669,1713,1757,1802,1848,1895,1942,1989,2038,2087,2137,2187,2239,
    2291,2344,2397,2451,2506,2562,2618,2675,2733,2792,2852,2912,2973,3035,3097,3161,
    3225,3290,3356,3422,3490,3558,3627,3697,3768,3839,3912,3985,4059,4134,4210,4287,
    4364,4443,4522,4602,4683,4765,4848,4932,5017,5102,5189,5276,5365,5454,5544,5635,
    5727,5820,5914,6009,6105,6202,6300,6398,6498,6599,6700,6803,6906,7011,7117,7223,
    7331,7439,7549,7659,7770,7883,7996,8111,8226,8343,8460,8579,8698,8819,8941,9063,
    9187,9311,9437,9564,9691,9820,9950,10081,10213,10345,10479,10614,10750,10887,11025,11164,
    11304,11445,11587,11731,11875,12020,12166,12314,12462,12611,12762,12913,13066,13219,13373,13529,
    13685,13843,14002,14161,14322,14483,14646,14809,14974,15139,15306,15474,15642,15812,15982,16154,
    16326,16500,16674,16849,17026,17203,17381,17560,17741,17922,18104,18287,18470,18655,18841,19027,
    19215,19403,19593,19783,19974,20166,20358,20552,20746,20942,21138,21335,21533,21731,21931,22131,
    22332,22534,22737,22940,23144,23349,23555,23761,23968,24176,24385,24594,24804,25015,25226,25438,
    25651,25864,26078,26293,26508,26724,26941,27158,27375,27593,27812,28032,28251,28472,28693,28914,
    29136,29358,29581,29805,30029,30253,30477,30703,30928,31154,31380,31607,31834,32061,32289,32517,
    32745,32974,33203,33432,33661,33891,34121,34351,34581,34812,35042,35273,35504,35735,35967,36198,
    36429,36661,36893,37124,37356,37588,37819,38051,38283,38515,38746,38978,39209,39441,39672,39903,
    40134,40365,40596,40826,41057,41287,41517,41747,41976,42205,42434,42663,42891,43119,43347,43574,
    43801,44027,44254,44479,44705,44929,45154,45377,45601,45824,46046,46268,46489,46710,46930,47149,
    47368,47586,47804,48021,48237,48452,48667,48881,49094,49307,49519,49730,49940,50149,50358,50565,
    50772,50978,51183,51387,51591,51793,51994,52195,52394,52592,52790,52986,53181,53375,53569,53761,
    53952,54141,54330,54518,54704,54889,55073,55256,55438,55618,55798,55975,56152,56327,56502,56674,
    56846,57016,57185,57352,57518,57683,57846,58008,58168,58327,58484,58640,58795,58948,59100,59250,
    59398,59545,59691,59835,59977,60118,60257,60394,60530,60665,60797,60928,61058,61186,61312,61436,
    61559,61680,61799,61916,62032,62146,62258,62369,62478,62585,62690,62793,62895,62995,63093,63189,
    63283,63375,63466,63555,63642,63727,63810,63891,63970,64048,64123,64197,64269,64338,64406,64472,
    64536,64598,64658,64716,64772,64827,64879,64929,64977,65024,65068,65110,65151,65189,65225,65260,
    65292,65322,65351,65377,65401,65423,65444,65462,65478,65492,65504,65514,65523,65529,65533,65535
};

/*
 * =============================================================================
 * Signal Processing Functions
 * =============================================================================
 */

/* Remove DC offset from signed 32-bit I2S samples
 * I2S samples are already signed, so we just need to remove the DC component
 */
static void remove_dc_offset_i2s(const sample_t *input, int32_t *output, uint16_t size) {
    int64_t sum = 0;
    int32_t mean;
    uint16_t i;

    /* Calculate mean (using 64-bit to avoid overflow with 24-bit samples) */
    for (i = 0; i < size; i++) {
        sum += input[i];
    }
    mean = (int32_t)(sum / size);

    /* Subtract mean from each sample */
    for (i = 0; i < size; i++) {
        output[i] = input[i] - mean;
    }
}

/* Apply Blackman window for better sidelobe suppression */
static void apply_blackman_window_32bit(int32_t *buffer, uint16_t size) {
    uint16_t i;

    for (i = 0; i < size; i++) {
        int32_t window_coef;

        if (i < size / 2) {
            window_coef = blackman_window_half[i];
        } else {
            window_coef = blackman_window_half[size - 1 - i];
        }

        int64_t windowed = ((int64_t)buffer[i] * (int64_t)window_coef) >> 16;
        buffer[i] = (int32_t)windowed;
    }
}

/*
 * =============================================================================
 * McASP Initialization for I2S Receive
 * =============================================================================
 */

static void init_mcasp_i2s(struct audio_control_block *ctrl) {
    /*
     * McASP I2S Receive Configuration for 48 kHz mono capture
     *
     * Clock source: External 24.576 MHz on AHCLKR (P9_28)
     * Bit clock: 3.072 MHz (24.576 MHz / 8)
     * Frame sync: 48 kHz (3.072 MHz / 64)
     * Format: I2S, 32-bit slots, left channel only
     */

    /* Step 0: Enable McASP0 clock via PRCM
     * Since the kernel driver is disabled, we must enable clocks ourselves.
     * Write 0x02 to enable module, then wait for IDLEST to show functional.
     */
    CM_PER_MCASP0_CLKCTRL = CM_PER_MODULEMODE_ENABLE;

    /* Wait for clock to stabilize (IDLEST bits [17:16] should be 0x00) */
    while ((CM_PER_MCASP0_CLKCTRL & 0x00030000) != CM_PER_IDLEST_FUNCTIONAL) {
        /* Wait for module to become functional */
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
     *
     * NOTE: External 24.576 MHz from CLKOUT2 is not available - CLKOUT2 only
     * outputs 32.768 kHz regardless of HDMI audio overlay setting.
     *
     * HCLKRM = 1: Internal clock source (mcasp0_fck)
     * HCLKRDIV = 0: No division
     *
     * Measured sample rate is ~36 kHz (not the expected 47 kHz from 24 MHz/8/64).
     * This suggests mcasp0_fck may not be running at 24 MHz when used by McASP.
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
     * Bit 0 = slot 0, Bit 1 = slot 1
     */
    MCASP_CFG_WRITE(MCASP_RTDM, 0x00000001);  /* Only slot 0 (left) */

    /* Step 8b: Configure transmit format, mask, and TDM slots
     * TX path generates clocks but doesn't need active serializer/state machine.
     * Kernel uses XTDM=0 even for RX-only operation.
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
     *
     * IMPORTANT: Must enable AFIFO BEFORE starting the receive state machine!
     * The kernel driver enables FIFO early in the initialization sequence.
     *
     * IMPORTANT: FIFO registers are at CFG_BASE + 0x1000 (0x48039xxx),
     * NOT at DATA_BASE + 0x1000!
     *
     * The AFIFO routes serializer data to the data port at 0x46000000.
     * Without AFIFO enabled, data goes only to RBUF which doesn't work
     * reliably for CPU polling.
     *
     * RFIFOCTL configuration:
     *   RNUMDMA [7:0] = 1: Transfer 1 word per "DMA event" (we poll instead)
     *   RNUMEVT [15:8] = 1: Generate event when 1 word in FIFO
     *   RENA [16] = 1: Enable receive FIFO
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
     * This fix is from the kernel driver patch that made ALSA work.
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

    /* Capture register values for debugging */
    ctrl->debug_gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    ctrl->debug_rstat = MCASP_CFG_READ(MCASP_RSTAT);
    ctrl->debug_pfunc = MCASP_CFG_READ(MCASP_PFUNC);
    ctrl->debug_pdir = MCASP_CFG_READ(MCASP_PDIR);
    ctrl->debug_aclkrctl = MCASP_CFG_READ(MCASP_ACLKRCTL);
    ctrl->debug_ahclkrctl = MCASP_CFG_READ(MCASP_AHCLKRCTL);
    ctrl->debug_srctl0 = MCASP_CFG_READ(MCASP_SRCTL0);
    ctrl->debug_loop_count = 0;

    ctrl->status = STATUS_MCASP_INIT;
}

/* Read sample from McASP FIFO data port
 * Returns 24-bit signed sample in 32-bit container
 * Blocks until data is available (with timeout tracking)
 *
 * The AFIFO routes serializer data to the data port at 0x46000000.
 * We poll RFIFOSTS (at CFG_BASE + 0x100C) to check when data is available,
 * then read directly from the data port.
 */
static inline int32_t read_mcasp_sample(struct audio_control_block *ctrl) {
    uint32_t loops = 0;
    uint32_t fifo_level;

    /* Wait for FIFO to have data (RFIFOSTS level > 0)
     * NOTE: RFIFOSTS is at CFG_BASE + 0x100C, not DATA_BASE!
     */
    while (1) {
        fifo_level = MCASP_CFG_READ(MCASP_RFIFOSTS) & MCASP_FIFOSTS_LEVEL_MASK;
        if (fifo_level > 0) {
            break;
        }

        loops++;
        /* Update debug every 1M iterations to show we're alive */
        if ((loops & 0xFFFFF) == 0) {
            ctrl->debug_loop_count = loops;
            ctrl->debug_rstat = MCASP_CFG_READ(MCASP_RSTAT);
        }
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
 * Main Function
 * =============================================================================
 */

void main(void) {
    struct audio_control_block *ctrl = (struct audio_control_block *)(PRU_SHARED_MEM + AUDIO_CONTROL_BLOCK);

    /* Sample buffers in shared memory */
    sample_t *buffer_a = (sample_t *)BUFFER_A_SHARED;
    sample_t *buffer_b = (sample_t *)BUFFER_B_SHARED;
    fft_buffer_t *fft_buf = (fft_buffer_t *)FFT_BUFFER_OFFSET;

    sample_t *current_buffer;
    sample_t *completed_buffer;
    uint32_t buffer_index;
    int32_t sample;
    uint32_t fft_end_time;

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
    ctrl->min_sample = 0x7FFFFFFF;   /* Max positive for signed */
    ctrl->max_sample = 0x80000000;   /* Max negative for signed */
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

        /* Store in buffer */
        current_buffer[buffer_index] = sample;
        buffer_index++;

        /* Update statistics */
        ctrl->last_sample = (uint32_t)sample;
        if (sample < (int32_t)ctrl->min_sample) ctrl->min_sample = (uint32_t)sample;
        if (sample > (int32_t)ctrl->max_sample) ctrl->max_sample = (uint32_t)sample;
        ctrl->total_samples++;

        /* Check if buffer is full */
        if (buffer_index >= BUFFER_SIZE) {
            ctrl->buffer_count++;

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

                /* Use FFT buffer as temp working space */
                int32_t *temp_buffer = (int32_t *)fft_buf;

                /* Remove DC offset (I2S samples are already signed) */
                remove_dc_offset_i2s(completed_buffer, temp_buffer, BUFFER_SIZE);

                /* Apply Blackman window */
                apply_blackman_window_32bit(temp_buffer, BUFFER_SIZE);

                /* Scale 24-bit samples to 16-bit for FFT
                 * Shift right by 8 bits to fit in Q15 format
                 */
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

                /* Run FFT */
                fft_init(fft_buf, (const int16_t *)completed_buffer);
                fft_compute(fft_buf);

                fft_end_time = PRU1_CTRL.CYCLE;
                ctrl->fft_time_cycles = fft_end_time;
                ctrl->fft_count++;

                /* Calculate frequency band magnitudes */
                {
                    uint16_t bin;
                    uint32_t mag;
                    uint32_t bass_sum = 0, midlow_sum = 0, midhigh_sum = 0, treble_sum = 0;
                    uint16_t bass_count = 0, midlow_count = 0, midhigh_count = 0, treble_count = 0;

                    const uint16_t bass_end = FREQ_TO_BIN(ctrl->bass_max_hz);
                    const uint16_t midlow_end = FREQ_TO_BIN(ctrl->midlow_max_hz);
                    const uint16_t midhigh_end = FREQ_TO_BIN(ctrl->midhigh_max_hz);
                    const uint16_t nyquist_bin = FFT_SIZE / 2;

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
