/*
 * PRU0 IR Remote Detector Firmware
 *
 * Detects IR remote signals on P9_27 (PRU0 __R31 bit 5), decodes 34-bit NEC-like protocol,
 * matches against 44 known button codes, and writes events to shared memory
 * ring buffer for consumption by Go application.
 *
 * IMPORTANT: P9_27 IR input is ACTIVE-LOW (pulled high by default)
 *   - GPIO HIGH (1) = No signal (idle state)
 *   - GPIO LOW (0)  = Signal present (IR transmitting)
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

/* Memory Layout */
#define PRU_SHARED_MEM      0x00010000
#define BUTTON_RING_BUFFER  0x00000000  /* Offset 0: Ring buffer */
#define CONTROL_BLOCK       0x00001000  /* Offset 4KB: Control structure */
#define RING_BUFFER_SIZE    256         /* 256 events */
#define TIMING_DATA_OFFSET  0x00002000  /* Offset 8KB: Raw timing data */
#define MAX_TIMING_SAMPLES  512         /* Maximum transitions to capture */
#define GPIO_RAW_DATA_OFFSET 0x00002000 /* Offset 8KB: Raw GPIO readings */
#define MAX_GPIO_SAMPLES    8192        /* 8K samples of raw GPIO data */

/* PRU Clock: 200 MHz = 5 ns per cycle */
#define PRU_CLOCK_HZ        200000000
#define CYCLES_PER_US       200         /* 200 cycles per microsecond */

/* Timing Thresholds (in cycles) */
#define THRESHOLD_1MS       200000      /* 1000 μs = 200,000 cycles */
#define MAX_BIT_WAIT        800000000     /* 400 ms = 40M cycles */
#define MAX_PACKET_DURATION 2000000000    /* 1000 ms = 200M cycles */
#define IDLE_DELAY          1000        /* 5 μs between checks */
#define TIMEOUT_1_SECOND    200000000   /* 1 second = 200M cycles */
#define DEBOUNCE_CYCLES     2000        /* 10 μs debounce = 2,000 cycles */

/* Protocol Constants */
#define IR_PACKET_LENGTH    33
#define NUM_BUTTON_CODES    44

/* Control Block Structure */
struct control_block {
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t event_count;
    volatile uint32_t error_count;
    volatile uint32_t overrun_count;
    volatile uint32_t status;
};

/* Button Event Structure (8 bytes) */
struct button_event {
    uint8_t button_type;
    uint8_t reserved1;
    uint16_t reserved2;
    uint32_t timestamp;
};

/* Timing Sample Structure (8 bytes) */
struct timing_sample {
    uint32_t state;       /* 0 or 1 */
    uint32_t duration;    /* Duration in cycles */
};

/* Timing Capture Structure */
struct timing_data {
    volatile uint32_t sample_count;
    volatile uint32_t complete;
    struct timing_sample samples[MAX_TIMING_SAMPLES];
};

/* Raw GPIO Data Structure - Simple byte array for raw readings */
struct gpio_raw_data {
    volatile uint32_t sample_count;
    volatile uint32_t complete;
    volatile uint32_t first_full_register_value;  /* Store first full 32-bit read for debugging */
    volatile uint8_t samples[MAX_GPIO_SAMPLES];  /* Each byte is a GPIO reading (0 or 1) */
};

/* Debug structure for captured IR bits - at offset 0x1100 (after control block) */
#define DEBUG_BITS_OFFSET 0x00001100
struct debug_bits_data {
    volatile uint32_t valid;          /* 1 if data is valid, 0 otherwise */
    volatile uint32_t error_code;     /* Error code if decoding failed */
    volatile uint8_t bits[33];        /* The 33 bits that were captured */
    volatile uint32_t durations[33];  /* LOW pulse durations in cycles for each bit */
};

/* PRU Direct GPIO Access via __R31 register */
/* P9_27 maps to PRU0 bit 5 - much faster and more reliable than memory-mapped GPIO */
/* No volatile tricks needed - __R31 is a hardware register designed for this */
static inline uint32_t read_gpio(void) {
    /* Read PRU0 input register bit 5 (P9_27) directly */
    return (__R31 & 0x20) >> 5;
}

