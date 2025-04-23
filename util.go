package main

import (
	"encoding/csv"
	"fmt"
	"os"
)

// SaveToCSV saves the given data array to a CSV file with the specified filename.
// Each row in the data array is written as a comma-separated line in the CSV.
func SaveToCSV(data [][]string, filename string) error {
	// Create or open the CSV file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file %s: %v", filename, err)
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write all rows to the CSV file
	for _, row := range data {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row to CSV file %s: %v", filename, err)
		}
	}

	// Check for any errors during the write process
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error flushing CSV writer for file %s: %v", filename, err)
	}

	fmt.Printf("Successfully saved data to CSV file: %s\n", filename)
	return nil
}

// FileExists checks if a file exists at the given path and returns true if it does, false otherwise.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true // File exists
	}
	if os.IsNotExist(err) {
		return false // File does not exist
	}
	// If there's another error (e.g., permission denied), assume the file doesn't exist for safety
	return false
}
