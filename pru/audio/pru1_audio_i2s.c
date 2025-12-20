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

/* Frequency Bin Boundaries (in Hz) - Configurable via control block */
#define BASS_MAX_HZ         150
#define MIDLOW_MAX_HZ       1000
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
 * Pre-computed Hann Window Coefficients
 * =============================================================================
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

/* Apply Hann window (same as ADC version) */
static void apply_hann_window_32bit(int32_t *buffer, uint16_t size) {
    uint16_t i;

    for (i = 0; i < size; i++) {
        int32_t window_coef;

        if (i < size / 2) {
            window_coef = hann_window_half[i];
        } else {
            window_coef = hann_window_half[size - 1 - i];
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

                /* Apply Hann window */
                apply_hann_window_32bit(temp_buffer, BUFFER_SIZE);

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
