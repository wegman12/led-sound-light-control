/*
 * Fixed-Point FFT Implementation for PRU - 2048 Point Version
 *
 * 2048-point Radix-2 Cooley-Tukey FFT using Q15 fixed-point arithmetic
 * Uses DDR memory for buffers (too large for PRU local memory)
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#ifndef FFT_2048_H
#define FFT_2048_H

#include <stdint.h>
#include "pru_ddr_memory.h"

/* FFT Configuration */
#define FFT_SIZE        2048        /* FFT size (must be power of 2) */
#define FFT_LOG2_SIZE   11          /* log2(2048) = 11 */

/* Q15 Fixed-Point Format
 * Range: -1.0 to +0.99997
 * 1.0 is represented as 32767 (0x7FFF)
 * -1.0 is represented as -32768 (0x8000)
 */
typedef int16_t q15_t;

#define Q15_ONE         32767       /* 1.0 in Q15 format */
#define Q15_HALF        16384       /* 0.5 in Q15 format */

/* Complex number in Q15 format */
typedef struct {
    q15_t real;
    q15_t imag;
} complex_q15_t;

/* FFT buffer structure - points to DDR memory */
typedef struct {
    complex_q15_t *data;    /* Pointer to DDR FFT buffer (2048 complex samples) */
} fft_buffer_t;

/* Q15 multiplication with rounding
 * Multiply two Q15 numbers and return Q15 result
 */
static inline q15_t q15_mul(q15_t a, q15_t b) {
    int32_t result = ((int32_t)a * (int32_t)b);
    /* Shift right by 15 bits with rounding */
    result += 0x4000;  /* Add 0.5 for rounding */
    return (q15_t)(result >> 15);
}

/* Complex multiplication in Q15
 * (a + jb) * (c + jd) = (ac - bd) + j(ad + bc)
 */
static inline complex_q15_t complex_mul_q15(complex_q15_t a, complex_q15_t b) {
    complex_q15_t result;
    result.real = q15_mul(a.real, b.real) - q15_mul(a.imag, b.imag);
    result.imag = q15_mul(a.real, b.imag) + q15_mul(a.imag, b.real);
    return result;
}

/* Bit-reversal permutation for 11-bit indices (FFT_SIZE = 2048) */
static inline uint16_t bit_reverse_11(uint16_t x) {
    uint16_t result = 0;
    result |= ((x & 0x001) << 10);
    result |= ((x & 0x002) << 8);
    result |= ((x & 0x004) << 6);
    result |= ((x & 0x008) << 4);
    result |= ((x & 0x010) << 2);
    result |= ((x & 0x020) << 0);
    result |= ((x & 0x040) >> 2);
    result |= ((x & 0x080) >> 4);
    result |= ((x & 0x100) >> 6);
    result |= ((x & 0x200) >> 8);
    result |= ((x & 0x400) >> 10);
    return result;
}

/* Pre-computed twiddle factors for 2048-point FFT
 * W_N^k = e^(-2πjk/N) = cos(2πk/N) - j*sin(2πk/N)
 * Only need N/2 = 1024 values due to symmetry
 * Stored in DDR memory
 */
extern const complex_q15_t twiddle_factors_2048[FFT_SIZE / 2];

/* Function prototypes */

/* Initialize FFT system - must be called once at startup
 * Sets up pointers to DDR memory regions
 */
void fft_2048_init_system(void);

/* Initialize FFT buffer from real samples
 * Copies samples to DDR FFT buffer with bit-reversal
 * input_samples: pointer to 2048 int16_t samples
 */
void fft_2048_init(const int16_t *input_samples);

/* Compute 2048-point FFT using Radix-2 Cooley-Tukey algorithm
 * Input: Complex samples in DDR FFT buffer (already bit-reversed)
 * Output: FFT result in-place in DDR FFT buffer
 */
void fft_2048_compute(void);

/* Get pointer to FFT result data in DDR */
complex_q15_t* fft_2048_get_data(void);

/* Calculate magnitude squared of complex number
 * Returns: |c|^2 = real^2 + imag^2 (in Q15 format, scaled to uint32)
 */
uint32_t fft_magnitude_squared(complex_q15_t c);

/* Calculate magnitude of complex number
 * Returns: |c| = sqrt(real^2 + imag^2)
 */
uint32_t fft_magnitude(complex_q15_t c);

#endif /* FFT_2048_H */
