/*
 * PRU1 Raw I2S Capture Firmware
 *
 * Captures raw 24-bit samples from I2S MEMS microphone without any processing.
 * Used for offline analysis and audio parameter tuning.
 *
 * This firmware reuses the McASP initialization from pru1_audio_i2s.c but
 * stores samples in a circular buffer instead of doing FFT processing.
 *
 * Hardware Required:
 *   - INMP441 or SPH0645 I2S MEMS Microphone
 *   - Device tree overlay: BB-I2S-PRU-00A0.dtbo
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#include <stdint.h>
#include <pru_ctrl.h>
#include <pru_cfg.h>
#include "resource_table_empty.h"
#include "pru_shared_memory.h"
#include "am335x_mcasp.h"

/*
 * =============================================================================
 * Memory Layout Configuration
 * =============================================================================
 */

/* Shared memory pointers */
#define PRU_SHARED_MEM      PRU_SHARED_MEM_BASE

/* Control block at standard I2S location */
#define RAW_CONTROL_OFFSET  PRU1_AUDIO_CONTROL_OFFSET  /* 0x0900 */

/* Sample buffer uses both buffer A and B space for larger circular buffer */
#define RAW_BUFFER_OFFSET   PRU1_SAMPLE_BUFFER_A_OFFSET  /* 0x0A00 */

/* Buffer size: use both A and B regions = 8KB = 2048 32-bit samples */
#define RAW_BUFFER_SIZE     2048  /* ~64ms at 32 kHz */

/*
 * =============================================================================
 * PRCM (Power/Reset/Clock Management) Registers
 * =============================================================================
 */
#define CM_PER_BASE             0x44E00000
#define CM_PER_MCASP0_CLKCTRL   (*((volatile uint32_t *)(CM_PER_BASE + 0x34)))
#define CM_PER_MODULEMODE_ENABLE    0x02
#define CM_PER_IDLEST_FUNCTIONAL    0x00

/*
 * =============================================================================
 * I2S/McASP Configuration
 * =============================================================================
 */
#define ACLK_DIV            7   /* Divide by 8: 24 MHz / 8 = 3 MHz */

/*
 * =============================================================================
 * Status Codes
 * =============================================================================
 */
#define STATUS_RAW_I2S      0x52493253  /* "RI2S" - Raw I2S capture running */
#define STATUS_MCASP_INIT   0x4D435350  /* "MCSP" - McASP initialized */
#define STATUS_CAPTURING    0x43415054  /* "CAPT" - Capture active */

/*
 * =============================================================================
 * Control Block Structure (must match Go utility)
 * =============================================================================
 */
struct raw_i2s_control_block {
    /* === Configuration Section (written by host, read by PRU) === */
    volatile uint32_t enable_capture;       /* 1 = capture enabled, 0 = paused */

    /* === Status Section (written by PRU, read by host) === */
    volatile uint32_t status;               /* PRU running status code */
    volatile uint32_t total_samples;        /* Total samples collected */
    volatile uint32_t buffer_write_index;   /* Current write position in circular buffer */
    volatile uint32_t buffer_wrap_count;    /* Number of times buffer has wrapped */
    volatile uint32_t mcasp_errors;         /* McASP timeout/error count */
    volatile uint32_t last_sample;          /* Most recent sample value */
    volatile uint32_t min_sample;           /* Minimum sample value (signed) */
    volatile uint32_t max_sample;           /* Maximum sample value (signed) */

    /* Debug: McASP register values for diagnostics */
    volatile uint32_t debug_gblctl;         /* GBLCTL register after init */
    volatile uint32_t debug_rstat;          /* RSTAT register (updated in loop) */
    volatile uint32_t debug_aclkrctl;       /* ACLKRCTL register */
    volatile uint32_t debug_ahclkrctl;      /* AHCLKRCTL register */
    volatile uint32_t debug_loop_count;     /* Loop iterations waiting for data */

    /* Padding to 256 bytes */
    volatile uint32_t reserved[50];
};

/*
 * =============================================================================
 * McASP Initialization (same as pru1_audio_i2s.c)
 * =============================================================================
 */
