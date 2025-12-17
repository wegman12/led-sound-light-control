/*
 * PRU1 Raw Audio Capture Firmware
 *
 * High-speed raw ADC sampling from AIN1 at 40 kHz.
 * Writes unprocessed samples to circular buffer in shared memory for offline analysis.
 * No FFT, windowing, or smoothing - just pure ADC data capture.
 *
 * Purpose: Capture baseline and music samples for offline parameter optimization
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <pru_iep.h>
#include <sys_tscAdcSs.h>
#include "resource_table_empty.h"

/* Memory Layout */
#define PRU_SHARED_MEM       0x00010000
#define RAW_CONTROL_OFFSET   0x00000000  /* Start at beginning of shared memory */
#define RAW_BUFFER_OFFSET    0x00000100  /* Start buffer after 256-byte control block */

/* PRU and IEP Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define IEP_CLOCK_HZ        200000000

/* Sampling Configuration */
#define SAMPLE_RATE_HZ      40000       /* 40 kHz sampling rate */
#define SAMPLE_PERIOD_NS    25000       /* 25 μs = 25,000 ns */
#define IEP_CMP_VALUE       5000        /* 5,000 cycles @ 200 MHz = 25 μs */

/* Raw sample buffer configuration
 * Circular buffer in PRU shared memory
 * Each sample is 2 bytes (uint16_t)
 *
 * Total available shared memory: 12 KB (0x3000 bytes)
 * Control block: 256 bytes (0x100)
 * Remaining for samples: 0x3000 - 0x100 = 0x2F00 = 12032 bytes = 6016 samples
 * At 40 kHz, this is ~150 ms of buffering
 *
 * With 20ms polling interval, this gives 7.5x safety margin
 */
#define RAW_BUFFER_SIZE     6016        /* Number of uint16_t samples */
#define RAW_BUFFER_BYTES    (RAW_BUFFER_SIZE * 2)  /* Buffer size in bytes */

/* Control Module registers to enable the ADC peripheral */
#define CM_WKUP_CLKSTCTRL  (*((volatile uint32_t *)0x44E00400))
#define CM_WKUP_ADC_TSC_CLKCTRL  (*((volatile uint32_t *)0x44E004BC))

/* Status Codes */
#define STATUS_RUNNING      0x52415743  /* "RAWC" in ASCII - Raw Capture running */
#define STATUS_ADC_INIT     0x41444349  /* "ADCI" in ASCII - ADC initialized */
#define STATUS_IEP_INIT     0x49455049  /* "IEPI" in ASCII - IEP initialized */
#define STATUS_SAMPLING     0x53414D50  /* "SAMP" in ASCII - Sampling active */

/* Raw Capture Control Block Structure (in shared memory)
 * Layout: Configuration (written by host) + Status (written by PRU)
 */
struct raw_control_block {
    /* === Configuration Section (written by host, read by PRU) === */
    volatile uint32_t enable_capture;       /* 1 = capture enabled, 0 = paused */

    /* === Status Section (written by PRU, read by host) === */
    volatile uint32_t status;               /* PRU running status */
    volatile uint32_t total_samples;        /* Total samples collected */
    volatile uint32_t buffer_write_index;   /* Current write position in circular buffer */
    volatile uint32_t buffer_wrap_count;    /* Number of times buffer has wrapped */
    volatile uint32_t adc_timeouts;         /* ADC timeout errors */
    volatile uint32_t last_sample;          /* Most recent sample value */
    volatile uint32_t min_sample;           /* Minimum sample value */
    volatile uint32_t max_sample;           /* Maximum sample value */

    /* Padding to ensure control block is aligned */
    volatile uint32_t reserved[55];         /* Pad to 256 bytes total */
};

/* Sample buffer type (16-bit samples for 12-bit ADC) */
typedef uint16_t sample_t;

