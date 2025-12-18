/*
 * McASP Register Debug Tool
 * Reads key McASP0 registers to diagnose interrupt and clock issues
 */

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>

#define MCASP0_BASE 0x48038000
#define MCASP_SIZE  0x1000

// Key register offsets from AM335x TRM Chapter 22
#define PFUNC       0x010  // Pin function register
#define PDIR        0x014  // Pin direction register
#define GBLCTL      0x044  // Global control register
#define RXSTAT      0x040  // RX status register
#define RXTDM       0x048  // RX TDM slot register
#define EVTCTLR     0x054  // RX event control register
#define RXINTCTL    0x058  // RX interrupt control register
#define ACLKXCTL    0x020  // TX clock control
#define AHCLKXCTL   0x024  // TX high freq clock control
#define AFSXCTL     0x028  // TX frame sync control
#define ACLKRCTL    0x0B0  // RX clock control
#define AFSRCTL     0x0AC  // RX frame sync control
#define SRCTL0      0x180  // Serializer 0 control (AXR0)
#define XRSRCTL0    0x180  // Transmit/receive serializer 0 control
#define XBUF0       0x200  // Transmit buffer 0
#define RBUF0       0x280  // Receive buffer 0
#define RFMT        0x0A8  // RX format register
#define RMASK       0x0A4  // RX format mask

