package cmd

// PRU shared memory constants
const (
	// PRU shared memory location
	pruSharedMemBase = 0x4A310000
	pruSharedMemSize = 0x3000

	// Audio control block offset (8KB into shared memory)
	audioControlBlockOffset = 0x2000
)
