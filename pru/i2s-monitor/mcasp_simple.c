/*
 * Simple McASP Register Reader
 * Reads registers safely one at a time
 */

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <unistd.h>

#define MCASP0_BASE 0x48038000
#define MCASP_SIZE  0x2000

int main(int argc, char **argv) {
    int fd = open("/dev/mem", O_RDWR | O_SYNC);
    if (fd < 0) {
        perror("Cannot open /dev/mem - try sudo");
        return 1;
    }

    void *mcasp_base = mmap(NULL, MCASP_SIZE, PROT_READ | PROT_WRITE,
                            MAP_SHARED, fd, MCASP0_BASE);
    if (mcasp_base == MAP_FAILED) {
        perror("Cannot mmap McASP registers");
        close(fd);
        return 1;
    }

    volatile uint32_t *regs = (volatile uint32_t *)mcasp_base;

    printf("McASP0 Register Dump (Base: 0x%08X)\n\n", MCASP0_BASE);

    // Read key registers safely
    printf("PFUNC    (0x010) = 0x%08X\n", regs[0x010/4]);
    printf("PDIR     (0x014) = 0x%08X\n", regs[0x014/4]);
    printf("ACLKXCTL (0x020) = 0x%08X\n", regs[0x020/4]);
    printf("AHCLKXCTL(0x024) = 0x%08X\n", regs[0x024/4]);
    printf("AFSXCTL  (0x028) = 0x%08X\n", regs[0x028/4]);
    printf("RXSTAT   (0x040) = 0x%08X\n", regs[0x040/4]);
    printf("GBLCTL   (0x044) = 0x%08X\n", regs[0x044/4]);
    printf("EVTCTLR  (0x054) = 0x%08X\n", regs[0x054/4]);
    printf("RXINTCTL (0x058) = 0x%08X\n", regs[0x058/4]);
    printf("SRCTL0   (0x180) = 0x%08X\n", regs[0x180/4]);

    munmap(mcasp_base, MCASP_SIZE);
    close(fd);

    return 0;
}