void read_mcasp_registers(volatile void *mcasp_base) {
    volatile uint32_t *regs = (volatile uint32_t *)mcasp_base;

    printf("McASP0 Register Dump\n");
    printf("====================\n\n");

    // Pin configuration
    printf("Pin Configuration:\n");
    printf("  PFUNC    (0x010) = 0x%08X ", regs[0x010/4]);
    uint32_t pfunc = regs[0x010/4];
    printf("(AXR0: %s mode)\n", (pfunc & 0x01) ? "GPIO" : "McASP");

    printf("  PDIR     (0x014) = 0x%08X ", regs[0x014/4]);
    uint32_t pdir = regs[0x014/4];
    printf("(ACLKX:%s FSX:%s AXR0:%s)\n",
           (pdir & 0x4000) ? "OUT" : "IN",
           (pdir & 0x8000) ? "OUT" : "IN",
           (pdir & 0x0001) ? "OUT" : "IN");

    // Global control
    printf("\nGlobal Control:\n");
    printf("  GBLCTL   (0x044) = 0x%08X\n", regs[0x044/4]);
    uint32_t gblctl = regs[0x044/4];
    printf("    RCLKRST:  %d (RX clock %s)\n", (gblctl >> 0) & 1, (gblctl & 0x01) ? "running" : "in reset");
    printf("    RHCLKRST: %d (RX high-freq clock %s)\n", (gblctl >> 1) & 1, (gblctl & 0x02) ? "running" : "in reset");
    printf("    RSRCLR:   %d (RX serializer %s)\n", (gblctl >> 2) & 1, (gblctl & 0x04) ? "active" : "in reset");
    printf("    RSMRST:   %d (RX state machine %s)\n", (gblctl >> 3) & 1, (gblctl & 0x08) ? "running" : "in reset");
    printf("    RFRST:    %d (RX frame sync %s)\n", (gblctl >> 4) & 1, (gblctl & 0x10) ? "running" : "in reset");
    printf("    XCLKRST:  %d (TX clock %s)\n", (gblctl >> 8) & 1, (gblctl & 0x100) ? "running" : "in reset");
    printf("    XHCLKRST: %d (TX high-freq clock %s)\n", (gblctl >> 9) & 1, (gblctl & 0x200) ? "running" : "in reset");
    printf("    XSRCLR:   %d (TX serializer %s)\n", (gblctl >> 10) & 1, (gblctl & 0x400) ? "active" : "in reset");
    printf("    XSMRST:   %d (TX state machine %s)\n", (gblctl >> 11) & 1, (gblctl & 0x800) ? "running" : "in reset");
    printf("    XFRST:    %d (TX frame sync %s)\n", (gblctl >> 12) & 1, (gblctl & 0x1000) ? "running" : "in reset");

    // Clock configuration
    printf("\nClock Configuration:\n");
    printf("  ACLKXCTL (0x020) = 0x%08X\n", regs[0x020/4]);
    uint32_t aclkxctl = regs[0x020/4];
    printf("    CLKXDIV: %d (divider = %d)\n", aclkxctl & 0x1F, (aclkxctl & 0x1F) + 1);
    printf("    CLKXP:   %d (polarity: %s)\n", (aclkxctl >> 7) & 1, (aclkxctl & 0x80) ? "falling" : "rising");
    printf("    ASYNC:   %d (TX/RX clocks %s)\n", (aclkxctl >> 6) & 1, (aclkxctl & 0x40) ? "independent" : "synchronous");
    printf("    CLKXM:   %d (clock is %s)\n", (aclkxctl >> 5) & 1, (aclkxctl & 0x20) ? "internally generated" : "external");

    printf("  AHCLKXCTL(0x024) = 0x%08X\n", regs[0x024/4]);
    printf("  AFSXCTL  (0x028) = 0x%08X\n", regs[0x028/4]);
    uint32_t afsxctl = regs[0x028/4];
    printf("    FSXM:    %d (frame sync is %s)\n", (afsxctl >> 0) & 1, (afsxctl & 0x01) ? "internally generated" : "external");
    printf("    FSXP:    %d (polarity: %s)\n", (afsxctl >> 1) & 1, (afsxctl & 0x02) ? "rising edge" : "falling edge");

    printf("  ACLKRCTL (0x0B0) = 0x%08X\n", regs[0x0B0/4]);
    printf("  AFSRCTL  (0x0AC) = 0x%08X\n", regs[0x0AC/4]);

    // Serializer configuration
    printf("\nSerializer 0 (AXR0) Configuration:\n");
    printf("  SRCTL0   (0x180) = 0x%08X ", regs[0x180/4]);
    uint32_t srctl0 = regs[0x180/4];
    printf("(mode: ");
    switch (srctl0 & 0x03) {
        case 0: printf("inactive"); break;
        case 1: printf("transmit"); break;
        case 2: printf("receive"); break;
        default: printf("unknown"); break;
    }
    printf(")\n");

    // RX status and interrupts
    printf("\nRX Status:\n");
    printf("  RXSTAT   (0x040) = 0x%08X\n", regs[0x040/4]);
    uint32_t rxstat = regs[0x040/4];
    printf("    RERR:    %d (RX error)\n", (rxstat >> 7) & 1);
    printf("    RDMAERR: %d (RX DMA error)\n", (rxstat >> 6) & 1);
    printf("    RDATA:   %d (RX data ready)\n", (rxstat >> 5) & 1);

    printf("  RXINTCTL (0x058) = 0x%08X\n", regs[0x058/4]);
    uint32_t rxintctl = regs[0x058/4];
    printf("    ROVRN:   %d (RX overrun int %s)\n", (rxintctl >> 0) & 1, (rxintctl & 0x01) ? "ENABLED" : "disabled");
    printf("    RDATA:   %d (RX data ready int %s)\n", (rxintctl >> 5) & 1, (rxintctl & 0x20) ? "ENABLED" : "disabled");

    printf("  EVTCTLR  (0x054) = 0x%08X\n", regs[0x054/4]);
    printf("  RFMT     (0x0A8) = 0x%08X\n", regs[0x0A8/4]);
    printf("  RMASK    (0x0A4) = 0x%08X\n", regs[0x0A4/4]);

    // Buffer status
    printf("\nBuffer Status:\n");
    printf("  RBUF0    (0x280) = 0x%08X\n", regs[0x280/4]);
    printf("  XBUF0    (0x200) = 0x%08X\n", regs[0x200/4]);
}

int main(int argc, char **argv) {
    int fd = open("/dev/mem", O_RDWR | O_SYNC);
    if (fd < 0) {
        perror("Cannot open /dev/mem");
        fprintf(stderr, "Try running with sudo\n");
        return 1;
    }

    void *mcasp_base = mmap(NULL, MCASP_SIZE, PROT_READ | PROT_WRITE,
                            MAP_SHARED, fd, MCASP0_BASE);
    if (mcasp_base == MAP_FAILED) {
        perror("Cannot mmap McASP registers");
        close(fd);
        return 1;
    }

    printf("McASP0 Base Address: 0x%08X\n\n", MCASP0_BASE);

    read_mcasp_registers(mcasp_base);

    munmap(mcasp_base, MCASP_SIZE);
    close(fd);

    return 0;
}