/* PRU Cycle Counter Access */
static inline uint32_t get_cycles(void) {
    return PRU0_CTRL.CYCLE;
}

/* Button Code Lookup Table (from api/remote/codes.go) */
static const uint8_t button_codes[NUM_BUTTON_CODES][IR_PACKET_LENGTH] = {
    /* PowerButtonType (0) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1, 1, 0, 1},
    /* PauseButtonType (1) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1, 0, 1},
    /* BrightnessDownButtonType (2) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1},
    /* BrightnessUpButtonType (3) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1, 0, 1},
    /* RedButtonType (4) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1, 0, 0, 1, 0, 1},
    /* GreenButtonType (5) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1},
    /* BlueButtonType (6) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1},
    /* WhiteButtonType (7) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 0, 1},
    /* PinkButtonType (8) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1, 0, 1, 1, 0, 1},
    /* LightBlueButtonType (9) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 1, 1, 0, 1},
    /* LightGreenButtonType (10) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 1, 0, 0, 1, 0, 1, 0, 1, 0, 1},
    /* OrangeButtonType (11) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 0, 1},
    /* LightOrangeButtonType (12) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1, 0, 1, 0, 1},
    /* GreenBlueButtonType (13) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1, 0, 1, 0, 1},
    /* IndigoButtonType (14) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 1},
    /* LightPinkButtonType (15) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 0, 1},
    /* SkyBlueButtonType (16) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1},
    /* VioletButtonType (17) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1},
    /* TealButtonType (18) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1},
    /* GoldButtonType (19) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1},
    /* YellowButtonType (20) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 1, 0, 0, 0, 1, 1, 1, 0, 0, 1, 1, 1},
    /* DarkTealButtonType (21) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1},
    /* PurpleButtonType (22) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 1, 0, 0, 0, 1, 0, 1, 0, 0, 1, 1, 1},
    /* LightSkyBlueButtonType (23) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1},
    /* QuickButtonType (24) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1},
    /* BlueUpButtonType (25) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 0, 1, 0, 1, 1, 1},
    /* GreenUpButtonType (26) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1},
    /* RedUpButtonType (27) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1, 1, 1},
    /* RedDownButtonType (28) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1},
    /* GreenDownButtonType (29) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1},
    /* BlueDownButtonType (30) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 1, 0, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1},
    /* SlowButtonType (31) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1},
    /* AutoButtonType (32) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1},
    /* Diy3ButtonType (33) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 0, 0, 1, 1, 1, 1},
    /* Diy2ButtonType (34) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1},
    /* Diy1ButtonType (35) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 1, 1},
    /* Diy4ButtonType (36) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 1, 0, 0, 0, 0, 1, 1, 1, 0, 1, 1, 1, 1},
    /* Diy5ButtonType (37) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1},
    /* Diy6ButtonType (38) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 0, 1, 0, 1, 1, 1, 1},
    /* FlashButtonType (39) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1},
    /* Fade7ButtonType (40) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1},
    /* Fade3ButtonType (41) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 1, 1, 1, 1, 1},
    /* Jump7ButtonType (42) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 1},
    /* Jump3ButtonType (43) */
    {1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 1, 1, 1}
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
    uint32_t start = get_cycles();
    uint32_t elapsed;

    while (read_gpio() != target_state) {
        elapsed = get_cycles() - start;
        if (elapsed > max_cycles) {
            *timed_out = 1;
            return elapsed;
        }
        /* 800ns delay between reads to match working raw GPIO test (160 cycles @ 200MHz) */
        __delay_cycles(160);
    }

    *timed_out = 0;
    return get_cycles() - start;
}

