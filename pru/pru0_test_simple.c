/*
 * Simple PRU Test - Just write incrementing values to shared memory
 * This tests that PRU -> ARM communication is working
 */

/* Type definitions - no includes needed */
typedef unsigned int uint32_t;
typedef unsigned short uint16_t;

/* Shared memory - 12KB starting at 0x00010000 in PRU memory map */
#define PRU_SHARED_MEM  0x00010000
#define CONTROL_OFFSET  0x2000

/* Control structure */
struct control_block {
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t sample_count;
    volatile uint32_t overrun_count;
};

#define CONTROL ((struct control_block *)(PRU_SHARED_MEM + CONTROL_OFFSET))
#define SAMPLE_BUFFER ((volatile uint16_t *)PRU_SHARED_MEM)
#define BUFFER_SIZE 4096

/* Simple delay function */
void delay(uint32_t cycles) {
    volatile uint32_t i;
    for (i = 0; i < cycles; i++) {
        /* Just loop - compiler won't optimize away volatile */
    }
}

int main(void) {
    uint16_t counter = 0;
    uint32_t loop_count = 0;

    /* Initialize control block */
    CONTROL->write_index = 0;
    CONTROL->read_index = 0;
    CONTROL->sample_count = 0;
    CONTROL->overrun_count = 0;

    /* Main loop - write incrementing values */
    while (1) {
        uint32_t next_write = (CONTROL->write_index + 1) % BUFFER_SIZE;

        /* Check for buffer overrun */
        if (next_write == CONTROL->read_index) {
            CONTROL->overrun_count++;
        } else {
            /* Write incrementing value */
            SAMPLE_BUFFER[CONTROL->write_index] = counter;
            CONTROL->write_index = next_write;
            CONTROL->sample_count++;
            counter++;
        }

        /* Delay to simulate ~1000 Hz (200,000 cycles = 1ms at 200 MHz) */
        delay(200000);

        loop_count++;
    }

    return 0;
}