/* Initialize IEP timer for precise 40 kHz sampling */
static void init_iep_timer(void) {
    /* Disable IEP timer during configuration */
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
static void init_adc(struct raw_control_block *ctrl) {
    /* Enable ADC clock in Control Module */
    while (!(CM_WKUP_ADC_TSC_CLKCTRL == 0x02)) {
        CM_WKUP_CLKSTCTRL = 0;
        CM_WKUP_ADC_TSC_CLKCTRL = 0x02;
    }

    /* Disable ADC module so we can configure it */
    ADC_TSC.CTRL_bit.ENABLE = 0;

    /* Enable step configuration writes */
    ADC_TSC.CTRL_bit.STEPCONFIG_WRITEPROTECT_N_ACTIVE_LOW = 1;

    /* Configure ADC clock divider (0 = divide by 1, highest speed) */
    ADC_TSC.ADC_CLKDIV_bit.ADC_CLKDIV = 0;

    /* Configure Step 1 for AIN1 (Channel 1) sampling */
    ADC_TSC.STEPCONFIG1_bit.MODE = 0;  /* One-shot mode */
    ADC_TSC.STEPCONFIG1_bit.AVERAGING = 0;  /* No averaging */
    ADC_TSC.STEPCONFIG1_bit.SEL_INP_SWC_3_0 = 1;  /* AIN1 */
    ADC_TSC.STEPCONFIG1_bit.FIFO_SELECT = 0;  /* FIFO0 */

    /* Set step delays to 0 (fastest) */
    ADC_TSC.STEPDELAY1 = 0;

    /* Disable step configuration writes */
    ADC_TSC.CTRL_bit.STEPCONFIG_WRITEPROTECT_N_ACTIVE_LOW = 0;

    /* Enable channel ID tagging in FIFO data */
    ADC_TSC.CTRL_bit.STEP_ID_TAG = 1;

    /* Enable ADC module */
    ADC_TSC.CTRL_bit.ENABLE = 1;

    ctrl->status = STATUS_ADC_INIT;
}

/* Fast ADC sample read */
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

/* Main function - High-speed 40 kHz raw ADC sampling */
void main(void) {
    struct raw_control_block *ctrl = (struct raw_control_block *)(PRU_SHARED_MEM + RAW_CONTROL_OFFSET);
    sample_t *sample_buffer = (sample_t *)(PRU_SHARED_MEM + RAW_BUFFER_OFFSET);
    uint32_t write_index;
    uint16_t sample;

    /* Enable PRU cycle counter */
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port - allows PRU to access peripheral registers */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize configuration section with defaults */
    ctrl->enable_capture = 1;  /* Capture enabled by default */

    /* Initialize status section */
    ctrl->status = STATUS_RUNNING;
    ctrl->total_samples = 0;
    ctrl->buffer_write_index = 0;
    ctrl->buffer_wrap_count = 0;
    ctrl->adc_timeouts = 0;
    ctrl->last_sample = 0;
    ctrl->min_sample = 4095;   /* Start at max */
    ctrl->max_sample = 0;      /* Start at min */

    /* Initialize ADC */
    init_adc(ctrl);

    /* Initialize IEP timer for 40 kHz sampling */
    init_iep_timer();
    ctrl->status = STATUS_IEP_INIT;

    /* Set initial write index */
    write_index = 0;

    /* Trigger first ADC conversion */
    trigger_adc();

    /* Update status to indicate sampling is active */
    ctrl->status = STATUS_SAMPLING;

    /* Main sampling loop - 40 kHz continuous sampling */
    while (1) {
        /* Wait for IEP CMP0 event (40 kHz tick) */
        while ((CT_IEP.TMR_CMP_STS & 0x01) == 0) {
            /* Busy wait for CMP0 event */
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

        /* Only write to buffer if capture is enabled */
        if (ctrl->enable_capture) {
            /* Store sample in circular buffer */
            sample_buffer[write_index] = sample;
            write_index++;

            /* Update statistics */
            ctrl->last_sample = sample;
            if (sample < ctrl->min_sample && sample > 0) ctrl->min_sample = sample;
            if (sample > ctrl->max_sample) ctrl->max_sample = sample;
            ctrl->total_samples++;

            /* Handle circular buffer wrap */
            if (write_index >= RAW_BUFFER_SIZE) {
                write_index = 0;
                ctrl->buffer_wrap_count++;
            }

            /* Update write index in control block */
            ctrl->buffer_write_index = write_index;
        }
    }
}
