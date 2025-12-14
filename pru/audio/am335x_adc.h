/*
 * AM335x TSC_ADC_SS Register Definitions for PRU
 *
 * BeagleBone Black ADC Hardware Access
 * Base Address: 0x44E0D000
 *
 * Reference: AM335x TRM (SPRUH73Q) Section 12
 * Author: Generated with Claude Code
 */

#ifndef AM335X_ADC_H
#define AM335X_ADC_H

#include <stdint.h>

/* ADC Base Address (physical address accessible via OCP) */
#define ADC_BASE_ADDR       0x44E0D000

/* Register Offsets */
#define ADC_REVISION        0x000  /* Revision ID */
#define ADC_SYSCONFIG       0x010  /* System configuration */
#define ADC_IRQSTATUS_RAW   0x024  /* IRQ status raw */
#define ADC_IRQSTATUS       0x028  /* IRQ status */
#define ADC_IRQENABLE_SET   0x02C  /* IRQ enable set */
#define ADC_IRQENABLE_CLR   0x030  /* IRQ enable clear */
#define ADC_IRQWAKEUP       0x034  /* IRQ wakeup enable */
#define ADC_DMAENABLE_SET   0x038  /* DMA enable set */
#define ADC_DMAENABLE_CLR   0x03C  /* DMA enable clear */
#define ADC_CTRL            0x040  /* Control register */
#define ADC_ADCSTAT         0x044  /* ADC status */
#define ADC_ADCRANGE        0x048  /* ADC range */
#define ADC_ADC_CLKDIV      0x04C  /* ADC clock divider */
#define ADC_ADC_MISC        0x050  /* ADC misc control */
#define ADC_STEPENABLE      0x054  /* Step enable register */
#define ADC_IDLECONFIG      0x058  /* Idle configuration */
#define ADC_TS_CHARGE_STEPCONFIG  0x05C  /* TS charge step config */
#define ADC_TS_CHARGE_DELAY 0x060  /* TS charge delay */

/* Step Configuration Registers (16 steps: 1-16) */
#define ADC_STEPCONFIG1     0x064
#define ADC_STEPDELAY1      0x068
#define ADC_STEPCONFIG2     0x06C
#define ADC_STEPDELAY2      0x070
/* ... continues for steps 3-16 ... */
#define ADC_STEPCONFIG(n)   (0x064 + ((n-1) * 8))  /* n = 1..16 */
#define ADC_STEPDELAY(n)    (0x068 + ((n-1) * 8))  /* n = 1..16 */

/* FIFO Registers (2 FIFOs: 0-1) */
#define ADC_FIFO0COUNT      0x0E4  /* FIFO 0 sample count */
#define ADC_FIFO0THRESHOLD  0x0E8  /* FIFO 0 threshold */
#define ADC_FIFO0DATA       0x100  /* FIFO 0 data (read pops) */
#define ADC_FIFO1COUNT      0x0F0  /* FIFO 1 sample count */
#define ADC_FIFO1THRESHOLD  0x0F4  /* FIFO 1 threshold */
#define ADC_FIFO1DATA       0x200  /* FIFO 1 data (read pops) */

/* Control Register (ADC_CTRL) Bits */
#define ADC_CTRL_ENABLE         (1 << 0)   /* Module enable */
#define ADC_CTRL_STEPCONFIG_WP  (1 << 2)   /* Step config write protect (0=writable) */
#define ADC_CTRL_STEPIDLE       (1 << 1)   /* Step idle mode */
#define ADC_CTRL_POWERDOWN      (1 << 4)   /* Power down ADC */

/* Step Configuration Register Bits */
#define ADC_STEPCONFIG_MODE_SW_ONESHOT  (0 << 0)  /* Software enabled, one-shot */
#define ADC_STEPCONFIG_MODE_SW_CONT     (1 << 0)  /* Software enabled, continuous */
#define ADC_STEPCONFIG_MODE_HW_SYNC     (2 << 0)  /* Hardware sync enabled */

