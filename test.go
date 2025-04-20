package main

import (
	"fmt"
)

func firstSpawn() {
	// Spawn multiple constructs
	err := SpawnMultipleConstructs(
		5,                      // Number of constructs to spawn
		"ARC", "openfluke.com", // Role and domain for unit names
		1, 1, // Starting gen and version
		"127.0.0.1:14000", "my_secure_password", "<???DONE???---", // Server details
		"construct_config.json",    // Path to the JSON template
		true,                       // Orbit around planet (parameter kept for compatibility)
		[]float64{120.0, 0.0, 0.0}, // Offset for orbit radius
	)
	if err != nil {
		fmt.Printf("❌ Failed to spawn multiple constructs: %v\n", err)
		return
	}
}
