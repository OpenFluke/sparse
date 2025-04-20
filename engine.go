package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	serverAddr = "127.0.0.1:14000"
	//serverAddr = "192.168.0.227:10002"
	//authPass  = "my_secure_password"
	delimiter = "<???DONE???---"
)

var scanner = &SparseScanner{}

type Message map[string]interface{}

type Cube struct {
	Name     string
	Position []float64
	UnitName string // Optional: metadata tag
}

type CubeLink struct {
	JointName string
	CubeA     string
	CubeB     string
}

var (
	globalCubeList    []string
	cubeListMutex     sync.Mutex
	globalCubeLinks   []CubeLink
	linkListMutex     sync.Mutex
	occupiedPositions []occupiedPosition // Track positions of spawned constructs
	positionMutex     sync.Mutex
)

type occupiedPosition struct {
	Position []float64
	UnitName string
}

func sendJSONMessage(conn net.Conn, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, []byte(delimiter)...)
	_, err = conn.Write(data)
	return err
}

func readResponse(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var builder strings.Builder
	for {
		line, err := reader.ReadString('-')
		if err != nil {
			break
		}
		builder.WriteString(line)
		if strings.Contains(line, delimiter) {
			break
		}
	}
	full := strings.ReplaceAll(builder.String(), delimiter, "")
	return strings.TrimSpace(full), nil
}

func spawnCube(cube Cube, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("[Spawn] Failed to connect:", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
		fmt.Println("[Spawn] Auth write error:", err)
		return
	}
	_, err = readResponse(conn)
	if err != nil {
		fmt.Println("[Spawn] Failed to read auth response:", err)
		return
	}

	spawn := Message{
		"type":      "spawn_cube",
		"cube_name": cube.Name,
		"position":  cube.Position,
		"rotation":  []float64{0, 0, 0},
		"is_base":   true,
	}
	if err := sendJSONMessage(conn, spawn); err != nil {
		fmt.Println("[Spawn] Failed to spawn cube:", err)
		return
	}

	fullCubeName := cube.Name + "_BASE"
	cubeListMutex.Lock()
	globalCubeList = append(globalCubeList, fullCubeName)
	cubeListMutex.Unlock()
}

func unfreezeAllCubes() {
	var wg sync.WaitGroup
	for _, cube := range globalCubeList {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				fmt.Println("[Unfreeze] Failed to connect:", err)
				return
			}
			defer conn.Close()

			if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
				return
			}
			_, _ = readResponse(conn)

			unfreeze := Message{
				"type":      "freeze_cube",
				"cube_name": name,
				"freeze":    false,
			}
			sendJSONMessage(conn, unfreeze)
		}(cube)
	}
	wg.Wait()
}

// setJointParam sends a JSON command to set a specific parameter for a joint.
func setJointParam(conn net.Conn, jointName, paramName string, value float64) {
	// Build the command message.
	cmd := Message{
		"type":       "set_joint_param",
		"joint_name": jointName,
		"param_name": paramName,
		"value":      value,
	}
	// Send the JSON command.
	if err := sendJSONMessage(conn, cmd); err != nil {
		fmt.Printf("[setJointParam] Failed to send command for joint %s: %v\n", jointName, err)
		return
	}
	// Optionally, read the server response.
	resp, err := readResponse(conn)
	if err != nil {
		fmt.Printf("[setJointParam] Error reading response for joint %s: %v\n", jointName, err)
		return
	}
	fmt.Printf("[setJointParam] Joint %s param %s set to %v, response: %s\n", jointName, paramName, value, resp)
}

func setJointParams(conn net.Conn, jointName string, params map[string]float64) {
	cmd := Message{
		"type":       "set_joint_params",
		"joint_name": jointName,
		"params":     params,
	}
	if err := sendJSONMessage(conn, cmd); err != nil {
		fmt.Printf("[setJointParams] Failed to send joint params for %s: %v\n", jointName, err)
		return
	}
	resp, err := readResponse(conn)
	if err != nil {
		fmt.Printf("[setJointParams] Read error for %s: %v\n", jointName, err)
		return
	}
	fmt.Printf("[setJointParams] %s response: %s\n", jointName, resp)
}

