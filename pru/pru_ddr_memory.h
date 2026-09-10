/*
 * PRU DDR Memory Layout for 2048-point FFT
 *
 * This header defines the DDR memory region reserved for PRU FFT processing.
 * The memory is reserved via device tree overlay (BB-PRU-DDR-00A0.dts) and
 * accessible to PRU via L3 interconnect.
 *
 * Physical Address: 0x9FF00000 (from both ARM and PRU perspective)
 * Total Size: 64KB (0x10000)
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black AM335x PRU
 */

#ifndef PRU_DDR_MEMORY_H
#define PRU_DDR_MEMORY_H

/*
 * =============================================================================
 * DDR MEMORY BASE ADDRESS
 * =============================================================================
 *
 * The PRU can access DDR memory directly via the L3 interconnect.
 * DDR on BeagleBone Black (512MB): 0x80000000 - 0x9FFFFFFF
 *
 * We reserve 64KB at 0x9FF00000 (end of DDR region) to avoid conflicts
 * with Linux kernel and user applications.
 *
 * Access characteristics:
 *   - Latency: ~100-200 PRU cycles (vs 1-3 for local memory)
 *   - Bandwidth: Sufficient for FFT at 20-50 Hz update rate
 *   - No caching: PRU accesses are uncached, so data is always coherent
 */
#define PRU_DDR_BASE            0x9FF00000
#define PRU_DDR_SIZE            0x00010000  /* 64KB */

/*
 * =============================================================================
 * DDR MEMORY LAYOUT (64KB total)
 * =============================================================================
 *
 * Offset      Size    Description
 * ----------  ------  ---------------------------------------------------------
 * 0x0000      8KB     Sample Buffer A (2048 x 32-bit I2S samples)
 * 0x2000      8KB     Sample Buffer B (2048 x 32-bit I2S samples)
 * 0x4000      8KB     FFT Working Buffer (2048 x complex_q15_t)
 * 0x6000      8KB     Twiddle Factors (1024 x complex_q15_t)
 * 0x8000      32KB    Reserved for future use
 *
 * =============================================================================
 */

/*
 * -----------------------------------------------------------------------------
 * Sample Buffers (Double Buffering)
 * -----------------------------------------------------------------------------
 * Two 8KB buffers for 2048 x 32-bit samples each.
 * PRU fills one buffer while FFT processes the other.
 */
#define DDR_SAMPLE_BUFFER_A_OFFSET      0x0000
#define DDR_SAMPLE_BUFFER_A_SIZE        0x2000  /* 8KB */
#define DDR_SAMPLE_BUFFER_A_SAMPLES     2048

#define DDR_SAMPLE_BUFFER_B_OFFSET      0x2000
#define DDR_SAMPLE_BUFFER_B_SIZE        0x2000  /* 8KB */
#define DDR_SAMPLE_BUFFER_B_SAMPLES     2048

/*
 * -----------------------------------------------------------------------------
 * FFT Working Buffer
 * -----------------------------------------------------------------------------
 * 8KB buffer for 2048 complex Q15 values (2048 x 4 bytes = 8KB)
 * Used for in-place FFT computation.
 */
#define DDR_FFT_BUFFER_OFFSET           0x4000
#define DDR_FFT_BUFFER_SIZE             0x2000  /* 8KB */

/*
 * -----------------------------------------------------------------------------
 * Twiddle Factors
 * -----------------------------------------------------------------------------
 * Pre-computed twiddle factors for 2048-point FFT.
 * Need N/2 = 1024 complex values = 4KB, but we reserve 8KB for alignment.
 */
#define DDR_TWIDDLE_OFFSET              0x6000
#define DDR_TWIDDLE_SIZE                0x2000  /* 8KB reserved, 4KB used */
#define DDR_TWIDDLE_COUNT               1024    /* N/2 twiddle factors */

/*
 * -----------------------------------------------------------------------------
 * Reserved Space
 * -----------------------------------------------------------------------------
 */
#define DDR_RESERVED_OFFSET             0x8000
#define DDR_RESERVED_SIZE               0x8000  /* 32KB */

/*
 * -----------------------------------------------------------------------------
 * Absolute Addresses (for direct PRU access)
 * -----------------------------------------------------------------------------
 */
#define DDR_SAMPLE_BUFFER_A_ADDR        (PRU_DDR_BASE + DDR_SAMPLE_BUFFER_A_OFFSET)
#define DDR_SAMPLE_BUFFER_B_ADDR        (PRU_DDR_BASE + DDR_SAMPLE_BUFFER_B_OFFSET)
#define DDR_FFT_BUFFER_ADDR             (PRU_DDR_BASE + DDR_FFT_BUFFER_OFFSET)
#define DDR_TWIDDLE_ADDR                (PRU_DDR_BASE + DDR_TWIDDLE_OFFSET)

/*
 * -----------------------------------------------------------------------------
 * FFT Configuration for 2048 points
 * -----------------------------------------------------------------------------
 */
#define FFT_SIZE_2048                   2048
#define FFT_LOG2_SIZE_2048              11      /* log2(2048) = 11 */

/*
 * -----------------------------------------------------------------------------
 * Compile-time Boundary Checks
 * -----------------------------------------------------------------------------
 */
#define DDR_SAMPLE_BUFFER_A_END     (DDR_SAMPLE_BUFFER_A_OFFSET + DDR_SAMPLE_BUFFER_A_SIZE)
#define DDR_SAMPLE_BUFFER_B_END     (DDR_SAMPLE_BUFFER_B_OFFSET + DDR_SAMPLE_BUFFER_B_SIZE)
#define DDR_FFT_BUFFER_END          (DDR_FFT_BUFFER_OFFSET + DDR_FFT_BUFFER_SIZE)
#define DDR_TWIDDLE_END             (DDR_TWIDDLE_OFFSET + DDR_TWIDDLE_SIZE)
#define DDR_RESERVED_END            (DDR_RESERVED_OFFSET + DDR_RESERVED_SIZE)

#if DDR_SAMPLE_BUFFER_A_END > DDR_SAMPLE_BUFFER_B_OFFSET
#error "DDR Sample buffer A overlaps with buffer B"
#endif

#if DDR_SAMPLE_BUFFER_B_END > DDR_FFT_BUFFER_OFFSET
#error "DDR Sample buffer B overlaps with FFT buffer"
#endif

#if DDR_FFT_BUFFER_END > DDR_TWIDDLE_OFFSET
#error "DDR FFT buffer overlaps with twiddle factors"
#endif

#if DDR_RESERVED_END > PRU_DDR_SIZE
#error "DDR layout exceeds allocated size"
#endif

#endif /* PRU_DDR_MEMORY_H */
