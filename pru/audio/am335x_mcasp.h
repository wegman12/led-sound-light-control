/*
 * AM335x McASP Register Definitions for PRU Access
 *
 * McASP (Multichannel Audio Serial Port) register definitions for direct
 * PRU control of I2S audio capture on BeagleBone Black.
 *
 * Reference: AM335x Technical Reference Manual (SPRUH73P), Chapter 22
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU (AM335x)
 */

#ifndef AM335X_MCASP_H
#define AM335X_MCASP_H

#include <stdint.h>

/*
 * =============================================================================
 * McASP Base Addresses (accessible from PRU via OCP master port)
 * =============================================================================
 */

/* McASP0 Configuration Registers */
#define MCASP0_CFG_BASE     0x48038000

/* McASP0 Data Port (FIFO) - for DMA or direct data access */
#define MCASP0_DATA_BASE    0x46000000

/*
 * =============================================================================
 * McASP Configuration Register Offsets (from MCASP0_CFG_BASE)
 * =============================================================================
 */

/* Revision and Power Management */
#define MCASP_REV           0x0000  /* Revision Register */
#define MCASP_PWRIDLESYSCONFIG 0x0004  /* Power Idle SYSCONFIG */

/* Pin Function and Direction */
#define MCASP_PFUNC         0x0010  /* Pin Function Register */
#define MCASP_PDIR          0x0014  /* Pin Direction Register */
#define MCASP_PDOUT         0x0018  /* Pin Data Out */
#define MCASP_PDIN          0x001C  /* Pin Data In (directly active pins) */
#define MCASP_PDCLR         0x0020  /* Pin Data Clear */

/* Global Control */
#define MCASP_GBLCTL        0x0044  /* Global Control Register */
#define MCASP_AMUTE         0x0048  /* Mute Control Register */
#define MCASP_DLBCTL        0x004C  /* Digital Loopback Control */
#define MCASP_DITCTL        0x0050  /* DIT Mode Control */

/* DMA Event Control */
#define MCASP_RGBLCTL       0x0060  /* Receive Global Control */
#define MCASP_RMASK         0x0064  /* Receive Format Mask */
#define MCASP_RFMT          0x0068  /* Receive Format */
#define MCASP_AFSRCTL       0x006C  /* Receive Frame Sync Control */
#define MCASP_ACLKRCTL      0x0070  /* Receive Clock Control */
#define MCASP_AHCLKRCTL     0x0074  /* Receive High-Frequency Clock Control */
#define MCASP_RTDM          0x0078  /* Receive TDM Time Slot Register */
#define MCASP_RINTCTL       0x007C  /* Receive Interrupt Control */
#define MCASP_RSTAT         0x0080  /* Receive Status Register */
#define MCASP_RSLOT         0x0084  /* Receive Current Slot */
#define MCASP_RCLKCHK       0x0088  /* Receive Clock Check */
#define MCASP_REVTCTL       0x008C  /* Receive DMA Event Control */

/* Transmit Registers (for reference, not used in receive-only mode) */
#define MCASP_XGBLCTL       0x00A0  /* Transmit Global Control */
#define MCASP_XMASK         0x00A4  /* Transmit Format Mask */
#define MCASP_XFMT          0x00A8  /* Transmit Format */
#define MCASP_AFSXCTL       0x00AC  /* Transmit Frame Sync Control */
#define MCASP_ACLKXCTL      0x00B0  /* Transmit Clock Control */
#define MCASP_AHCLKXCTL     0x00B4  /* Transmit High-Frequency Clock Control */
#define MCASP_XTDM          0x00B8  /* Transmit TDM Time Slot Register */
#define MCASP_XINTCTL       0x00BC  /* Transmit Interrupt Control */
#define MCASP_XSTAT         0x00C0  /* Transmit Status Register */
#define MCASP_XSLOT         0x00C4  /* Transmit Current Slot */
#define MCASP_XCLKCHK       0x00C8  /* Transmit Clock Check */
#define MCASP_XEVTCTL       0x00CC  /* Transmit DMA Event Control */

