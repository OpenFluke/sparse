package main

import (
	"fmt"
	"sync"
	"time"
)

// Safe append into allUnits with mutex to prevent race condition
var allUnitsMutex sync.Mutex

func staticBulkTest() {
	var unitNames []string

	// Spawn 5 units with different offsets and IDs
	for i := 1; i <= 5; i++ {
		unitName := generateUnitID("ARC", "openfluke.com", i, 1)
		unitNames = append(unitNames, unitName)

		offset := []float64{float64(20 * i), 120, -3} // Spread out by +20 X
		fmt.Printf("\n🚀 Spawning unit: %s\n", unitName)
		staticBuilder(unitName, offset)
	}

	// Unfreeze them all
	for _, unitName := range unitNames {
		targetedUnfreezeAllCubes(unitName)
	}
	fmt.Println("🌀 All constructs unfrozen")
	time.Sleep(3 * time.Second)

	// Despawn one-by-one with delay
	for _, unitName := range unitNames {
		targetedDespawnAllCubes(unitName)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("🧹 All constructs removed, simulation complete.")
}

func spawnConstructsAroundSphere(gen int, role string, domain string, planetCenters [][]float64, radius float64, paddingDegrees float64, constructsPerPlanet int) {
	var allUnits []string
	var wg sync.WaitGroup

	for planetIdx, center := range planetCenters {
		fmt.Printf("🪐 Setting up Planet %d at (%.2f, %.2f, %.2f)\n", planetIdx+1, center[0], center[1], center[2])

		// Generate evenly distributed points using Fibonacci sphere
		positions := fibonacciSphere(constructsPerPlanet, radius, center)

		for i, pos := range positions {
			wg.Add(1)
			go func(planetIdx, i int, position []float64) {
				defer wg.Done()
				ver := (planetIdx * constructsPerPlanet) + i + 1 // Unique version ID
				unitName := generateUnitID(role, domain, gen, ver)
				allUnits = appendUnitSafely(allUnits, unitName)
				fmt.Printf("\n🚀 Spawning unit: %s at position (%.2f, %.2f, %.2f)\n", unitName, position[0], position[1], position[2])
				buildDynamicConstruct(unitName, position, radius, 0) // Angle is unused with Fibonacci sphere
				targetedUnfreezeAllCubes(unitName)                   // Unfreeze right after spawning
			}(planetIdx, i, pos)
		}
	}

	wg.Wait()
	fmt.Println("🌀 All constructs unfrozen")

	// Despawn one-by-one with 500ms delay
	/*for _, unit := range allUnits {
		targetedDespawnAllCubes(unit)
		//time.Sleep(500 * time.Millisecond)
	}*/
	fmt.Println("🧹 All constructs removed, simulation complete.")
}

func buildDynamicConstruct(unitName string, center []float64, radius float64, angle float64) {
	fmt.Printf("\n🚀 Spawning unit: %s at position (%.2f, %.2f, %.2f)\n", unitName, center[0], center[1], center[2])

	// Create all the cubes relative to the center position
	cubes := []Cube{
		{Name: unitName + "_head", Position: []float64{center[0], center[1] + 3.6, center[2]}},
		{Name: unitName + "_body", Position: []float64{center[0], center[1] + 2.4, center[2]}},
		{Name: unitName + "_left_arm", Position: []float64{center[0] - 1.2, center[1] + 2.4, center[2]}},
		{Name: unitName + "_right_arm", Position: []float64{center[0] + 1.2, center[1] + 2.4, center[2]}},
		{Name: unitName + "_left_leg", Position: []float64{center[0] - 0.6, center[1] + 1.2, center[2]}},
		{Name: unitName + "_right_leg", Position: []float64{center[0] + 0.6, center[1] + 1.2, center[2]}},
		{Name: unitName + "_left_foot", Position: []float64{center[0] - 0.6, center[1] + 0.0, center[2]}},
		{Name: unitName + "_right_foot", Position: []float64{center[0] + 0.6, center[1] + 0.0, center[2]}},
	}

	// Spawn all cubes
	var wg sync.WaitGroup
	wg.Add(len(cubes))
	for _, cube := range cubes {
		go spawnCube(cube, &wg)
	}
	wg.Wait()
	fmt.Printf("✅ Construct %s spawned\n", unitName)

	// Define joint stiffness
	jointParams := map[string]float64{
		"limit_upper":           0.0,
		"limit_lower":           0.0,
		"motor_enable":          1.0,
		"motor_target_velocity": 0.0,
		"motor_max_impulse":     1000.0,
	}

	// Link the joints
	chains := [][]string{
		{unitName + "_head_BASE", unitName + "_body_BASE"},
		{unitName + "_body_BASE", unitName + "_left_arm_BASE", unitName + "_left_leg_BASE", unitName + "_left_foot_BASE"},
		{unitName + "_body_BASE", unitName + "_right_arm_BASE", unitName + "_right_leg_BASE", unitName + "_right_foot_BASE"},
	}
	if err := linkCubeChains(chains, "hinge", jointParams); err != nil {
		fmt.Printf("❌ Error linking cubes for %s: %v\n", unitName, err)
		return
	}
	fmt.Printf("🔗 Construct %s linked\n", unitName)
}