#define ADC_STEPCONFIG_AVG_SHIFT        2
#define ADC_STEPCONFIG_AVG_1            (0 << ADC_STEPCONFIG_AVG_SHIFT)  /* No averaging */
#define ADC_STEPCONFIG_AVG_2            (1 << ADC_STEPCONFIG_AVG_SHIFT)  /* 2 samples */
#define ADC_STEPCONFIG_AVG_4            (2 << ADC_STEPCONFIG_AVG_SHIFT)  /* 4 samples */
#define ADC_STEPCONFIG_AVG_8            (3 << ADC_STEPCONFIG_AVG_SHIFT)  /* 8 samples */
#define ADC_STEPCONFIG_AVG_16           (4 << ADC_STEPCONFIG_AVG_SHIFT)  /* 16 samples */

#define ADC_STEPCONFIG_SEL_INP_SHIFT    19
#define ADC_STEPCONFIG_SEL_INM_SHIFT    23
#define ADC_STEPCONFIG_FIFO_SELECT      (1 << 26)  /* 0=FIFO0, 1=FIFO1 */
#define ADC_STEPCONFIG_DIFF_CNTRL       (1 << 25)  /* 0=single-ended, 1=differential */

/* ADC Input Channel Selection (0-7 for AIN0-AIN7) */
#define ADC_CHANNEL_AIN0    0
#define ADC_CHANNEL_AIN1    1
#define ADC_CHANNEL_AIN2    2
#define ADC_CHANNEL_AIN3    3
#define ADC_CHANNEL_AIN4    4
#define ADC_CHANNEL_AIN5    5
#define ADC_CHANNEL_AIN6    6
#define ADC_CHANNEL_AIN7    7

/* FIFO Data Register Format */
#define ADC_FIFO_DATA_MASK      0x0FFF  /* 12-bit ADC value */
#define ADC_FIFO_CHANNEL_SHIFT  16      /* Channel ID in upper bits */
#define ADC_FIFO_CHANNEL_MASK   0x000F0000

/* Step Delay Register */
#define ADC_STEPDELAY_SAMPLE_SHIFT  0   /* Sample delay (ADC clock cycles) */
#define ADC_STEPDELAY_OPEN_SHIFT    24  /* Open delay (ADC clock cycles) */

/* Useful Macros */
#define ADC_REG(offset)     (*(volatile uint32_t *)(ADC_BASE_ADDR + (offset)))

/* ADC Configuration Helpers */
static inline void adc_enable_step_config_write(void) {
    /* Clear write protect bit to allow step configuration */
    ADC_REG(ADC_CTRL) &= ~ADC_CTRL_STEPCONFIG_WP;
}

static inline void adc_disable_step_config_write(void) {
    /* Set write protect bit to lock step configuration */
    ADC_REG(ADC_CTRL) |= ADC_CTRL_STEPCONFIG_WP;
}

static inline void adc_enable(void) {
    ADC_REG(ADC_CTRL) |= ADC_CTRL_ENABLE;
}

static inline void adc_disable(void) {
    ADC_REG(ADC_CTRL) &= ~ADC_CTRL_ENABLE;
}

static inline void adc_enable_step(uint32_t step_num) {
    /* step_num: 1-16 */
    ADC_REG(ADC_STEPENABLE) |= (1 << step_num);
}

static inline void adc_disable_step(uint32_t step_num) {
    /* step_num: 1-16 */
    ADC_REG(ADC_STEPENABLE) &= ~(1 << step_num);
}

static inline uint32_t adc_fifo0_count(void) {
    return ADC_REG(ADC_FIFO0COUNT) & 0x7F;
}

static inline uint32_t adc_read_fifo0(void) {
    return ADC_REG(ADC_FIFO0DATA);
}

static inline uint32_t adc_fifo_get_value(uint32_t fifo_data) {
    return fifo_data & ADC_FIFO_DATA_MASK;
}

static inline uint32_t adc_fifo_get_channel(uint32_t fifo_data) {
    return (fifo_data & ADC_FIFO_CHANNEL_MASK) >> ADC_FIFO_CHANNEL_SHIFT;
}

#endif /* AM335X_ADC_H */