func linkCubeChains(chains [][]string, jointType string, jointParams map[string]float64) error {
	// Establish TCP connection
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to connect: %v", err)
	}
	defer conn.Close()

	// Authenticate
	if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
		return fmt.Errorf("[linkCubeChains] Auth write error: %v", err)
	}
	authResp, err := readResponse(conn)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to read auth response: %v", err)
	}
	fmt.Println("[linkCubeChains] Auth response:", authResp)

	// Construct the command
	cmd := Message{
		"type":         "link_cube_chains",
		"chains":       chains,
		"joint_type":   jointType,
		"joint_params": jointParams,
	}

	// Send the command
	if err := sendJSONMessage(conn, cmd); err != nil {
		return fmt.Errorf("[linkCubeChains] Failed to send command: %v", err)
	}

	// Read response (optional)
	resp, err := readResponse(conn)
	if err != nil {
		return fmt.Errorf("[linkCubeChains] Error reading response: %v", err)
	}
	fmt.Println("[linkCubeChains] Server response:", resp)

	// Update globalCubeLinks for tracking (optional, adjust as needed)
	linkListMutex.Lock()
	defer linkListMutex.Unlock()
	for _, chain := range chains {
		for i := 0; i < len(chain)-1; i++ {
			cubeA := chain[i]
			cubeB := chain[i+1]
			jointName := fmt.Sprintf("joint_%s_%s_%s", jointType, cubeA, cubeB) // Simplified name
			globalCubeLinks = append(globalCubeLinks, CubeLink{
				JointName: jointName,
				CubeA:     cubeA,
				CubeB:     cubeB,
			})
		}
	}

	return nil
}

func targetedUnfreezeAllCubes(unitName string) {
	var wg sync.WaitGroup
	for _, cube := range globalCubeList {
		if strings.HasPrefix(cube, unitName+"_") {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				conn, err := net.Dial("tcp", serverAddr)
				if err != nil {
					fmt.Printf("[%s] [Unfreeze] Connection failed: %v\n", unitName, err)
					return
				}
				defer conn.Close()

				if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
					return
				}
				_, _ = readResponse(conn)

				unfreeze := Message{
					"type":      "freeze_cube",
					"cube_name": name,
					"freeze":    false,
				}
				sendJSONMessage(conn, unfreeze)
			}(cube)
		}
	}
	wg.Wait()
	fmt.Printf("🌀 [%s] All cubes unfrozen.\n", unitName)
}

func main() {

	// --- Discovery Phase ---
	scanner.InitSparseScanner(
		[]string{
			"192.168.0.229",
			"192.168.0.227",
		},
		10002, // starting port
	)

	scanner.ScanAllPods()
	scanner.PrintSummary()

	firstSpawn()

	//singlePod()
	//centers := scanner.ExtractPlanetCenters()

	// List of planets' center coordinates
	/*planetCenters := [][]float64{
		{0, 0, 0},
	}

	// Settings
	role := "ARC"
	domain := "openfluke.com"
	radius := 120.0             // Distance from center
	paddingDegrees := 360.0 / 8 // Evenly spaced around sphere (for 8 constructs)
	constructsPerPlanet := 10   // How many per planet

	nukeAllCubes()
	// Spawn around all planets
	spawnConstructsAroundSphere(1, role, domain, planetCenters, radius, paddingDegrees, constructsPerPlanet)

	singlePod()

	nukeAllCubes()*/
}

func singlePod() {
	// Step 1: Initialize the SparseScanner
	scannerSingle := NewSparseScanner([]string{"127.0.0.1"}, 14000)

	// Step 2: Create a PodResult
	podResult := scannerSingle.ScanSinglePod("127.0.0.1", 14000)

	// Step 3: Add the PodResult to the scanner
	scannerSingle.AddPodResult(podResult)

	// Step 4: Verify the results
	//fmt.Println("Scanner Results:", scannerSingle.Results)
	fmt.Println("Planets Map:", scannerSingle.PlanetsMap)
	fmt.Println("Cubes Map:", scannerSingle.CubesMap)
	scannerSingle.PrintSummary()

	unitName := "[ARC]-OC-gen1-v10"

	singleFilter := scannerSingle.GetCubesByPrefix(unitName)
	fmt.Println(singleFilter)

	// In buildDynamicConstruct, after linking cubes
	/*connections, err := scannerSingle.GetCubesAndConnections(unitName)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve cubes and connections for %s: %v\n", unitName, err)
		return
	}
	fmt.Printf("🔗 Cubes and Connections for %s:\n", unitName)
	for _, conn := range connections {
		fmt.Printf("  Cube: %s\n", conn.CubeName)
		for _, joint := range conn.Joints {
			if joint.ConnectedCube != "" {
				fmt.Printf("    Joint %s connects to %s\n", joint.JointName, joint.ConnectedCube)
			} else {
				fmt.Printf("    Joint %s (connected cube unknown)\n", joint.JointName)
			}
		}
	}*/

	// In buildDynamicConstruct, after linking cubes
	connections, err := scannerSingle.GetCubesAndConnectionsParallel(unitName)
	if err != nil {
		fmt.Printf("❌ Failed to retrieve cubes and connections for %s: %v\n", unitName, err)
		return
	}
	fmt.Printf("🔗 Cubes and Connections for %s:\n", unitName)
	for _, conn := range connections {
		fmt.Printf("  Cube: %s\n", conn.CubeName)
		for _, joint := range conn.Joints {
			if joint.ConnectedCube != "" {
				fmt.Printf("    Joint %s connects to %s\n", joint.JointName, joint.ConnectedCube)
			} else {
				fmt.Printf("    Joint %s (connected cube unknown)\n", joint.JointName)
			}
		}
	}
}