static void init_mcasp_i2s(struct raw_i2s_control_block *ctrl) {
    /* Enable McASP0 clock via PRCM */
    CM_PER_MCASP0_CLKCTRL = CM_PER_MODULEMODE_ENABLE;

    /* Wait for clock to stabilize */
    while ((CM_PER_MCASP0_CLKCTRL & 0x00030000) != CM_PER_IDLEST_FUNCTIONAL) {
        __delay_cycles(100);
    }

    /* Reset McASP */
    MCASP_CFG_WRITE(MCASP_GBLCTL, 0x00000000);

    /* Configure pin directions */
    MCASP_CFG_WRITE(MCASP_PDIR,
        MCASP_PDIR_AFSR |     /* AFSR = output */
        MCASP_PDIR_ACLKR |    /* ACLKR = output */
        MCASP_PDIR_AFSX |     /* AFSX = output */
        MCASP_PDIR_ACLKX      /* ACLKX = output */
    );

    /* Configure high-frequency clock (internal mode) */
    MCASP_CFG_WRITE(MCASP_AHCLKRCTL, MCASP_AHCLKRCTL_HCLKRM);

    /* Configure receive bit clock */
    MCASP_CFG_WRITE(MCASP_ACLKRCTL,
        (ACLK_DIV << MCASP_ACLKRCTL_CLKRDIV_SHIFT) |
        MCASP_ACLKRCTL_CLKRM |
        MCASP_ACLKRCTL_CLKRP
    );

    /* Configure transmit clock (for clock output) */
    MCASP_CFG_WRITE(MCASP_AHCLKXCTL, MCASP_AHCLKRCTL_HCLKRM);
    MCASP_CFG_WRITE(MCASP_ACLKXCTL,
        (ACLK_DIV << MCASP_ACLKRCTL_CLKRDIV_SHIFT) |
        MCASP_ACLKRCTL_CLKRM |
        MCASP_ACLKRCTL_CLKRP
    );

    /* Configure receive frame sync */
    MCASP_CFG_WRITE(MCASP_AFSRCTL,
        (2 << MCASP_AFSRCTL_RMOD_SHIFT) |
        MCASP_AFSRCTL_FRWID |
        MCASP_AFSRCTL_FSRM |
        MCASP_AFSRCTL_FSRP
    );

    /* Configure transmit frame sync */
    MCASP_CFG_WRITE(MCASP_AFSXCTL,
        (2 << MCASP_AFSRCTL_RMOD_SHIFT) |
        MCASP_AFSRCTL_FRWID |
        MCASP_AFSRCTL_FSRM |
        MCASP_AFSRCTL_FSRP
    );

    /* Configure receive format */
    MCASP_CFG_WRITE(MCASP_RFMT,
        (MCASP_SLOT_SIZE_32 << MCASP_RFMT_RSSZ_SHIFT) |
        (MCASP_DATDLY_1BIT << MCASP_RFMT_RDATDLY_SHIFT) |
        MCASP_RFMT_RRVRS
    );

    /* Configure receive mask and TDM */
    MCASP_CFG_WRITE(MCASP_RMASK, 0xFFFFFFFF);
    MCASP_CFG_WRITE(MCASP_RTDM, 0x00000001);  /* Only slot 0 (left) */

    /* Configure transmit format, mask, TDM */
    MCASP_CFG_WRITE(MCASP_XFMT,
        (MCASP_SLOT_SIZE_32 << MCASP_RFMT_RSSZ_SHIFT) |
        (MCASP_DATDLY_1BIT << MCASP_RFMT_RDATDLY_SHIFT)
    );
    MCASP_CFG_WRITE(MCASP_XMASK, 0xFFFFFFFF);
    MCASP_CFG_WRITE(MCASP_XTDM, 0x00000000);

    /* Configure serializer 0 for receive */
    MCASP_CFG_WRITE(MCASP_SRCTL0, (MCASP_SRMOD_RX << MCASP_SRCTL_SRMOD_SHIFT));

    /* Enable Audio FIFO for receive */
    MCASP_CFG_WRITE(MCASP_RFIFOCTL,
        (1 << MCASP_FIFOCTL_NUMDMA_SHIFT) |
        (1 << MCASP_FIFOCTL_NUMEVT_SHIFT) |
        MCASP_FIFOCTL_ENA
    );

    /* Enable McASP clocks and state machines */
    uint32_t gblctl;

    /* Enable high-frequency clock dividers */
    MCASP_CFG_WRITE(MCASP_GBLCTL, MCASP_GBLCTL_RHCLKRST | MCASP_GBLCTL_XHCLKRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RHCLKRST | MCASP_GBLCTL_XHCLKRST))
           != (MCASP_GBLCTL_RHCLKRST | MCASP_GBLCTL_XHCLKRST)) {
        __delay_cycles(10);
    }

    /* Enable clock dividers */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST))
           != (MCASP_GBLCTL_RCLKRST | MCASP_GBLCTL_XCLKRST)) {
        __delay_cycles(10);
    }

    /* Re-enable AHCLK after clock reset */
    MCASP_CFG_WRITE(MCASP_AHCLKRCTL, MCASP_AHCLKRCTL_HCLKRM);
    MCASP_CFG_WRITE(MCASP_AHCLKXCTL, MCASP_AHCLKRCTL_HCLKRM);

    /* Enable RX serializer */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RSRCLR);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & MCASP_GBLCTL_RSRCLR) != MCASP_GBLCTL_RSRCLR) {
        __delay_cycles(10);
    }

    /* Enable RX state machine */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RSMRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & MCASP_GBLCTL_RSMRST) != MCASP_GBLCTL_RSMRST) {
        __delay_cycles(10);
    }

    /* Enable frame sync generators */
    gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    MCASP_CFG_WRITE(MCASP_GBLCTL, gblctl | MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST);
    while ((MCASP_CFG_READ(MCASP_GBLCTL) & (MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST))
           != (MCASP_GBLCTL_RFRST | MCASP_GBLCTL_XFRST)) {
        __delay_cycles(10);
    }

    /* Clear any pending status */
    MCASP_CFG_WRITE(MCASP_RSTAT, 0xFFFFFFFF);

    /* Capture register values for debugging */
    ctrl->debug_gblctl = MCASP_CFG_READ(MCASP_GBLCTL);
    ctrl->debug_rstat = MCASP_CFG_READ(MCASP_RSTAT);
    ctrl->debug_aclkrctl = MCASP_CFG_READ(MCASP_ACLKRCTL);
    ctrl->debug_ahclkrctl = MCASP_CFG_READ(MCASP_AHCLKRCTL);
    ctrl->debug_loop_count = 0;

    ctrl->status = STATUS_MCASP_INIT;
}

