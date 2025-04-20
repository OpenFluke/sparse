package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"time"
)

// ConstructConfig holds the configuration for a construct, loaded from JSON.
type ConstructConfig struct {
	Cubes       []Cube             `json:"cubes"`        // List of cubes with their positions
	Chains      [][]string         `json:"chains"`       // Chains of cube names to link
	JointType   string             `json:"joint_type"`   // Type of joint (e.g., "hinge")
	JointParams map[string]float64 `json:"joint_params"` // Parameters for joints
}

// Construct represents a dynamic construct with its configuration and server details.
type Construct struct {
	Config              ConstructConfig
	constructServerAddr string // IP:Port of the target server
	constructAuthPass   string // Authentication password for the server
	constructDelimiter  string // Message delimiter for the TCP protocol
	unitName            string // Unique identifier for this construct instance
}

// NewConstruct creates a new Construct instance with the given server details.
func NewConstruct(constructServerAddr, constructAuthPass, constructDelimiter string) *Construct {
	return &Construct{
		constructServerAddr: constructServerAddr,
		constructAuthPass:   constructAuthPass,
		constructDelimiter:  constructDelimiter,
	}
}

// LoadConfigFromJSON loads the construct configuration from a JSON file and applies the unitName.
func (c *Construct) LoadConfigFromJSON(filename, unitName string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read JSON file %s: %v", filename, err)
	}

	var config ConstructConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Set the unitName for this construct instance
	c.unitName = unitName

	// Prefix all cube names with the unitName
	for i := range config.Cubes {
		config.Cubes[i].Name = unitName + "_" + config.Cubes[i].Name
	}

	// Prefix all chain names with the unitName
	for i := range config.Chains {
		for j := range config.Chains[i] {
			config.Chains[i][j] = unitName + "_" + config.Chains[i][j]
		}
	}

	c.Config = config
	return nil
}

// spawnCubeWithConfig spawns a cube using the Construct's server configuration.
func (c *Construct) spawnCubeWithConfig(cube Cube, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", c.constructServerAddr)
	if err != nil {
		fmt.Printf("[Spawn] Failed to connect to %s: %v\n", c.constructServerAddr, err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(c.constructAuthPass + c.constructDelimiter)); err != nil {
		fmt.Printf("[Spawn] Auth write error to %s: %v\n", c.constructServerAddr, err)
		return
	}

	_, err = readResponse(conn)
	if err != nil {
		fmt.Printf("[Spawn] Failed to read auth response from %s: %v\n", c.constructServerAddr, err)
		return
	}

	// Check for NaN or Inf in cube.Position
	for i, coord := range cube.Position {
		if math.IsNaN(coord) || math.IsInf(coord, 0) {
			fmt.Printf("[Spawn] Invalid position for cube %s: coordinate %d is %v\n", cube.Name, i, coord)
			return
		}
	}

	spawn := Message{
		"type":      "spawn_cube",
		"cube_name": cube.Name,
		"position":  cube.Position,
		"rotation":  []float64{0, 0, 0},
		"is_base":   true,
	}
	if err := sendJSONMessage(conn, spawn); err != nil {
		fmt.Printf("[Spawn] Failed to spawn cube on %s: %v\n", c.constructServerAddr, err)
		return
	}

	fullCubeName := cube.Name + "_BASE"
	cubeListMutex.Lock()
	globalCubeList = append(globalCubeList, fullCubeName)
	cubeListMutex.Unlock()
}

// linkCubeChainsWithConfig links cube chains using the Construct's server configuration.
func (c *Construct) linkCubeChainsWithConfig(chains [][]string, jointType string, jointParams map[string]float64) error {
	conn, err := net.Dial("tcp", c.constructServerAddr)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to connect to %s: %v", c.constructServerAddr, err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(c.constructAuthPass + c.constructDelimiter)); err != nil {
		return fmt.Errorf("[linkCubeChains] Auth write error to %s: %v", c.constructServerAddr, err)
	}

	authResp, err := readResponse(conn)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to read auth response from %s: %v", c.constructServerAddr, err)
	}
	fmt.Printf("[linkCubeChains] Auth response from %s: %s\n", c.constructServerAddr, authResp)

	cmd := Message{
		"type":         "link_cube_chains",
		"chains":       chains,
		"joint_type":   jointType,
		"joint_params": jointParams,
	}

	if err := sendJSONMessage(conn, cmd); err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to send command to %s: %v", c.constructServerAddr, err)
	}

	resp, err := readResponse(conn)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Error reading response from %s: %v", c.constructServerAddr, err)
	}
	fmt.Printf("[linkCubeChains] Server response from %s: %s\n", c.constructServerAddr, resp)

	linkListMutex.Lock()
	defer linkListMutex.Unlock()
	for _, chain := range chains {
		for i := 0; i < len(chain)-1; i++ {
			cubeA := chain[i]
			cubeB := chain[i+1]
			jointName := fmt.Sprintf("joint_%s_%s_%s", jointType, cubeA, cubeB)
			globalCubeLinks = append(globalCubeLinks, CubeLink{
				JointName: jointName,
				CubeA:     cubeA,
				CubeB:     cubeB,
			})
		}
	}

	return nil
}

