/*
 * PRU Shared Memory Layout
 *
 * This header defines the shared memory organization for both PRU0 (IR Remote)
 * and PRU1 (Audio/FFT). All PRU programs should include this header to ensure
 * consistent memory allocation and avoid conflicts.
 *
 * Physical Address: 0x4A310000 (from ARM/Linux perspective)
 * PRU Address: 0x00010000 (from PRU perspective)
 * Total Size: 12KB (0x3000)
 *
 * NOTE: For 2048-point FFT, sample buffers are moved to DDR memory.
 *       See pru_ddr_memory.h for DDR layout details.
 *       Shared memory is still used for the control block.
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black AM335x
 */

#ifndef PRU_SHARED_MEMORY_H
#define PRU_SHARED_MEMORY_H

/*
 * =============================================================================
 * SHARED MEMORY MAP (12KB total)
 * =============================================================================
 *
 * Offset      Size    Description
 * ----------  ------  ---------------------------------------------------------
 * 0x0000      2KB     PRU0 IR Ring Buffer (256 events x 8 bytes)
 * 0x0800      256B    PRU0 IR Control Block
 * 0x0900      256B    PRU1 Audio Control Block
 * 0x0A00      4KB     PRU1 Audio Sample Buffer A (1024 x 32-bit samples)
 * 0x1A00      4KB     PRU1 Audio Sample Buffer B (1024 x 32-bit samples)
 * 0x2A00      1.5KB   Reserved for future use
 *
 * =============================================================================
 */

/* Base address of PRU shared memory (from PRU's perspective) */
#define PRU_SHARED_MEM_BASE         0x00010000

/* Total shared memory size */
#define PRU_SHARED_MEM_SIZE         0x00003000  /* 12KB */

/*
 * -----------------------------------------------------------------------------
 * PRU0 IR Remote Detector Memory Layout
 * -----------------------------------------------------------------------------
 */

/* Ring buffer for IR button events (256 events x 8 bytes = 2KB) */
#define PRU0_IR_RING_BUFFER_OFFSET  0x00000000
#define PRU0_IR_RING_BUFFER_SIZE    0x00000800  /* 2KB */
#define PRU0_IR_RING_BUFFER_ENTRIES 256
#define PRU0_IR_EVENT_SIZE          8           /* bytes per event */

/* Control block for PRU0 IR detector (28 bytes used, 256 bytes reserved) */
#define PRU0_IR_CONTROL_OFFSET      0x00000800
#define PRU0_IR_CONTROL_SIZE        0x00000100  /* 256B reserved */

/*
 * -----------------------------------------------------------------------------
 * PRU1 Audio Sampling and FFT Memory Layout
 * -----------------------------------------------------------------------------
 */

/* Audio control block (currently ~100 bytes, 256 bytes reserved for expansion) */
#define PRU1_AUDIO_CONTROL_OFFSET   0x00000900
#define PRU1_AUDIO_CONTROL_SIZE     0x00000100  /* 256B reserved */

/* Sample Buffer A for 24-bit I2S audio (1024 x 4 bytes = 4KB) */
#define PRU1_SAMPLE_BUFFER_A_OFFSET 0x00000A00
#define PRU1_SAMPLE_BUFFER_A_SIZE   0x00001000  /* 4KB */

/* Sample Buffer B for 24-bit I2S audio (1024 x 4 bytes = 4KB) */
#define PRU1_SAMPLE_BUFFER_B_OFFSET 0x00001A00
#define PRU1_SAMPLE_BUFFER_B_SIZE   0x00001000  /* 4KB */

/* Audio buffer configuration */
#define PRU1_AUDIO_BUFFER_SAMPLES   1024        /* Samples per buffer */
#define PRU1_AUDIO_SAMPLE_SIZE      4           /* Bytes per sample (32-bit) */

/* Reserved space for future expansion */
#define PRU_RESERVED_OFFSET         0x00002A00
#define PRU_RESERVED_SIZE           0x00000600  /* 1.5KB */

/*
 * -----------------------------------------------------------------------------
 * Memory Boundary Checks (compile-time validation)
 * -----------------------------------------------------------------------------
 */

/* End of each region (exclusive) */
#define PRU0_IR_RING_BUFFER_END     (PRU0_IR_RING_BUFFER_OFFSET + PRU0_IR_RING_BUFFER_SIZE)
#define PRU0_IR_CONTROL_END         (PRU0_IR_CONTROL_OFFSET + PRU0_IR_CONTROL_SIZE)
#define PRU1_AUDIO_CONTROL_END      (PRU1_AUDIO_CONTROL_OFFSET + PRU1_AUDIO_CONTROL_SIZE)
#define PRU1_SAMPLE_BUFFER_A_END    (PRU1_SAMPLE_BUFFER_A_OFFSET + PRU1_SAMPLE_BUFFER_A_SIZE)
#define PRU1_SAMPLE_BUFFER_B_END    (PRU1_SAMPLE_BUFFER_B_OFFSET + PRU1_SAMPLE_BUFFER_B_SIZE)

/* Verify no overlap (compile-time checks) */
#if PRU0_IR_RING_BUFFER_END > PRU0_IR_CONTROL_OFFSET
#error "PRU0 ring buffer overlaps with PRU0 control block"
#endif

#if PRU0_IR_CONTROL_END > PRU1_AUDIO_CONTROL_OFFSET
#error "PRU0 control block overlaps with PRU1 audio control"
#endif

#if PRU1_AUDIO_CONTROL_END > PRU1_SAMPLE_BUFFER_A_OFFSET
#error "PRU1 audio control overlaps with sample buffer A"
#endif

#if PRU1_SAMPLE_BUFFER_A_END > PRU1_SAMPLE_BUFFER_B_OFFSET
#error "Sample buffer A overlaps with sample buffer B"
#endif

#if PRU1_SAMPLE_BUFFER_B_END > PRU_SHARED_MEM_SIZE
#error "Sample buffer B exceeds shared memory bounds"
#endif

#endif /* PRU_SHARED_MEMORY_H */
