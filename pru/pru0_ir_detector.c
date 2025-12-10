/*
 * PRU0 IR Remote Detector Firmware
 *
 * Detects IR remote signals on GPIO 20, decodes 34-bit NEC-like protocol,
 * matches against 44 known button codes, and writes events to shared memory
 * ring buffer for consumption by Go application.
 *
 * IMPORTANT: GPIO 20 is ACTIVE-LOW (pulled high by default)
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
#define SYSCFG_ADDR     ((volatile uint32_t *)(PRUSS_CFG_BASE + SYSCFG_OFFSET))

/* SYSCFG register bits */
#define SYSCFG_STANDBY_INIT  (1 << 4)
#define SYSCFG_IDLE_MODE     (0 << 2)  /* No idle mode */
#define SYSCFG_SUB_MWAIT     (1 << 5)  /* Enable OCP master ports */

/* Memory Layout */
#define PRU_SHARED_MEM      0x00010000
#define BUTTON_RING_BUFFER  0x00000000  /* Offset 0: Ring buffer */
#define CONTROL_BLOCK       0x00001000  /* Offset 4KB: Control structure */
#define RING_BUFFER_SIZE    256         /* 256 events */
#define TIMING_DATA_OFFSET  0x00002000  /* Offset 8KB: Raw timing data */
#define MAX_TIMING_SAMPLES  512         /* Maximum transitions to capture */
#define GPIO_RAW_DATA_OFFSET 0x00002000 /* Offset 8KB: Raw GPIO readings */
#define MAX_GPIO_SAMPLES    8192        /* 8K samples of raw GPIO data */

/* GPIO Configuration */
#define GPIO0_BASE          0x44E07000
#define GPIO_DATAIN_OFFSET  0x138
#define GPIO_PIN_20         (1 << 20)

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
    volatile uint8_t samples[MAX_GPIO_SAMPLES];  /* Each byte is a GPIO reading (0 or 1) */
};

/* GPIO Access */
#define GPIO_DATAIN (*(volatile uint32_t *)(GPIO0_BASE + GPIO_DATAIN_OFFSET))

static inline uint32_t read_gpio20(void) {
    return (GPIO_DATAIN & GPIO_PIN_20) ? 1 : 0;
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
/* Includes 1μs delay between reads to match Go implementation */
static uint32_t wait_for_state(uint32_t target_state, uint32_t max_cycles, int *timed_out) {
    uint32_t start = get_cycles();
    uint32_t elapsed;

    while (read_gpio20() != target_state) {
        elapsed = get_cycles() - start;
        if (elapsed > max_cycles) {
            *timed_out = 1;
            return elapsed;
        }
        /* 1μs delay between reads to match Go's polling behavior (200 cycles @ 200MHz) */
        __delay_cycles(20);
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
    uint8_t bits[IR_PACKET_LENGTH];
    int i;
    int timed_out;
    uint32_t space_cycles;
    uint32_t packet_start = get_cycles();

    /* GPIO is already LOW when this function is called (detected in main loop) */
    /* Wait for it to go HIGH to complete the initial LOW pulse */
    wait_for_state(1, MAX_BIT_WAIT, &timed_out);
    if (timed_out) {
        ctrl->event_count = 0xFFFF;
        ctrl->overrun_count = 0;
        return -1;
    }

    /* Read 33 pulses */
    for (i = 0; i < IR_PACKET_LENGTH; i++) {
        /* Check packet timeout */
        if ((get_cycles() - packet_start) > MAX_PACKET_DURATION) {
            ctrl->event_count = 0xFFFD;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Wait for GPIO LOW (end of HIGH/space period) - measure HIGH duration */
        space_cycles = wait_for_state(0, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {

            ctrl->event_count = 0xFFFC;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Wait for GPIO HIGH (end of LOW/mark period) - don't measure this */
        wait_for_state(1, MAX_BIT_WAIT, &timed_out);
        if (timed_out) {
            ctrl->event_count = 0xFFFB;
            ctrl->overrun_count = i;
            return -1;
        }

        /* Decode based on HIGH duration (space) */
        bits[i] = decode_bit(space_cycles);
    }

    /* Validate protocol structure */
    if (bits[0] != 1) {
        ctrl->event_count = 0xFFDF;
        return -1;
    }                          /* START bit */
    if (!all_zeros(&bits[1], 8)) {
        ctrl->event_count = 0xFFDE;
        return -1;
    }               /* HEADER (bits 1-8) */
    if (!all_ones(&bits[9], 8)) {

        ctrl->event_count = 0xFFDD;
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
 * ORIGINAL DETECTION LOGIC - COMMENTED OUT FOR DEBUG
 * This function contains the original IR packet detection logic.
 * Restore this in main() after debugging is complete.
 */
#if 0
static void run_ir_detection_loop(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    int button_code;
    int timed_out;

    /* Main IR detection loop */
    while (1) {
        /* Wait for GPIO LOW (signal present) - matches Go's trigger condition */
        if (read_gpio20() == 0) {
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

        /* 1μs delay between main loop iterations to match Go's polling behavior */
        __delay_cycles(200);
    }
}
#endif

/* Main function - DEBUG MODE: Raw GPIO Reading Test */
void main(void) {
    struct control_block *ctrl = (struct control_block *)(PRU_SHARED_MEM + CONTROL_BLOCK);
    struct gpio_raw_data *gpio_data = (struct gpio_raw_data *)(PRU_SHARED_MEM + GPIO_RAW_DATA_OFFSET);

    /* Enable OCP master port - allows PRU to access peripheral registers like GPIO */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize control block */
    ctrl->write_index = 0;
    ctrl->read_index = 0;
    ctrl->event_count = 0;
    ctrl->error_count = 0;
    ctrl->overrun_count = 0;
    ctrl->status = 1;  /* Running */

    /* Initialize GPIO data structure */
    gpio_data->sample_count = 0;
    gpio_data->complete = 0;

    /* Wait for initial LOW (0) to start capturing */
    while (read_gpio20() == 1) {
        /* Busy wait - no delay */
    }

    /* Raw GPIO reading loop - sample as fast as possible */
    uint32_t i;
    for (i = 0; i < MAX_GPIO_SAMPLES; i++) {
        gpio_data->samples[i] = (uint8_t)read_gpio20();
    }

    /* Mark capture as complete */
    gpio_data->sample_count = MAX_GPIO_SAMPLES;
    gpio_data->complete = 1;

    /* Signal completion in control block */
    ctrl->event_count = MAX_GPIO_SAMPLES;
    ctrl->status = 2;  /* Capture complete */

    /* Halt - wait forever */
    while (1) {
        __delay_cycles(1000);
    }
}