// Spawn spawns the construct at the specified orbit position around the planet.
func (c *Construct) Spawn(orbitPosition []float64, planetCenter []float64) error {
	fmt.Printf("\n🚀 Spawning unit: %s at planet center (%.2f, %.2f, %.2f)\n",
		c.unitName, planetCenter[0], planetCenter[1], planetCenter[2])

	// Create a copy of the cubes to adjust their positions
	adjustedCubes := make([]Cube, len(c.Config.Cubes))
	for i, cube := range c.Config.Cubes {
		adjustedCubes[i] = Cube{
			Name:     cube.Name,
			Position: make([]float64, 3),
		}
		copy(adjustedCubes[i].Position, cube.Position)
	}

	// Calculate the construct's centroid to determine its reference point
	var centroid [3]float64
	for _, cube := range adjustedCubes {
		centroid[0] += cube.Position[0]
		centroid[1] += cube.Position[1]
		centroid[2] += cube.Position[2]
	}
	count := float64(len(adjustedCubes))
	centroid[0] /= count
	centroid[1] /= count
	centroid[2] /= count

	// Adjust each cube's position relative to the new orbit position
	for i := range adjustedCubes {
		// Calculate the cube's relative position to the old centroid
		relX := adjustedCubes[i].Position[0] - centroid[0]
		relY := adjustedCubes[i].Position[1] - centroid[1]
		relZ := adjustedCubes[i].Position[2] - centroid[2]

		// Position the cube relative to the new orbit position
		adjustedCubes[i].Position[0] = orbitPosition[0] + relX
		adjustedCubes[i].Position[1] = orbitPosition[1] + relY
		adjustedCubes[i].Position[2] = orbitPosition[2] + relZ

		// Log the adjusted position for debugging
		fmt.Printf("Adjusted position for cube %s: [%.2f, %.2f, %.2f]\n",
			adjustedCubes[i].Name,
			adjustedCubes[i].Position[0],
			adjustedCubes[i].Position[1],
			adjustedCubes[i].Position[2],
		)
	}

	// Calculate the angle in the XZ plane for logging, with a fallback for zero displacement
	dx := orbitPosition[0] - planetCenter[0]
	dz := orbitPosition[2] - planetCenter[2]
	angle := 0.0
	if dx != 0 || dz != 0 {
		angle = math.Atan2(dz, dx) * (180.0 / math.Pi)
	} else {
		fmt.Println("Warning: Orbit position coincides with planet center, angle set to 0 degrees")
	}

	// Calculate the radius from the planet center to the orbit position
	radius := math.Sqrt(
		(orbitPosition[0]-planetCenter[0])*(orbitPosition[0]-planetCenter[0]) +
			(orbitPosition[1]-planetCenter[1])*(orbitPosition[1]-planetCenter[1]) +
			(orbitPosition[2]-planetCenter[2])*(orbitPosition[2]-planetCenter[2]),
	)

	fmt.Printf("🪐 Orbiting construct %s around planet at radius %.2f with angle %.2f degrees\n",
		c.unitName, radius, angle)

	// Record the orbit position as occupied
	positionMutex.Lock()
	occupiedPositions = append(occupiedPositions, occupiedPosition{
		Position: orbitPosition,
		UnitName: c.unitName,
	})
	positionMutex.Unlock()

	// Step 1: Spawn all cubes concurrently with adjusted positions
	var wg sync.WaitGroup
	wg.Add(len(adjustedCubes))
	for _, cube := range adjustedCubes {
		go c.spawnCubeWithConfig(cube, &wg)
	}
	wg.Wait()
	fmt.Printf("✅ Construct %s spawned\n", c.unitName)

	// Step 2: Link the cubes using the specified chains
	adjustedChains := make([][]string, len(c.Config.Chains))
	for i, chain := range c.Config.Chains {
		adjustedChains[i] = make([]string, len(chain))
		for j, cubeName := range chain {
			adjustedChains[i][j] = cubeName + "_BASE"
		}
	}

	if err := c.linkCubeChainsWithConfig(adjustedChains, c.Config.JointType, c.Config.JointParams); err != nil {
		return fmt.Errorf("❌ Error linking cubes for %s: %v", c.unitName, err)
	}
	fmt.Printf("🔗 Construct %s linked\n", c.unitName)

	return nil
}

