package cmd

// PRU shared memory constants (must match pru_shared_memory.h)
const (
	// PRU shared memory location
	pruSharedMemBase = 0x4A310000
	pruSharedMemSize = 0x3000 // 12KB

	// PRU1 audio control block offset (was 0x2000, now after PRU0 control)
	audioControlBlockOffset = 0x0900

	// PRU1 Audio Firmware Status Codes
	statusADCRunning = 0x41554431 // "AUD1" - ADC audio firmware (40 kHz)
	statusI2SRunning = 0x49325331 // "I2S1" - I2S/McASP firmware (48 kHz)

	// Sample rates for each firmware type
	sampleRateADC = 40000 // ADC firmware: 40 kHz
	sampleRateI2S = 48000 // I2S firmware: 48 kHz
)