/* Serializer Control Registers (SRCTL0-15) */
#define MCASP_SRCTL0        0x0180  /* Serializer 0 Control */
#define MCASP_SRCTL1        0x0184  /* Serializer 1 Control */
#define MCASP_SRCTL2        0x0188  /* Serializer 2 Control */
#define MCASP_SRCTL3        0x018C  /* Serializer 3 Control */
/* ... up to SRCTL15 at 0x01BC */

/* Transmit Buffer Registers (XBUF0-15) */
#define MCASP_XBUF0         0x0200  /* Transmit Buffer 0 */
/* ... up to XBUF15 */

/* Receive Buffer Registers (RBUF0-15) */
#define MCASP_RBUF0         0x0280  /* Receive Buffer 0 */
#define MCASP_RBUF1         0x0284  /* Receive Buffer 1 */
/* ... up to RBUF15 at 0x02BC */

/*
 * =============================================================================
 * FIFO Registers (from MCASP0_CFG_BASE + 0x1000)
 * =============================================================================
 *
 * IMPORTANT: The AFIFO registers are at CFG_BASE + 0x1000 (0x48039000),
 * NOT at DATA_BASE + 0x1000 as some documentation suggests!
 *
 * The AM335x McASP has an Audio FIFO (AFIFO) that buffers data between the
 * serializers and the data port. The FIFO must be enabled for data to flow
 * through the data port at 0x46000000.
 *
 * For receive with AFIFO:
 *   1. Configure RFIFOCTL with RNUMDMA (words per event) and RENA=1
 *   2. Read from data port (0x46000000) when RFIFOSTS shows data available
 *
 * Without AFIFO enabled, you would use RBUF polling:
 *   1. Wait for SRCTL.RRDY = 1
 *   2. Read from RBUF0 at 0x48038280
 *
 * The kernel driver always uses AFIFO + DMA for efficiency.
 */

#define MCASP_WFIFOCTL      0x1000  /* Write FIFO Control (CFG_BASE + 0x1000) */
#define MCASP_WFIFOSTS      0x1004  /* Write FIFO Status */
#define MCASP_RFIFOCTL      0x1008  /* Read FIFO Control (CFG_BASE + 0x1008) */
#define MCASP_RFIFOSTS      0x100C  /* Read FIFO Status */

/* FIFO Control Register Bits (RFIFOCTL / WFIFOCTL) */
#define MCASP_FIFOCTL_NUMDMA_SHIFT  0   /* Words transferred per DMA event (8 bits) */
#define MCASP_FIFOCTL_NUMDMA_MASK   0xFF
#define MCASP_FIFOCTL_NUMEVT_SHIFT  8   /* Words in FIFO before DMA event (8 bits) */
#define MCASP_FIFOCTL_NUMEVT_MASK   0xFF
#define MCASP_FIFOCTL_ENA           (1 << 16)  /* FIFO Enable */

/* FIFO Status Register Bits (RFIFOSTS / WFIFOSTS) */
#define MCASP_FIFOSTS_LEVEL_MASK    0xFF  /* Current FIFO level (words) */

/*
 * =============================================================================
 * Register Bit Definitions
 * =============================================================================
 */

/* GBLCTL / RGBLCTL / XGBLCTL - Global Control Register Bits */
#define MCASP_GBLCTL_RCLKRST    (1 << 0)   /* Receive Clock Divider Reset */
#define MCASP_GBLCTL_RHCLKRST   (1 << 1)   /* Receive High-Freq Clock Divider Reset */
#define MCASP_GBLCTL_RSRCLR     (1 << 2)   /* Receive Serializer Clear */
#define MCASP_GBLCTL_RSMRST     (1 << 3)   /* Receive State Machine Reset */
#define MCASP_GBLCTL_RFRST      (1 << 4)   /* Receive Frame Sync Generator Reset */
#define MCASP_GBLCTL_XCLKRST    (1 << 8)   /* Transmit Clock Divider Reset */
#define MCASP_GBLCTL_XHCLKRST   (1 << 9)   /* Transmit High-Freq Clock Divider Reset */
#define MCASP_GBLCTL_XSRCLR     (1 << 10)  /* Transmit Serializer Clear */
#define MCASP_GBLCTL_XSMRST     (1 << 11)  /* Transmit State Machine Reset */
#define MCASP_GBLCTL_XFRST      (1 << 12)  /* Transmit Frame Sync Generator Reset */

