package main

import (
	"fmt"
	"math"
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

		for i := 0; i < constructsPerPlanet; i++ {
			wg.Add(1)

			go func(planetIdx, i int, center []float64) {
				defer wg.Done()

				ver := (planetIdx * constructsPerPlanet) + i + 1 // Unique gen ID
				unitName := generateUnitID(role, domain, gen, ver)

				allUnits = appendUnitSafely(allUnits, unitName)

				angle := float64(i) * paddingDegrees

				fmt.Printf("\n🚀 Spawning unit: %s at %.2f degrees\n", unitName, angle)
				buildDynamicConstruct(unitName, center, radius, angle)

				targetedUnfreezeAllCubes(unitName) // Unfreeze right after spawning
			}(planetIdx, i, center)
		}
	}

	wg.Wait()
	fmt.Println("🌀 All constructs unfrozen")

	// Wait to simulate running phase
	//time.Sleep(10 * time.Second)

	// Despawn one-by-one
	for _, unit := range allUnits {
		targetedDespawnAllCubes(unit)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("🧹 All constructs removed, simulation complete.")
}

func buildDynamicConstruct(unitName string, center []float64, radius float64, angle float64) {
	fmt.Printf("\n🚀 Spawning unit: %s at angle %.2f°\n", unitName, angle)

	// Calculate position around the sphere
	rad := angle * math.Pi / 180
	x := center[0] + radius*math.Cos(rad)
	y := center[1]
	z := center[2] + radius*math.Sin(rad)

	// Create all the cubes relative to this new x, y, z
	cubes := []Cube{
		{Name: unitName + "_head", Position: []float64{x, y + 3.6, z}},
		{Name: unitName + "_body", Position: []float64{x, y + 2.4, z}},
		{Name: unitName + "_left_arm", Position: []float64{x - 1.2, y + 2.4, z}},
		{Name: unitName + "_right_arm", Position: []float64{x + 1.2, y + 2.4, z}},
		{Name: unitName + "_left_leg", Position: []float64{x - 0.6, y + 1.2, z}},
		{Name: unitName + "_right_leg", Position: []float64{x + 0.6, y + 1.2, z}},
		{Name: unitName + "_left_foot", Position: []float64{x - 0.6, y + 0.0, z}},
		{Name: unitName + "_right_foot", Position: []float64{x + 0.6, y + 0.0, z}},
	}

	// 🛡️ Make a REAL WaitGroup
	var wg sync.WaitGroup
	wg.Add(len(cubes))

	// 🛠️ Now properly spawn all cubes
	for _, cube := range cubes {
		go spawnCube(cube, &wg) // <<< PASS the wg pointer
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
