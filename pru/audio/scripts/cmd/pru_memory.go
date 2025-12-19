package cmd

// PRU shared memory constants (must match pru_shared_memory.h)
const (
	// PRU shared memory location
	pruSharedMemBase = 0x4A310000
	pruSharedMemSize = 0x3000 // 12KB

	// PRU1 audio control block offset (was 0x2000, now after PRU0 control)
	audioControlBlockOffset = 0x0900
)
