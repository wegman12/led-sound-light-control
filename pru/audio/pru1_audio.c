/*
 * PRU1 Audio Sampling Test Firmware
 *
 * Simple test firmware to verify PRU1 is running and accessible.
 * Toggles a status bit in shared memory every second.
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include "resource_table_empty.h"

/* Memory Layout */
#define PRU_SHARED_MEM      0x00010000
#define AUDIO_CONTROL_BLOCK 0x00002000  /* Offset 8KB in shared memory (after PRU0's 4KB) */

/* PRU Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define CYCLES_PER_SECOND   200000000

/* Status Codes */
#define STATUS_RUNNING      0x41554431  /* "AUD1" in ASCII - indicates PRU1 audio is running */

/* Audio Control Block Structure */
struct audio_control_block {
    volatile uint32_t status;           /* PRU running status */
    volatile uint32_t counter;          /* Simple counter for testing */
    volatile uint32_t toggle_bit;       /* Toggles every second */
    volatile uint32_t reserved[13];     /* Reserved for future use (64 bytes total) */
};

/* Delay function using PRU cycle counter */
static void delay_cycles(uint32_t cycles) {
    uint32_t start = PRU1_CTRL.CYCLE;
    while ((PRU1_CTRL.CYCLE - start) < cycles) {
        /* Wait */
    }
}

/* Main function - Simple test loop */
void main(void) {
    struct audio_control_block *ctrl = (struct audio_control_block *)(PRU_SHARED_MEM + AUDIO_CONTROL_BLOCK);

    /* Enable PRU cycle counter */
    PRU1_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port - allows PRU to access peripheral registers */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize control block */
    ctrl->status = STATUS_RUNNING;
    ctrl->counter = 0;
    ctrl->toggle_bit = 0;

    /* Main test loop - toggle bit every second */
    while (1) {
        /* Increment counter */
        ctrl->counter++;

        /* Toggle bit */
        ctrl->toggle_bit = (ctrl->toggle_bit == 0) ? 1 : 0;

        /* Wait 1 second (200 million cycles @ 200 MHz) */
        delay_cycles(CYCLES_PER_SECOND);
    }
}
