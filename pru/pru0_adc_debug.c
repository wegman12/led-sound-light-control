/*
 * PRU ADC Debug - Check if ADC is accessible and working
 */

/* Type definitions */
typedef unsigned int uint32_t;
typedef unsigned short uint16_t;

/* PRU-ICSS CFG Registers - Enable OCP master port */
#define PRUSS_CFG_BASE  0x00026000
#define PRUSS_CFG_SYSCFG (*(volatile uint32_t *)(PRUSS_CFG_BASE + 0x04))
#define SYSCFG_STANDBY_INIT (1 << 4)  /* Enable OCP master port */
#define SYSCFG_IDLE_MODE_NO (1 << 2)  /* No standby mode */

/* ADC register addresses */
#define ADC_TSC         0x44E0D000
#define ADC_CTRL        (*(volatile uint32_t *)(ADC_TSC + 0x40))
#define ADC_STEPCONFIG1 (*(volatile uint32_t *)(ADC_TSC + 0x64))
#define ADC_STEPDELAY1  (*(volatile uint32_t *)(ADC_TSC + 0x68))
#define ADC_STEPENABLE  (*(volatile uint32_t *)(ADC_TSC + 0x54))
#define ADC_FIFO0DATA   (*(volatile uint32_t *)(ADC_TSC + 0x100))
#define ADC_FIFO0COUNT  (*(volatile uint32_t *)(ADC_TSC + 0xE4))
#define ADC_IRQSTATUS   (*(volatile uint32_t *)(ADC_TSC + 0x28))

/* Shared memory */
#define PRU_SHARED_MEM  0x00010000
#define CONTROL_OFFSET  0x2000

/* Debug info structure */
struct debug_info {
    volatile uint32_t adc_ctrl;
    volatile uint32_t adc_stepenable;
    volatile uint32_t adc_fifo0count;
    volatile uint32_t adc_fifo0data;
    volatile uint32_t attempts;
    volatile uint32_t successful_reads;
    volatile uint32_t timeouts;
    volatile uint32_t last_sample;
};

#define DEBUG ((struct debug_info *)(PRU_SHARED_MEM + CONTROL_OFFSET))
#define SAMPLE_BUFFER ((volatile uint16_t *)PRU_SHARED_MEM)

void delay(uint32_t cycles) {
    volatile uint32_t i;
    for (i = 0; i < cycles; i++) {
        /* Empty loop */
    }
}

int main(void) {
    uint32_t loop_count = 0;
    uint16_t sample_index = 0;

    /* CRITICAL: Enable PRU OCP master port for peripheral access */
    PRUSS_CFG_SYSCFG = SYSCFG_STANDBY_INIT | SYSCFG_IDLE_MODE_NO;

    /* Small delay after enabling OCP */
    delay(10000);

    /* Initialize debug info */
    DEBUG->adc_ctrl = 0;
    DEBUG->adc_stepenable = 0;
    DEBUG->adc_fifo0count = 0;
    DEBUG->adc_fifo0data = 0;
    DEBUG->attempts = 0;
    DEBUG->successful_reads = 0;
    DEBUG->timeouts = 0;
    DEBUG->last_sample = 0;

    /* Try to initialize ADC */
    ADC_CTRL = 0x00000000;  /* Disable first */
    delay(1000);

    /* Configure Step 1 for AIN1 */
    ADC_STEPCONFIG1 = 0x00000001 | (0x00 << 19);
    ADC_STEPDELAY1 = 0x00000000;

    /* Enable ADC */
    ADC_CTRL = 0x00000007;
    ADC_STEPENABLE = 0x00000002;  /* Enable step 1 */

    delay(10000);

    /* Main loop - try to read ADC */
    while (1) {
        DEBUG->attempts++;

        /* Read current register values */
        DEBUG->adc_ctrl = ADC_CTRL;
        DEBUG->adc_stepenable = ADC_STEPENABLE;
        DEBUG->adc_fifo0count = ADC_FIFO0COUNT;

        /* Trigger ADC */
        ADC_STEPENABLE = 0x00000002;

        /* Wait for data with timeout */
        uint32_t timeout = 10000;
        while ((ADC_FIFO0COUNT == 0) && (timeout > 0)) {
            timeout--;
        }

        if (ADC_FIFO0COUNT > 0) {
            /* Success! */
            uint32_t data = ADC_FIFO0DATA;
            uint16_t sample = (uint16_t)(data & 0x0FFF);

            DEBUG->successful_reads++;
            DEBUG->last_sample = sample;
            DEBUG->adc_fifo0data = data;

            /* Write to sample buffer */
            SAMPLE_BUFFER[sample_index] = sample;
            sample_index = (sample_index + 1) % 100;  /* Keep last 100 samples */
        } else {
            /* Timeout */
            DEBUG->timeouts++;
        }

        /* Delay between reads (~1000 Hz) */
        delay(200000);

        loop_count++;
    }

    return 0;
}
