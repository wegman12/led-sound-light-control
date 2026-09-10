/*
 * Fixed-Point FFT Implementation for PRU
 *
 * 1024-point Radix-2 Cooley-Tukey FFT using Q15 fixed-point arithmetic
 * Optimized for PRU real-time audio processing
 *
 * Author: Generated with Claude Code
 * Target: BeagleBone Black PRU1 (AM335x)
 */

#ifndef FFT_H
#define FFT_H

#include <stdint.h>

/* FFT Configuration */
#define FFT_SIZE        1024        /* FFT size (must be power of 2) */
#define FFT_LOG2_SIZE   10          /* log2(1024) = 10 */

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

/* FFT working memory layout (4KB in PRU1 DRAM at 0x1000) */
#define FFT_BUFFER_OFFSET   0x1000
typedef struct {
    complex_q15_t data[FFT_SIZE];   /* 1024 complex samples = 4KB */
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

/* Bit-reversal permutation for 10-bit indices (FFT_SIZE = 1024) */
static inline uint16_t bit_reverse_10(uint16_t x) {
    uint16_t result = 0;
    result |= ((x & 0x001) << 9);
    result |= ((x & 0x002) << 7);
    result |= ((x & 0x004) << 5);
    result |= ((x & 0x008) << 3);
    result |= ((x & 0x010) << 1);
    result |= ((x & 0x020) >> 1);
    result |= ((x & 0x040) >> 3);
    result |= ((x & 0x080) >> 5);
    result |= ((x & 0x100) >> 7);
    result |= ((x & 0x200) >> 9);
    return result;
}

/* Pre-computed twiddle factors for 1024-point FFT
 * W_N^k = e^(-2πjk/N) = cos(2πk/N) - j*sin(2πk/N)
 * Only need N/2 = 512 values due to symmetry
 */
extern const complex_q15_t twiddle_factors[FFT_SIZE / 2];

/* Function prototypes */
void fft_init(fft_buffer_t *fft_buf, const int16_t *input_samples);
void fft_compute(fft_buffer_t *fft_buf);
uint32_t fft_magnitude_squared(complex_q15_t c);
uint32_t fft_magnitude(complex_q15_t c);

#endif /* FFT_H */
