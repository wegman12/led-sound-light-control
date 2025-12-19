/*
 * PRU0 IR Remote Detector Firmware
 *
 * Detects IR remote signals on P9_24 (PRU0 __R31 bit 16), decodes 34-bit NEC-like protocol,
 * matches against 44 known button codes, and writes events to shared memory
 * ring buffer for consumption by Go application.
 *
 * IMPORTANT: P9_24 IR input is ACTIVE-LOW (pulled high by default)
 *   - GPIO HIGH (1) = No signal (idle state)
 *   - GPIO LOW (0)  = Signal present (IR transmitting)
 *
 * Note: P9_24 chosen instead of P9_27 to avoid interference from
 *       adjacent I2S clock signals (P9_25-31 cluster).
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU0 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include <rsc_types.h>
#include <pru_iep.h>
#include <pru_intc.h>
#include "../pru_shared_memory.h"

/* PRU input/output registers */
volatile register uint32_t __R30;
volatile register uint32_t __R31;

/* Resource Table - REQUIRED by RemoteProc */
struct my_resource_table {
    struct resource_table base;
    uint32_t offset[0];  /* No additional resources */
} resourceTable __attribute__((section(".resource_table"), retain)) = {
    {
        1,              /* version */
        0,              /* number of entries */
        {0, 0},         /* reserved */
    },
};

/* PRU Configuration Registers */
#define PRUSS_CFG_BASE  0x00026000
#define SYSCFG_OFFSET   0x04
#define GPCFG0_OFFSET   0x08  /* GPI configuration for PRU0 */
#define SYSCFG_ADDR     ((volatile uint32_t *)(PRUSS_CFG_BASE + SYSCFG_OFFSET))
#define GPCFG0_ADDR     ((volatile uint32_t *)(PRUSS_CFG_BASE + GPCFG0_OFFSET))

/* SYSCFG register bits */
#define SYSCFG_STANDBY_INIT  (1 << 4)
#define SYSCFG_IDLE_MODE     (0 << 2)  /* No idle mode */
#define SYSCFG_SUB_MWAIT     (1 << 5)  /* Enable OCP master ports */

/* GPCFG0 register values */
#define GPCFG_PRU_GPI_MODE  0x0  /* Direct connect mode - GPI pins connect to __R31 */

/* Memory Layout - uses shared header for consistency with PRU1 */
#define PRU_SHARED_MEM      PRU_SHARED_MEM_BASE
#define BUTTON_RING_BUFFER  PRU0_IR_RING_BUFFER_OFFSET
#define CONTROL_BLOCK       PRU0_IR_CONTROL_OFFSET
#define RING_BUFFER_SIZE    PRU0_IR_RING_BUFFER_ENTRIES

/* PRU Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define CYCLES_PER_US       200         /* 200 cycles per microsecond */

/* Timing Thresholds (in cycles) */
#define THRESHOLD_1MS       200000        /* 1000 μs = 200,000 cycles */
#define MAX_BIT_WAIT        80000000     /* 40 ms = 80M cycles */
#define MAX_PACKET_DURATION 200000000    /* 100 ms = .2B cycles */
#define COOLDOWN_AFTER_SUCCESS 40000000  /* 200 ms cooldown after successful detection */

/* Protocol Constants */
#define IR_PACKET_LENGTH    34
#define NUM_BUTTON_CODES    44

/* Control Block Structure */
struct control_block {
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t event_count;
    volatile uint32_t error_count;
    volatile uint32_t overrun_count;
    volatile uint32_t status;      /* PRU running status (non-zero = running) */
    volatile uint32_t error_code;  /* Last error code for debugging */
};

/* Status Codes */
#define STATUS_RUNNING          0x52554E  /* "RUN" in ASCII - indicates PRU is running */

/* Error Codes */
#define ERROR_NONE              0x0000
#define ERROR_LEADER_LOW        0x0001  /* Leader LOW pulse timeout */
#define ERROR_LEADER_HIGH       0x0002  /* Leader HIGH space timeout */
#define ERROR_FIRST_LOW         0x0003  /* First data LOW pulse timeout */
#define ERROR_DATA_HIGH         0x0004  /* Data HIGH space timeout */
#define ERROR_DATA_LOW          0x0005  /* Data LOW pulse timeout */
#define ERROR_START_BIT         0x0006  /* START bit validation failed */
#define ERROR_HEADER_BITS       0x0007  /* HEADER bits validation failed */
#define ERROR_SEPARATOR_BITS    0x0008  /* SEPARATOR bits validation failed */
#define ERROR_NO_MATCH          0x0009  /* No button code match */

