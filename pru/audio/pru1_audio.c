/*
 * PRU1 Audio Sampling Firmware
 *
 * Samples audio from AIN1 using direct ADC hardware access.
 * Test version: Samples at ~10 Hz and stores results in shared memory.
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <sys_tscAdcSs.h>
#include "resource_table_empty.h"

/* Memory Layout */
#define PRU_SHARED_MEM      0x00010000
#define AUDIO_CONTROL_BLOCK 0x00002000  /* Offset 8KB in shared memory (after PRU0's 4KB) */

/* PRU Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define CYCLES_PER_SECOND   200000000

/* Control Module registers to enable the ADC peripheral */
#define CM_WKUP_CLKSTCTRL  (*((volatile uint32_t *)0x44E00400))
#define CM_WKUP_ADC_TSC_CLKCTRL  (*((volatile uint32_t *)0x44E004BC))

/* Status Codes */
#define STATUS_RUNNING      0x41554431  /* "AUD1" in ASCII - indicates PRU1 audio is running */
#define STATUS_ADC_INIT     0x41444349  /* "ADCI" in ASCII - ADC initialized */
#define STATUS_ADC_SAMPLING 0x41445353  /* "ADSS" in ASCII - ADC sampling */

/* Sample Buffer Size */
#define MAX_SAMPLES         32  /* Store last 32 samples */

/* Audio Control Block Structure */
struct audio_control_block {
    volatile uint32_t status;           /* PRU running status */
    volatile uint32_t sample_count;     /* Total samples collected */
    volatile uint32_t sample_index;     /* Current index in sample buffer (0-31) */
    volatile uint32_t adc_errors;       /* ADC error count */
    volatile uint32_t samples[MAX_SAMPLES];  /* Ring buffer of recent samples */
    volatile uint32_t reserved[12];     /* Reserved for future use (128 bytes total) */
};

/* PRU Cycle Counter Functions */
static inline void reset_counter(void) {
    /* Reset counter to 0 by disabling and re-enabling */
    PRU1_CTRL.CTRL_bit.CTR_EN = 0;
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Reset counter to 0 by writing directly to CYCLE register */
    PRU1_CTRL.CYCLE = 0;
}

/* Delay for approximately 100ms (10 Hz sampling rate) */
static void delay_100ms(void) {
    /* 100ms = 20,000,000 cycles @ 200 MHz */
    __delay_cycles(20000000);
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

/* Trigger and read one sample from ADC */
static uint32_t read_adc_sample(struct audio_control_block *ctrl) {
    uint32_t fifo_count;
    uint32_t fifo_data;
    uint32_t sample_value;
    uint32_t timeout;

    /* Clear any stale data from FIFO0 */
    fifo_count = ADC_TSC.FIFO0COUNT;
    while (fifo_count > 0) {
        fifo_data = ADC_TSC.FIFO0DATA;  /* Read and discard */
        fifo_count = ADC_TSC.FIFO0COUNT;
    }

    /* Trigger a new capture by enabling step 1 */
    ADC_TSC.STEPENABLE = (1 << 1);

    /* Wait for FIFO to have data (with timeout) */
    timeout = 10000;  /* ~50us timeout @ 200 MHz */
    while (ADC_TSC.FIFO0COUNT == 0 && timeout > 0) {
        timeout--;
    }

    if (timeout == 0) {
        /* Timeout - no data available */
        ctrl->adc_errors++;
        return 0;
    }

    /* Read from FIFO */
    fifo_data = ADC_TSC.FIFO0DATA;

    /* Extract 12-bit sample value from bits [11:0] */
    sample_value = fifo_data & 0xFFF;

    return sample_value;
}

/* Main function - ADC sampling loop */
void main(void) {
    struct audio_control_block *ctrl = (struct audio_control_block *)(PRU_SHARED_MEM + AUDIO_CONTROL_BLOCK);
    uint32_t sample;
    uint32_t i;

    /* Enable PRU cycle counter */
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port - allows PRU to access peripheral registers */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize control block */
    ctrl->status = STATUS_RUNNING;
    ctrl->sample_count = 0;
    ctrl->sample_index = 0;
    ctrl->adc_errors = 0;

    /* Clear sample buffer */
    for (i = 0; i < MAX_SAMPLES; i++) {
        ctrl->samples[i] = 0;
    }

    /* Initialize ADC */
    init_adc(ctrl);

    /* Update status to indicate sampling has started */
    ctrl->status = STATUS_ADC_SAMPLING;

    /* Main sampling loop - read ADC at ~10 Hz */
    while (1) {
        /* Reset cycle counter to avoid overflow */
        reset_counter();

        /* Read ADC sample */
        sample = read_adc_sample(ctrl);

        /* Store sample in ring buffer */
        ctrl->samples[ctrl->sample_index] = sample;

        /* Update index (wrap around at MAX_SAMPLES) */
        ctrl->sample_index = (ctrl->sample_index + 1) % MAX_SAMPLES;

        /* Increment total sample count */
        ctrl->sample_count++;

        /* Wait 100ms before next sample (10 Hz rate) */
        delay_100ms();
    }
}