/* RFMT - Receive Format Register Bits */
#define MCASP_RFMT_RROT_SHIFT   0          /* Receive Bit Rotation (3 bits) */
#define MCASP_RFMT_RBUSEL       (1 << 3)   /* Receive Buffer Origin (0=CFG, 1=DATA) */
#define MCASP_RFMT_RSSZ_SHIFT   4          /* Receive Slot Size (4 bits) */
#define MCASP_RFMT_RPBIT_SHIFT  8          /* Receive Bit Pad Value Position (5 bits) */
#define MCASP_RFMT_RPAD_SHIFT   13         /* Receive Pad Value (2 bits) */
#define MCASP_RFMT_RRVRS        (1 << 15)  /* Receive Bit Reverse (0=MSB, 1=LSB first) */
#define MCASP_RFMT_RDATDLY_SHIFT 16        /* Receive Data Delay (2 bits) */

/* Slot Size Values (for RSSZ/XSSZ) */
#define MCASP_SLOT_SIZE_8       0x03       /* 8 bits */
#define MCASP_SLOT_SIZE_12      0x05       /* 12 bits */
#define MCASP_SLOT_SIZE_16      0x07       /* 16 bits */
#define MCASP_SLOT_SIZE_20      0x09       /* 20 bits */
#define MCASP_SLOT_SIZE_24      0x0B       /* 24 bits */
#define MCASP_SLOT_SIZE_28      0x0D       /* 28 bits */
#define MCASP_SLOT_SIZE_32      0x0F       /* 32 bits */

/* Data Delay Values (for RDATDLY/XDATDLY) - I2S uses 1-bit delay */
#define MCASP_DATDLY_0BIT       0x00       /* No delay (left-justified) */
#define MCASP_DATDLY_1BIT       0x01       /* 1-bit delay (I2S standard) */
#define MCASP_DATDLY_2BIT       0x02       /* 2-bit delay */

/* AFSRCTL - Receive Frame Sync Control Register Bits */
#define MCASP_AFSRCTL_FSRP      (1 << 0)   /* Receive Frame Sync Polarity */
#define MCASP_AFSRCTL_FSRM      (1 << 1)   /* Receive Frame Sync Mode (0=ext, 1=int) */
#define MCASP_AFSRCTL_FRWID     (1 << 4)   /* Receive Frame Sync Width (0=bit, 1=word) */
#define MCASP_AFSRCTL_RMOD_SHIFT 7         /* Receive Frame Sync Mode (7 bits) */

/* ACLKRCTL - Receive Clock Control Register Bits */
#define MCASP_ACLKRCTL_CLKRDIV_SHIFT 0     /* Receive Clock Divide Ratio (5 bits) */
#define MCASP_ACLKRCTL_CLKRM    (1 << 5)   /* Receive Clock Mode (0=ext, 1=int) */
#define MCASP_ACLKRCTL_CLKRP    (1 << 7)   /* Receive Clock Polarity */

/* AHCLKRCTL - Receive High-Frequency Clock Control Register Bits */
#define MCASP_AHCLKRCTL_HCLKRDIV_SHIFT 0   /* Receive HCLK Divide Ratio (12 bits) */
#define MCASP_AHCLKRCTL_HCLKRP  (1 << 14)  /* Receive HCLK Polarity */
#define MCASP_AHCLKRCTL_HCLKRM  (1 << 15)  /* Receive HCLK Mode (0=ext, 1=int) */

/* RSTAT - Receive Status Register Bits */
#define MCASP_RSTAT_ROVRN       (1 << 0)   /* Receive Overrun */
#define MCASP_RSTAT_RSYNCERR    (1 << 1)   /* Receive Sync Error */
#define MCASP_RSTAT_RCKFAIL     (1 << 2)   /* Receive Clock Failure */
#define MCASP_RSTAT_RTDMSLOT    (1 << 3)   /* Receive TDM Slot */
#define MCASP_RSTAT_RLAST       (1 << 4)   /* Receive Last Slot */
#define MCASP_RSTAT_RDATA       (1 << 5)   /* Receive Data Ready */
#define MCASP_RSTAT_RSTAFRM     (1 << 6)   /* Receive Start of Frame */
#define MCASP_RSTAT_RERR        (1 << 8)   /* Receive Error */