/* Button Event Structure (8 bytes) */
struct button_event {
    uint8_t button_type;
    uint8_t reserved1;
    uint16_t reserved2;
    uint32_t timestamp;
};

/* PRU Direct GPIO Access via __R31 register */
/* P9_24 maps to PRU0 bit 16 - much faster and more reliable than memory-mapped GPIO */
/* No volatile tricks needed - __R31 is a hardware register designed for this */
static inline uint32_t read_gpio(void) {
    /* Read PRU0 input register bit 16 (P9_24) directly */
    return (__R31 & 0x10000) >> 16;
}

/* PRU Cycle Counter Functions */
static inline void reset_counter(void) {
    /* Reset counter to 0 by disabling and re-enabling */
    PRU0_CTRL.CTRL_bit.CTR_EN = 0;
    PRU0_CTRL.CTRL_bit.CTR_EN = 1;

    /* Reset counter to 0 by writing directly to CYCLE register */
    PRU0_CTRL.CYCLE = 0;
}

static inline uint32_t read_counter(void) {
    return PRU0_CTRL.CYCLE;
}

/* Button Code Lookup Table (from api/remote/codes.go) */
static const uint8_t button_codes[NUM_BUTTON_CODES][IR_PACKET_LENGTH] = {
    /* PowerButtonType (0) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1, 1, 0, 1, 1},
    /* PauseButtonType (1) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 0, 1, 1},
    /* BrightnessDownButtonType (2) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1},
    /* BrightnessUpButtonType (3) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1, 0, 1, 1},
    /* RedButtonType (4) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1, 1},
    /* GreenButtonType (5) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1},
    /* BlueButtonType (6) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 1},
    /* WhiteButtonType (7) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 0, 1, 1},
    /* PinkButtonType (8) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 1, 0, 1, 1},
    /* LightBlueButtonType (9) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1, 1},
    /* LightGreenButtonType (10) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 1, 1},
    /* OrangeButtonType (11) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 1, 1},
    /* LightOrangeButtonType (12) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 0, 1, 0, 1, 1},
    /* GreenBlueButtonType (13) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1, 1},
    /* IndigoButtonType (14) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 1, 1},
    /* LightPinkButtonType (15) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1, 1},
    /* SkyBlueButtonType (16) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1},
    /* VioletButtonType (17) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 1},
    /* TealButtonType (18) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1},
    /* GoldButtonType (19) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 1},
    /* YellowButtonType (20) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1, 1},
    /* DarkTealButtonType (21) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 1},
    /* PurpleButtonType (22) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 1},
    /* LightSkyBlueButtonType (23) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1},
    /* QuickButtonType (24) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1},
    /* BlueUpButtonType (25) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 1},
    /* GreenUpButtonType (26) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1},
    /* RedUpButtonType (27) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 1},
    /* RedDownButtonType (28) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1},
    /* GreenDownButtonType (29) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1},
    /* BlueDownButtonType (30) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 1},
    /* SlowButtonType (31) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1},
    /* AutoButtonType (32) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
    /* Diy3ButtonType (33) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1, 1},
    /* Diy2ButtonType (34) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1},
    /* Diy1ButtonType (35) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 1, 1},
    /* Diy4ButtonType (36) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1, 1},
    /* Diy5ButtonType (37) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1},
    /* Diy6ButtonType (38) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1},
    /* FlashButtonType (39) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1},
    /* Fade7ButtonType (40) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1},
    /* Fade3ButtonType (41) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 1},
    /* Jump7ButtonType (42) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1, 1},
    /* Jump3ButtonType (43) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1}
};

/* Helper Functions */
static int all_zeros(const uint8_t *arr, int count) {
    int i;
    for (i = 0; i < count; i++) {
        if (arr[i] != 0) return 0;
    }
    return 1;
}

static int all_ones(const uint8_t *arr, int count) {
    int i;
    for (i = 0; i < count; i++) {
        if (arr[i] != 1) return 0;
    }
    return 1;
}

/* Wait for GPIO pin to reach target state or timeout */
/* Uses 800ns delay between reads - this was critical to get clean GPIO readings */
static uint32_t wait_for_state(uint32_t target_state, uint32_t max_cycles, int *timed_out) {
    uint32_t start = read_counter();
    uint32_t elapsed;
    /* 100ns delay between reads to match working raw GPIO test (160 cycles @ 200MHz) */
    __delay_cycles(20);

    while (read_gpio() != target_state) {
        elapsed = read_counter() - start;
        if (elapsed > max_cycles) {
            *timed_out = 1;
            return elapsed;
        }
        /* 100ns delay between reads to match working raw GPIO test (160 cycles @ 200MHz) */
        __delay_cycles(20);
    }

    *timed_out = 0;
    return read_counter() - start;
}

/* Measure how long GPIO STAYS in current state (matches Go readWhileValue behavior) */
/* Assumes GPIO is currently in current_state, measures until it changes */
static uint32_t measure_state_duration(uint32_t current_state, uint32_t max_cycles, int *timed_out) {
    uint32_t start = read_counter();
    uint32_t elapsed;

    /* Critical: Delay before first read to allow signal to stabilize after transition */
    /* Without this, we get metastability issues reading too soon after state change */
    /* Timer already started, so this delay is included in the measurement (correct) */
    __delay_cycles(20);

    while (read_gpio() == current_state) {  /* Loop WHILE in current state */
        elapsed = read_counter() - start;
        if (elapsed > max_cycles) {
            *timed_out = 1;
            return elapsed;
        }
        /* 800ns delay between reads */
        __delay_cycles(20);
    }

    *timed_out = 0;
    return read_counter() - start;
}

/* Decode bit based on low duration */
static int decode_bit(uint32_t low_cycles) {
    return (low_cycles >= THRESHOLD_1MS) ? 1 : 0;
}

/* Match decoded bits against known button codes */
static int match_button_code(const uint8_t bits[IR_PACKET_LENGTH]) {
    int i, j;

    for (i = 0; i < NUM_BUTTON_CODES; i++) {
        int match = 1;
        for (j = 0; j < IR_PACKET_LENGTH; j++) {
            if (bits[j] != button_codes[i][j]) {
                match = 0;
                break;
            }
        }
        if (match) return i;
    }

    return -1;  /* No match */
}

/* Read and decode 34-bit IR packet */
static int read_ir_packet(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    uint8_t bits[IR_PACKET_LENGTH];
    uint32_t durations[IR_PACKET_LENGTH];
    int i;
    int timed_out;

    /* GPIO is already LOW when this function is called (detected in main loop) */
    /* NEC protocol: 9ms LOW leader + 4.5ms HIGH space + data bits */
    /* We're somewhere in the 9ms leader - wait for it to finish (up to 10ms timeout) */
    wait_for_state(1, 2000000, &timed_out);  /* 10ms = 2M cycles */
    if (timed_out) {
        ctrl->error_code = ERROR_LEADER_LOW;
        return -1;
    }

    /* Now wait for the 4.5ms HIGH space to complete (up to 6ms timeout) */
    wait_for_state(0, 1200000, &timed_out);  /* 6ms = 1.2M cycles */
    if (timed_out) {
        ctrl->error_code = ERROR_LEADER_HIGH;
        return -1;
    }

    /* GPIO is now LOW - we've skipped measuring the leader HIGH space */
    /* That leader HIGH space IS the START bit (always long = 1) */
    bits[0] = 1;  /* START bit - we use it for synchronization but don't measure it */

    /* Wait for this first LOW pulse to complete, positioning us for the loop */
    wait_for_state(1, MAX_BIT_WAIT, &timed_out);
    if (timed_out) {
        ctrl->error_code = ERROR_FIRST_LOW;
        return -1;
    }

    /* Now read the remaining 32 bits (bits 1-32) */
    for (i = 1; i < IR_PACKET_LENGTH; i++) {
        /* Measure HIGH space duration */
        uint32_t high_cycles = measure_state_duration(1, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {
            ctrl->error_code = ERROR_DATA_HIGH;
            return -1;
        }

        /* Wait for GPIO to return HIGH (end of LOW pulse) */
        wait_for_state(1, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {
            ctrl->error_code = ERROR_DATA_LOW;
            return -1;
        }

        /* Store duration and decode bit */
        bits[i] = decode_bit(high_cycles);
    }

    /* Validate protocol structure */
    if (bits[0] != 1) {
        ctrl->error_code = ERROR_START_BIT;
        return -1;
    }
    if (!all_zeros(&bits[1], 8)) {
        ctrl->error_code = ERROR_HEADER_BITS;
        return -1;
    }
    if (!all_ones(&bits[9], 8)) {
        ctrl->error_code = ERROR_SEPARATOR_BITS;
        return -1;
    }

    /* Match against known button codes */
    int button_code = match_button_code(bits);
    if (button_code < 0) {
        ctrl->error_code = ERROR_NO_MATCH;
        return -1;
    }

    /* Success - clear error code */
    ctrl->error_code = ERROR_NONE;
    return button_code;
}

/* Write button event to ring buffer */
static void write_button_event(uint8_t button_type) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    struct button_event *events = (struct button_event *)(PRU_SHARED_MEM + BUTTON_RING_BUFFER);

    uint32_t next_write = (ctrl->write_index + 1) % RING_BUFFER_SIZE;

    /* Check for buffer full */
    if (next_write == ctrl->read_index) {
        ctrl->overrun_count++;
        return;  /* Drop event */
    }

    /* Write event */
    events[ctrl->write_index].button_type = button_type;
    events[ctrl->write_index].timestamp = read_counter();
    events[ctrl->write_index].reserved1 = 0;
    events[ctrl->write_index].reserved2 = 0;

    /* Update write index (atomic on PRU) */
    ctrl->write_index = next_write;
    ctrl->event_count++;
}

/*
 * ORIGINAL DETECTION LOGIC - NOW USING PRU DIRECT GPIO
 * Migrated to P9_24 (PRU0 __R31 bit 16) for direct hardware register access
 */
#if 1
static void run_ir_detection_loop(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    int button_code;
    int timed_out;
    int loop_count = 0;

    /* Small delay to let debug tool read initial GPCFG0 values */
    __delay_cycles(20);  /* 100ns delay */

    /* Main IR detection loop */
    while (1) {
        /* Wait for GPIO LOW (signal present) - matches Go's trigger condition */
        if (read_gpio() == 0) {
            /* Reset counter before each detection to prevent hitting max value */
            reset_counter();

            /* Attempt to read and decode the IR packet */
            button_code = read_ir_packet();

            if (button_code >= 0) {
                /* Valid button detected */
                write_button_event((uint8_t)button_code);

                /* Wait for GPIO to return HIGH (idle) */
                wait_for_state(1, MAX_PACKET_DURATION, &timed_out);

                __delay_cycles(COOLDOWN_AFTER_SUCCESS);
            } else {
                ctrl->error_count++;

                /* Wait for GPIO to return HIGH (idle) before looking for next packet */
                wait_for_state(1, MAX_PACKET_DURATION, &timed_out);
            }
        }

        /* 100ns delay between main loop iterations for stable GPIO reads */
        __delay_cycles(20);
    }
}
#endif

/* Main function - IR Button Detection */
void main(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);

    /* Enable PRU cycle counter - required for timing measurements */
    PRU0_CTRL.CTRL_bit.CTR_EN = 1;

    /* Enable OCP master port - allows PRU to access peripheral registers */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Configure GPI mode for PRU0 - direct connect mode (GPI pins -> __R31) */
    CT_CFG.GPCFG0 = 0x0000;  /* Direct connect, 16-bit parallel capture disabled */

    /* Initialize control block */
    ctrl->write_index = 0;
    ctrl->read_index = 0;
    ctrl->event_count = 0;
    ctrl->error_count = 0;
    ctrl->overrun_count = 0;
    ctrl->status = STATUS_RUNNING;
    ctrl->error_code = ERROR_NONE;

    /* Run the IR detection loop (never returns) */
    run_ir_detection_loop();
}