/*
 * Read sample from McASP FIFO data port
 * Returns 24-bit signed sample in 32-bit container
 */
static inline int32_t read_mcasp_sample(struct raw_i2s_control_block *ctrl) {
    uint32_t loops = 0;
    uint32_t fifo_level;

    /* Wait for FIFO to have data */
    while (1) {
        fifo_level = MCASP_CFG_READ(MCASP_RFIFOSTS) & MCASP_FIFOSTS_LEVEL_MASK;
        if (fifo_level > 0) {
            break;
        }

        loops++;
        if ((loops & 0xFFFFF) == 0) {
            ctrl->debug_loop_count = loops;
            ctrl->debug_rstat = MCASP_CFG_READ(MCASP_RSTAT);
        }
        if (loops > 100000000) {
            ctrl->mcasp_errors++;
            loops = 0;
        }
    }

    /* Read from McASP data port */
    int32_t raw = (int32_t)(*((volatile uint32_t *)MCASP0_DATA_BASE));

    /* I2S data is MSB-justified, shift right by 8 for 24-bit value */
    return raw >> 8;
}

/*
 * =============================================================================
 * Main Function
 * =============================================================================
 */

void main(void) {
    struct raw_i2s_control_block *ctrl =
        (struct raw_i2s_control_block *)(PRU_SHARED_MEM + RAW_CONTROL_OFFSET);

    /* Sample buffer in shared memory */
    volatile int32_t *sample_buffer = (volatile int32_t *)(PRU_SHARED_MEM + RAW_BUFFER_OFFSET);

    uint32_t buffer_index = 0;
    int32_t sample;

    /* Enable OCP master port for peripheral access */
    CT_CFG.SYSCFG_bit.STANDBY_INIT = 0;

    /* Initialize status */
    ctrl->status = STATUS_RAW_I2S;
    ctrl->enable_capture = 1;  /* Start capturing immediately */
    ctrl->total_samples = 0;
    ctrl->buffer_write_index = 0;
    ctrl->buffer_wrap_count = 0;
    ctrl->mcasp_errors = 0;
    ctrl->last_sample = 0;
    ctrl->min_sample = 0x7FFFFFFF;   /* Max positive for signed */
    ctrl->max_sample = 0x80000000;   /* Max negative for signed */

    /* Initialize McASP for I2S receive */
    init_mcasp_i2s(ctrl);

    ctrl->status = STATUS_CAPTURING;

    /* Main capture loop */
    while (1) {
        /* Read sample from McASP */
        sample = read_mcasp_sample(ctrl);

        /* Store in circular buffer if capture is enabled */
        if (ctrl->enable_capture) {
            sample_buffer[buffer_index] = sample;
            buffer_index++;

            if (buffer_index >= RAW_BUFFER_SIZE) {
                buffer_index = 0;
                ctrl->buffer_wrap_count++;
            }

            ctrl->buffer_write_index = buffer_index;
            ctrl->total_samples++;

            /* Update statistics */
            ctrl->last_sample = (uint32_t)sample;
            if (sample < (int32_t)ctrl->min_sample) {
                ctrl->min_sample = (uint32_t)sample;
            }
            if (sample > (int32_t)ctrl->max_sample) {
                ctrl->max_sample = (uint32_t)sample;
            }
        }
    }
}
