/*
 * PRU I2S Signal Monitor
 *
 * Monitors I2S pins and logs signal activity to detect if clocks are present
 *
 * Target: PRU0
 *
 * Pins to monitor:
 * - P9_29 (GPIO3_15, bit 15) - I2S Word Select (should be ~48 kHz)
 * - P9_31 (GPIO3_14, bit 14) - I2S Bit Clock (should be ~3 MHz)
 * - P9_30 (GPIO3_16, bit 16) - I2S Data
 *
 * Note: These pins are on GPIO Bank 3 (__R31 register in PRU)
 */

#include <stdint.h>
#include "resource_table_empty.h"

/* PRU built-in functions */
extern void __delay_cycles(unsigned int);
extern void __halt(void);

/* GPIO Bank 3 pins (PRU __R31 register) - Direct read, no offset needed */
#define WS_PIN   15  /* P9_29: Word Select */
#define SCK_PIN  14  /* P9_31: Bit Clock */
#define SD_PIN   16  /* P9_30: Data */

/* Shared memory for results */
#define SHARED_RAM_BASE 0x00010000

struct monitor_results {
    uint32_t ws_transitions;    /* Count of WS edge changes */
    uint32_t sck_transitions;   /* Count of SCK edge changes */
    uint32_t sd_transitions;    /* Count of SD changes */
    uint32_t sample_count;      /* Number of samples taken */
    uint32_t ws_high_time;      /* Cycles WS stayed high */
    uint32_t ws_low_time;       /* Cycles WS stayed low */
    uint32_t status;            /* 1=running, 0=stopped */
};

volatile register uint32_t __R31;

void main(void) {
    volatile struct monitor_results *results = (volatile struct monitor_results *)SHARED_RAM_BASE;

    /* Initialize results */
    results->ws_transitions = 0;
    results->sck_transitions = 0;
    results->sd_transitions = 0;
    results->sample_count = 0;
    results->ws_high_time = 0;
    results->ws_low_time = 0;
    results->status = 1;

    uint32_t prev_ws = 0;
    uint32_t prev_sck = 0;
    uint32_t prev_sd = 0;
    uint32_t ws_state_cycles = 0;
    uint32_t current_ws = 0;

    /* Monitor for 1 second (200M cycles at 200 MHz) */
    uint32_t max_samples = 200000000;

    while (results->sample_count < max_samples) {
        /* Read current pin states directly from __R31 */
        uint32_t pins = __R31;
        current_ws = (pins >> WS_PIN) & 1;
        uint32_t current_sck = (pins >> SCK_PIN) & 1;
        uint32_t current_sd = (pins >> SD_PIN) & 1;

        /* Count transitions */
        if (current_ws != prev_ws) {
            results->ws_transitions++;

            /* Record time spent in previous state */
            if (prev_ws) {
                results->ws_high_time = ws_state_cycles;
            } else {
                results->ws_low_time = ws_state_cycles;
            }
            ws_state_cycles = 0;
        } else {
            ws_state_cycles++;
        }

        if (current_sck != prev_sck) {
            results->sck_transitions++;
        }

        if (current_sd != prev_sd) {
            results->sd_transitions++;
        }

        prev_ws = current_ws;
        prev_sck = current_sck;
        prev_sd = current_sd;
        results->sample_count++;

        /* Sample every 1000 cycles (5 microseconds) to reduce overhead */
        __delay_cycles(1000);
    }

    results->status = 0;  /* Done */
    __halt();
}
