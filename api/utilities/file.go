package utilities

import "os"

// FileExists checks if a file or directory exists at the given path.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false // File or directory does not exist
	}
	// Other errors might indicate permission issues or other problems,
	// but for existence, we only care about os.IsNotExist.
	return err == nil // File or directory exists if there's no error
}
