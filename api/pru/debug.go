package pru

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
)

const (
	controlBlockOffset = 0x0800
)

type pruControlBlock struct {
	WriteIndex   uint32
	ReadIndex    uint32
	EventCount   uint32
	ErrorCount   uint32
	OverrunCount uint32
	Status       uint32
	ErrorCode    uint32
}

const (
	statusRunning     = 0x52554E // "RUN" in ASCII
	errorNone         = 0x0000
	errorLeaderLow    = 0x0001
	errorLeaderHigh   = 0x0002
	errorFirstLow     = 0x0003
	errorDataHigh     = 0x0004
	errorDataLow      = 0x0005
	errorStartBit     = 0x0006
	errorHeaderBits   = 0x0007
	errorSeparatorBit = 0x0008
	errorNoMatch      = 0x0009
)

func statusString(status uint32) string {
	if status == statusRunning {
		return "RUN"
	} else if status == 0 {
		return "OFF"
	}
	return fmt.Sprintf("0x%X", status)
}

func errorCodeString(code uint32) string {
	switch code {
	case errorNone:
		return "NONE"
	case errorLeaderLow:
		return "LEADER_LOW"
	case errorLeaderHigh:
		return "LEADER_HIGH"
	case errorFirstLow:
		return "FIRST_LOW"
	case errorDataHigh:
		return "DATA_HIGH"
	case errorDataLow:
		return "DATA_LOW"
	case errorStartBit:
		return "START_BIT"
	case errorHeaderBits:
		return "HEADER_BITS"
	case errorSeparatorBit:
		return "SEPARATOR_BITS"
	case errorNoMatch:
		return "NO_MATCH"
	default:
		return fmt.Sprintf("UNKNOWN(0x%04X)", code)
	}
}

func MakeDebugCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pru-debug",
		Short: "Monitor PRU control block status",
		Long:  "Continuously monitors the PRU shared memory control block, showing write/read indices, event counts, errors, and status.",
		RunE:  runDebug,
	}
}

func runDebug(cmd *cobra.Command, args []string) error {
	fmt.Println("PRU Shared Memory Debug Tool")
	fmt.Println("============================")
	fmt.Println("")

	mem, err := MapPRUMemory()
	if err != nil {
		return err
	}
	defer syscall.Munmap(mem)

	fmt.Printf("Successfully mapped PRU shared memory at 0x%X\n", PRUSharedMemAddr)
	fmt.Println("")

	controlBlock := (*pruControlBlock)(unsafe.Pointer(uintptr(unsafe.Pointer(&mem[0])) + uintptr(controlBlockOffset)))

	fmt.Println("Monitoring PRU Control Block:")
	fmt.Println("Time       | write | read | events | errors | overrun | status | error_code")
	fmt.Println("-----------+-------+------+--------+--------+---------+--------+-------------------")

	lastErrorCode := uint32(0)
	lastEvents := uint32(0)
	lastErrors := uint32(0)

	for {
		time.Sleep(500 * time.Millisecond)

		writeIdx := controlBlock.WriteIndex
		readIdx := controlBlock.ReadIndex
		events := controlBlock.EventCount
		errors := controlBlock.ErrorCount
		overrun := controlBlock.OverrunCount
		status := controlBlock.Status
		errorCode := controlBlock.ErrorCode

		timestamp := time.Now().Format("15:04:05")

		errorCodeStr := errorCodeString(errorCode)
		if errorCode != lastErrorCode {
			errorCodeStr = fmt.Sprintf("\033[31m%s\033[0m", errorCodeStr)
		}

		eventsStr := fmt.Sprintf("%d", events)
		if events != lastEvents {
			eventsStr = fmt.Sprintf("\033[32m%d\033[0m", events)
		}

		errorsStr := fmt.Sprintf("%d", errors)
		if errors != lastErrors {
			errorsStr = fmt.Sprintf("\033[33m%d\033[0m", errors)
		}

		statusStr := statusString(status)
		if status == statusRunning {
			statusStr = fmt.Sprintf("\033[32m%s\033[0m", statusStr)
		} else {
			statusStr = fmt.Sprintf("\033[31m%s\033[0m", statusStr)
		}

		fmt.Printf("%s | %5d | %4d | %6s | %6s | %7d | %6s | %s\n",
			timestamp, writeIdx, readIdx, eventsStr, errorsStr, overrun, statusStr, errorCodeStr)

		lastErrorCode = errorCode
		lastEvents = events
		lastErrors = errors
	}
}