/* SRCTL - Serializer Control Register Bits */
#define MCASP_SRCTL_SRMOD_SHIFT 0          /* Serializer Mode (2 bits) */
#define MCASP_SRCTL_DISMOD_SHIFT 2         /* Pin Disable Mode (2 bits) */
#define MCASP_SRCTL_RRDY        (1 << 4)   /* Receive Ready */
#define MCASP_SRCTL_XRDY        (1 << 5)   /* Transmit Ready */

/* Serializer Mode Values */
#define MCASP_SRMOD_INACTIVE    0x00       /* Inactive */
#define MCASP_SRMOD_TX          0x01       /* Transmit */
#define MCASP_SRMOD_RX          0x02       /* Receive */

/* PDIR - Pin Direction Register Bits */
#define MCASP_PDIR_AXR0         (1 << 0)   /* AXR0 pin direction */
#define MCASP_PDIR_AXR1         (1 << 1)   /* AXR1 pin direction */
#define MCASP_PDIR_ACLKX        (1 << 26)  /* ACLKX pin direction */
#define MCASP_PDIR_AHCLKX       (1 << 27)  /* AHCLKX pin direction */
#define MCASP_PDIR_AFSX         (1 << 28)  /* AFSX pin direction */
#define MCASP_PDIR_ACLKR        (1 << 29)  /* ACLKR pin direction */
#define MCASP_PDIR_AHCLKR       (1 << 30)  /* AHCLKR pin direction */
#define MCASP_PDIR_AFSR         (1 << 31)  /* AFSR pin direction */

/*
 * =============================================================================
 * I2S Configuration Values for 48 kHz Stereo
 * =============================================================================
 *
 * Clock Configuration for 48 kHz:
 *   Master Clock (AHCLKR input): 24.576 MHz (from CLKOUT2 via jumper)
 *   Bit Clock (ACLKX output): 3.072 MHz = 24.576 MHz / 8
 *   Frame Sync (FSX output): 48 kHz = 3.072 MHz / 64
 *   Slots per frame: 2 (left + right)
 *   Bits per slot: 32
 *   Total bits per frame: 64
 *
 * Clock Divider Calculation:
 *   HCLKRDIV = 0 (use AHCLKR directly, no division)
 *   CLKRDIV = 7 (divide AHCLKR by 8 for bit clock)
 *   RMOD = 2 (2 slots per frame for I2S)
 */

#define MCASP_I2S_HCLKDIV       0          /* AHCLKR passthrough (24.576 MHz) */
#define MCASP_I2S_CLKDIV        7          /* Divide by 8 for BCLK (3.072 MHz) */
#define MCASP_I2S_SLOTS         2          /* 2 TDM slots (stereo) */
#define MCASP_I2S_SLOT_SIZE     MCASP_SLOT_SIZE_32  /* 32-bit slots */

/*
 * =============================================================================
 * Helper Macros for Register Access
 * =============================================================================
 */

/* Read McASP configuration register */
#define MCASP_CFG_READ(offset) \
    (*((volatile uint32_t *)(MCASP0_CFG_BASE + (offset))))

/* Write McASP configuration register */
#define MCASP_CFG_WRITE(offset, value) \
    (*((volatile uint32_t *)(MCASP0_CFG_BASE + (offset))) = (value))

/* Read McASP data register */
#define MCASP_DATA_READ(offset) \
    (*((volatile uint32_t *)(MCASP0_DATA_BASE + (offset))))

/* Write McASP data register */
#define MCASP_DATA_WRITE(offset, value) \
    (*((volatile uint32_t *)(MCASP0_DATA_BASE + (offset))) = (value))

/* Read receive buffer (RBUF0 for serializer 0) */
#define MCASP_READ_RBUF0() \
    MCASP_CFG_READ(MCASP_RBUF0)

/* Check if receive data is ready */
#define MCASP_RDATA_READY() \
    (MCASP_CFG_READ(MCASP_RSTAT) & MCASP_RSTAT_RDATA)

/* Clear receive status flags */
#define MCASP_CLEAR_RSTAT(flags) \
    MCASP_CFG_WRITE(MCASP_RSTAT, (flags))

#endif /* AM335X_MCASP_H */