/* Measure how long GPIO STAYS in current state (matches Go readWhileValue behavior) */
/* Assumes GPIO is currently in current_state, measures until it changes */
static uint32_t measure_state_duration(uint32_t current_state, uint32_t max_cycles, int *timed_out) {
    uint32_t start = get_cycles();  /* Start timer immediately */
    uint32_t elapsed;

    /* Critical: Delay before first read to allow signal to stabilize after transition */
    /* Without this, we get metastability issues reading too soon after state change */
    /* Timer already started, so this delay is included in the measurement (correct) */
    __delay_cycles(160);

    while (read_gpio() == current_state) {  /* Loop WHILE in current state */
        elapsed = get_cycles() - start;
        if (elapsed > max_cycles) {
            *timed_out = 1;
            return elapsed;
        }
        /* 800ns delay between reads */
        __delay_cycles(160);
    }

    *timed_out = 0;
    return get_cycles() - start;
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

/* Read and decode 33-bit IR packet */
static int read_ir_packet(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    struct debug_bits_data *debug = (struct debug_bits_data *)(PRU_SHARED_MEM + DEBUG_BITS_OFFSET);
    uint8_t bits[IR_PACKET_LENGTH];
    int i;
    int timed_out;
    uint32_t packet_start = get_cycles();

    /* GPIO is already LOW when this function is called (detected in main loop) */
    /* NEC protocol: 9ms LOW leader + 4.5ms HIGH space + data bits */
    /* We're somewhere in the 9ms leader - wait for it to finish (up to 10ms timeout) */
    wait_for_state(1, 2000000, &timed_out);  /* 10ms = 2M cycles */
    if (timed_out) {
        ctrl->event_count = 0xFFFF;  /* Leader LOW timeout */
        ctrl->overrun_count = 0;
        return -1;
    }

    /* Now wait for the 4.5ms HIGH space to complete (up to 6ms timeout) */
    wait_for_state(0, 1200000, &timed_out);  /* 6ms = 1.2M cycles */
    if (timed_out) {
        ctrl->event_count = 0xFFFE;  /* Leader HIGH timeout */
        ctrl->overrun_count = 0;
        return -1;
    }

    /* GPIO is now LOW - we've skipped measuring the leader HIGH space */
    /* That leader HIGH space IS the START bit (always long = 1) */
    bits[0] = 1;  /* START bit - we use it for synchronization but don't measure it */
    uint32_t durations[IR_PACKET_LENGTH];
    durations[0] = 0;  /* No duration measured for START bit */

    /* Wait for this first LOW pulse to complete, positioning us for the loop */
    wait_for_state(1, MAX_BIT_WAIT, &timed_out);
    if (timed_out) {
        ctrl->event_count = 0xFFFD;  /* First bit LOW timeout */
        ctrl->overrun_count = 0;
        return -1;
    }

    /* Now read the remaining 32 bits (bits 1-32) */
    for (i = 1; i < IR_PACKET_LENGTH; i++) {
        /* Check packet timeout */
        if ((get_cycles() - packet_start) > MAX_PACKET_DURATION) {
            ctrl->event_count = 0xFFFD;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Wait for GPIO to go LOW (end of HIGH space period) */
        wait_for_state(0, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {
            ctrl->event_count = 0xFFFC;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Measure how long GPIO STAYS LOW (the mark/pulse duration) */
        uint32_t low_cycles = measure_state_duration(0, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {
            ctrl->event_count = 0xFFFB;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Store duration for debugging */
        durations[i] = low_cycles;

        /* Decode based on LOW duration (mark) - matches Go code behavior */
        bits[i] = decode_bit(low_cycles);
    }

    /* Copy bits and durations to debug structure for inspection */
    for (i = 0; i < IR_PACKET_LENGTH; i++) {
        debug->bits[i] = bits[i];
        debug->durations[i] = durations[i];
    }

    /* Test pattern to verify struct alignment - write known values */
    debug->durations[0] = 0x11111111;  /* 286331153 */
    debug->durations[1] = 0x22222222;  /* 572662306 */
    debug->durations[2] = 0x33333333;  /* 858993459 */
    debug->durations[3] = 0x44444444;  /* 1145324612 */

    /* Validate protocol structure */
    if (bits[0] != 1) {
        ctrl->event_count = 0xFFDF;
        debug->error_code = 0xFFDF;
        return -1;
    }                          /* START bit */
    if (!all_zeros(&bits[1], 8)) {
        ctrl->event_count = 0xFFDE;
        debug->error_code = 0xFFDE;
        return -1;
    }               /* HEADER (bits 1-8) */
    if (!all_ones(&bits[9], 8)) {
        ctrl->event_count = 0xFFDD;
        debug->error_code = 0xFFDD;
        return -1;
    }               /* SEPARATOR (bits 9-16) */
    if (bits[32] != 1) {

        ctrl->event_count = 0xFFC;
        return -1;
    }                        /* STOP bit */

    /* Match against known button codes */
    return match_button_code(bits);
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
    events[ctrl->write_index].timestamp = get_cycles();
    events[ctrl->write_index].reserved1 = 0;
    events[ctrl->write_index].reserved2 = 0;

    /* Update write index (atomic on PRU) */
    ctrl->write_index = next_write;
    ctrl->event_count++;
}

/*
 * ORIGINAL DETECTION LOGIC - NOW USING PRU DIRECT GPIO
 * Migrated to P9_27 (PRU0 __R31 bit 5) for direct hardware register access
 */
#if 1
static void run_ir_detection_loop(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    int button_code;
    int timed_out;
    int loop_count = 0;

    /* Small delay to let debug tool read initial GPCFG0 values */
    __delay_cycles(2000000);  /* 10ms delay */

    /* Main IR detection loop */
    while (1) {
        /* Wait for GPIO LOW (signal present) - matches Go's trigger condition */
        if (read_gpio() == 0) {
            /* Attempt to read and decode the IR packet */
            button_code = read_ir_packet();

            if (button_code >= 0) {
                /* Valid button detected */
                write_button_event((uint8_t)button_code);
            } else {
                ctrl->error_count++;
            }

            /* Wait for GPIO to return HIGH (idle) before looking for next packet */
            wait_for_state(1, MAX_PACKET_DURATION, &timed_out);
        }

        /* 100ns delay between main loop iterations for stable GPIO reads */
        __delay_cycles(20);
    }
}
#endif

/* Main function - IR Button Detection */
void main(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    struct debug_bits_data *debug = (struct debug_bits_data *)(PRU_SHARED_MEM + DEBUG_BITS_OFFSET);
    int i;

    /* Enable OCP master port - allows PRU to access peripheral registers like GPIO */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Configure GPI mode for PRU0/PRU1 - direct connect mode (GPI pins -> __R31) */
    uint32_t gpcfg0_before = CT_CFG.GPCFG0;  /* Read current GPCFG0 value */
    uint32_t gpcfg1_before = CT_CFG.GPCFG1;  /* Read current GPCFG1 value */

    CT_CFG.GPCFG0 = 0x0000;  /* PRU0: Direct connect mode, 16-bit parallel capture disabled */
    CT_CFG.GPCFG1 = 0x0000;  /* PRU1: Direct connect mode, 16-bit parallel capture disabled */

    uint32_t gpcfg0_after = CT_CFG.GPCFG0;   /* Verify GPCFG0 */
    uint32_t gpcfg1_after = CT_CFG.GPCFG1;   /* Verify GPCFG1 */

    /* Initialize control block */
    ctrl->write_index = 0;
    ctrl->read_index = 0;
    ctrl->event_count = 0;
    ctrl->error_count = 0;
    ctrl->overrun_count = (gpcfg0_before << 16) | gpcfg1_before;   /* DEBUG: Store initial values */
    ctrl->status = (gpcfg0_after << 16) | gpcfg1_after;            /* DEBUG: Store final values */

    /* Initialize debug structure - clear any stale data */
    debug->valid = 0;
    debug->error_code = 0;
    for (i = 0; i < IR_PACKET_LENGTH; i++) {
        debug->bits[i] = 0;
        debug->durations[i] = 0;
    }

    /* Run the IR detection loop (never returns) */
    run_ir_detection_loop();
}
