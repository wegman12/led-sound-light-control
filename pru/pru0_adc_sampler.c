/*
 * PRU0 ADC Sampler - 48 kHz sampling from AIN1
 *
 * This PRU firmware samples the BeagleBone Black's ADC at 48 kHz
 * and writes samples to PRU shared memory for the host to read.
 */

/* Type definitions - no includes needed */
typedef unsigned int uint32_t;
typedef unsigned short uint16_t;

/* PRU-ICSS CFG Registers - Enable OCP master port */
#define PRUSS_CFG_BASE  0x00026000
#define PRUSS_CFG_SYSCFG (*(volatile uint32_t *)(PRUSS_CFG_BASE + 0x04))
#define SYSCFG_STANDBY_INIT (1 << 4)  /* Enable OCP master port */
#define SYSCFG_IDLE_MODE_NO (1 << 2)  /* No standby mode */

/* Memory addresses */
#define ADC_TSC         0x44E0D000
#define CM_WKUP         0x44E00400

/* ADC registers */
#define ADC_CTRL        (*(volatile uint32_t *)(ADC_TSC + 0x40))
#define ADC_STEPCONFIG1 (*(volatile uint32_t *)(ADC_TSC + 0x64))
#define ADC_STEPDELAY1  (*(volatile uint32_t *)(ADC_TSC + 0x68))
#define ADC_STEPENABLE  (*(volatile uint32_t *)(ADC_TSC + 0x54))
#define ADC_FIFO0DATA   (*(volatile uint32_t *)(ADC_TSC + 0x100))
#define ADC_FIFO0COUNT  (*(volatile uint32_t *)(ADC_TSC + 0xE4))
#define ADC_IRQSTATUS   (*(volatile uint32_t *)(ADC_TSC + 0x28))
#define ADC_IRQENABLE   (*(volatile uint32_t *)(ADC_TSC + 0x2C))

/* Shared memory - 12KB starting at 0x00010000 in PRU memory map */
#define PRU_SHARED_MEM  0x00010000
#define BUFFER_SIZE     4096  /* 4096 samples = 8192 bytes */
#define CONTROL_OFFSET  0x2000

/* Control structure at offset 0x2000 */
struct control_block {
    volatile uint32_t write_index;
    volatile uint32_t read_index;
    volatile uint32_t sample_count;
    volatile uint32_t overrun_count;
};

#define CONTROL ((struct control_block *)(PRU_SHARED_MEM + CONTROL_OFFSET))
#define SAMPLE_BUFFER ((volatile uint16_t *)PRU_SHARED_MEM)

/* Timing for 48 kHz sampling */
#define CYCLES_PER_SAMPLE (200000000 / 48000)  /* PRU runs at 200 MHz */

/* Function prototypes */
void init_adc(void);
uint16_t read_adc_sample(void);
void write_sample(uint16_t sample);
void delay_cycles(uint32_t cycles);

void main(void) {
    uint16_t sample;

    /* CRITICAL: Enable PRU OCP master port for peripheral access */
    PRUSS_CFG_SYSCFG = SYSCFG_STANDBY_INIT | SYSCFG_IDLE_MODE_NO;

    /* Initialize control block */
    CONTROL->write_index = 0;
    CONTROL->read_index = 0;
    CONTROL->sample_count = 0;
    CONTROL->overrun_count = 0;

    /* Initialize ADC */
    init_adc();

    /* Main sampling loop */
    while (1) {
        /* Read ADC sample */
        sample = read_adc_sample();

        /* Write to ring buffer */
        write_sample(sample);

        /* Delay for 48 kHz sampling (CYCLES_PER_SAMPLE at 200 MHz) */
        delay_cycles(CYCLES_PER_SAMPLE);
    }
}

void init_adc(void) {
    /* Disable ADC */
    ADC_CTRL = 0x00000000;

    /* Configure Step 1 for AIN1 (channel 1) */
    /* Mode: One-shot, Channel: AIN1, FIFO: 0 */
    ADC_STEPCONFIG1 = 0x00000001 |  /* Channel 1 (AIN1) */
                      (0x00 << 19) |  /* FIFO0 */
                      (0x00 << 26);   /* One-shot mode */

    /* Set step delay to 0 (fastest sampling) */
    ADC_STEPDELAY1 = 0x00000000;

    /* Enable ADC and step 1 */
    ADC_CTRL = 0x00000007;  /* Enable ADC, no idle, tag channel ID */
    ADC_STEPENABLE = 0x00000002;  /* Enable step 1 */
}

uint16_t read_adc_sample(void) {
    uint32_t data;

    /* Trigger step 1 by enabling it */
    ADC_STEPENABLE = 0x00000002;

    /* Wait for FIFO to have data (timeout after ~100 cycles) */
    uint32_t timeout = 100;
    while ((ADC_FIFO0COUNT == 0) && (timeout > 0)) {
        timeout--;
    }

    /* Read from FIFO */
    if (ADC_FIFO0COUNT > 0) {
        data = ADC_FIFO0DATA;
        /* Extract 12-bit ADC value (bits 0-11) */
        return (uint16_t)(data & 0x0FFF);
    }

    return 0;  /* Return 0 if timeout */
}

void write_sample(uint16_t sample) {
    uint32_t next_write = (CONTROL->write_index + 1) % BUFFER_SIZE;

    /* Check for buffer overrun */
    if (next_write == CONTROL->read_index) {
        CONTROL->overrun_count++;
        return;  /* Skip this sample */
    }

    /* Write sample to buffer */
    SAMPLE_BUFFER[CONTROL->write_index] = sample;

    /* Update write index and sample count */
    CONTROL->write_index = next_write;
    CONTROL->sample_count++;
}

void delay_cycles(uint32_t cycles) {
    volatile uint32_t i;
    for (i = 0; i < cycles; i++) {
        /* Empty loop for delay */
    }
}
