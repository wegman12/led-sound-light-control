package pru

import (
	"fmt"
	"os"
	"syscall"
)

const (
	PRUSharedMemAddr = 0x4A310000
	PRUSharedMemSize = 0x3000 // 12KB
)

// MapPRUMemory maps the PRU shared memory and returns the memory slice.
// Caller is responsible for calling syscall.Munmap when done.
func MapPRUMemory() ([]byte, error) {
	memFile, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/mem (are you running with sudo?): %w", err)
	}
	defer memFile.Close()

	mem, err := syscall.Mmap(
		int(memFile.Fd()),
		PRUSharedMemAddr,
		PRUSharedMemSize,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mmap PRU memory at 0x%X: %w", PRUSharedMemAddr, err)
	}

	return mem, nil
}
