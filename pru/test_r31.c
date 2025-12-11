/*
 * Simple PRU test to check if __R31 works at all
 * Based on PRU Cookbook input example
 */
#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <rsc_types.h>

/* PRU input/output registers */
volatile register uint32_t __R30;
volatile register uint32_t __R31;

/* Resource Table - REQUIRED by RemoteProc */
struct my_resource_table {
    struct resource_table base;
    uint32_t offset[0];
} resourceTable __attribute__((section(".resource_table"), retain)) = {
    {
        1,              /* version */
        0,              /* number of entries */
        {0, 0},         /* reserved */
    },
};

/* Shared memory for monitoring */
#define PRU_SHARED_MEM      0x00010000
#define CONTROL_BLOCK       0x00001000

struct control_block {
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t event_count;
    volatile uint32_t error_count;
    volatile uint32_t overrun_count;
    volatile uint32_t status;
};

void main(void)
{
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    uint32_t led;
    uint32_t sw;

    /* Clear SYSCFG[STANDBY_INIT] to enable OCP master port */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Configure GPI mode for direct connect */
    CT_CFG.GPCFG0 = 0x0000;
    CT_CFG.GPCFG1 = 0x0000;

    /* Initialize control block */
    ctrl->write_index = 0;
    ctrl->read_index = 0;
    ctrl->event_count = 0;
    ctrl->error_count = 0;
    ctrl->overrun_count = 0;
    ctrl->status = 1;

    led = 0x1<<0;	// P9_31 output bit 0
    sw  = 0x1<<7;	// P9_25 input bit 7

    while (1) {
        /* Write full __R31 value to shared memory for monitoring */
        ctrl->event_count = __R31;
        ctrl->error_count = (__R31 & sw) >> 7;  /* Bit 7 specifically */
        ctrl->overrun_count = __R31 & 0xFF;     /* Lower 8 bits */

        /* Simple LED control from switch (cookbook example) */
        if((__R31 & sw) == sw) {
            __R30 |= led;		// Turn on LED
        } else {
            __R30 &= ~led;		// Turn off LED
        }

        __delay_cycles(100);  /* Small delay */
    }
}