// SpawnMultipleConstructs spawns multiple constructs using the same JSON template at unique positions.
func SpawnMultipleConstructs(
	numConstructs int,
	role, domain string,
	startGen, startVersion int,
	serverAddr, authPass, delimiter, jsonTemplatePath string,
	planetCenter []float64,
	offset []float64,
) error {
	// Clear occupied positions before starting
	ClearOccupiedPositions()

	// Load the JSON template to calculate the construct size
	construct := NewConstruct(serverAddr, authPass, delimiter)
	unitName := generateUnitID(role, domain, startGen, startVersion) // Temporary name for sizing
	if err := construct.LoadConfigFromJSON(jsonTemplatePath, unitName); err != nil {
		return fmt.Errorf("failed to load JSON template for sizing: %v", err)
	}

	// Calculate the construct's centroid and bounding sphere radius
	var centroid [3]float64
	for _, cube := range construct.Config.Cubes {
		centroid[0] += cube.Position[0]
		centroid[1] += cube.Position[1]
		centroid[2] += cube.Position[2]
	}
	count := float64(len(construct.Config.Cubes))
	centroid[0] /= count
	centroid[1] /= count
	centroid[2] /= count

	maxDistance := 0.0
	for _, cube := range construct.Config.Cubes {
		dx := cube.Position[0] - centroid[0]
		dy := cube.Position[1] - centroid[1]
		dz := cube.Position[2] - centroid[2]
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	// Use the offset magnitude as the base radius of the orbit
	radius := math.Sqrt(offset[0]*offset[0] + offset[1]*offset[1] + offset[2]*offset[2])
	if radius == 0 {
		radius = maxDistance * 2 // Default radius if offset is zero
	}
	// Ensure the orbit radius is large enough to accommodate the construct
	radius += maxDistance

	// Generate positions for all constructs using fibonacciSphere
	positions := fibonacciSphere(numConstructs, radius, planetCenter)
	if len(positions) != numConstructs {
		return fmt.Errorf("fibonacciSphere returned %d positions, expected %d", len(positions), numConstructs)
	}

	// Define a minimum distance threshold to avoid overlaps (e.g., 2x the construct's diameter)
	minDistance := maxDistance * 4

	// Validate positions to ensure they are not too close
	availablePositions := make([][]float64, 0, numConstructs)
	usedPositions := make([][]float64, 0, numConstructs)
	for _, pos := range positions {
		tooClose := false
		for _, usedPos := range usedPositions {
			dx := pos[0] - usedPos[0]
			dy := pos[1] - usedPos[1]
			dz := pos[2] - usedPos[2]
			distance := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if distance < minDistance {
				tooClose = true
				break
			}
		}
		if !tooClose {
			availablePositions = append(availablePositions, pos)
			usedPositions = append(usedPositions, pos)
		}
	}

	if len(availablePositions) < numConstructs {
		return fmt.Errorf("not enough unique positions: got %d, need %d", len(availablePositions), numConstructs)
	}

	// Spawn constructs at the assigned positions
	var wg sync.WaitGroup
	wg.Add(numConstructs)
	unitNames := make([]string, numConstructs)

	for i := 0; i < numConstructs; i++ {
		unitNames[i] = generateUnitID(role, domain, startGen+i/100, startVersion+i%100)
		go func(idx int) {
			defer wg.Done()

			// Create a new Construct instance
			construct := NewConstruct(serverAddr, authPass, delimiter)

			// Load the JSON template with the unique unitName
			if err := construct.LoadConfigFromJSON(jsonTemplatePath, unitNames[idx]); err != nil {
				fmt.Printf("❌ Failed to load config for %s: %v\n", unitNames[idx], err)
				return
			}

			// Spawn the construct at the assigned position
			if err := construct.Spawn(availablePositions[idx], planetCenter); err != nil {
				fmt.Printf("❌ Failed to spawn construct %s: %v\n", unitNames[idx], err)
				return
			}

			// Unfreeze the construct
			targetedUnfreezeAllCubes(unitNames[idx])
			fmt.Printf("🌀 Construct %s unfrozen\n", unitNames[idx])
		}(i)
	}

	wg.Wait()

	// Despawn all constructs sequentially with a delay
	for _, unitName := range unitNames {
		targetedDespawnAllCubes(unitName)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("🧹 All constructs despawned, simulation complete.")
	return nil
}

// ClearOccupiedPositions resets the list of occupied positions.
func ClearOccupiedPositions() {
	positionMutex.Lock()
	defer positionMutex.Unlock()
	occupiedPositions = nil
}
