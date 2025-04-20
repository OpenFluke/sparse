package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- CONFIGURABLE CONSTANTS ---
const (
	startPort  = 10002
	portStep   = 3
	numPods    = 10
	authPass   = "my_secure_password"
	endMarker  = "<???DONE???---"
	timeoutSec = 10
)

// --- MAIN STRUCTS ---

type SparseScanner struct {
	Hosts      []string
	StartPort  int
	PortStep   int
	NumPods    int
	AuthPass   string
	EndMarker  string
	TimeoutSec int

	Results    []PodResult
	PlanetsMap map[string]PlanetRecord
	CubesMap   map[string]string // cubeName -> host
}

type PlanetRecord struct {
	Name        string
	Coordinates [3]float64
	Host        string
	Port        int
}

type PodResult struct {
	Host    string
	Port    int
	Success bool
	Error   string
	Cubes   []string
	Planets []Planet
}

type Planet struct {
	Position          map[string]float64   `json:"Position"`
	Seed              int                  `json:"Seed"`
	Name              string               `json:"Name"`
	ResourceLocations []map[string]float64 `json:"ResourceLocations"`
	TreeLocations     []map[string]float64 `json:"TreeLocations"`
	BiomeType         int                  `json:"BiomeType"`
}

// CubeConnection represents a cube and its connections (joints and connected cubes).
type CubeConnection struct {
	CubeName string      // Name of the cube
	Joints   []JointInfo // List of joints involving this cube
}

// JointInfo represents a joint and the cubes it connects.
type JointInfo struct {
	JointName     string // Name of the joint
	ConnectedCube string // The other cube connected by this joint (may be empty if unknown)
}

// --- CONSTRUCTOR ---

func NewSparseScanner(hosts []string, startPort int) *SparseScanner {
	return &SparseScanner{
		Hosts:      hosts,
		StartPort:  startPort,
		PortStep:   portStep,
		NumPods:    numPods,
		AuthPass:   authPass,
		EndMarker:  endMarker,
		TimeoutSec: timeoutSec,
		PlanetsMap: make(map[string]PlanetRecord),
		CubesMap:   make(map[string]string),
	}
}

func (s *SparseScanner) InitSparseScanner(hosts []string, startPort int) {
	s.Hosts = hosts
	s.StartPort = startPort
	s.PortStep = portStep
	s.NumPods = numPods
	s.AuthPass = authPass
	s.EndMarker = endMarker
	s.TimeoutSec = timeoutSec
	s.PlanetsMap = make(map[string]PlanetRecord)
	s.CubesMap = make(map[string]string)
}

// --- MAIN METHODS ---

func (s *SparseScanner) ScanAllPods() {
	startTime := time.Now()
	var wg sync.WaitGroup
	resultsChan := make(chan PodResult, s.NumPods*len(s.Hosts))

	for _, host := range s.Hosts {
		for i := 0; i < s.NumPods; i++ {
			port := s.StartPort + i*s.PortStep
			wg.Add(1)
			go func(host string, port int) {
				defer wg.Done()
				result := s.checkPod(host, port)
				resultsChan <- result
			}(host, port)
		}
	}

	wg.Wait()
	close(resultsChan)

	for result := range resultsChan {
		s.Results = append(s.Results, result)
	}

	s.processResults()

	fmt.Printf("\n🌌 Discovery complete in %s\n", time.Since(startTime))
}

func (s *SparseScanner) processResults() {
	for _, result := range s.Results {
		if !result.Success {
			continue
		}
		for _, planet := range result.Planets {
			coords := [3]float64{
				planet.Position["x"],
				planet.Position["y"],
				planet.Position["z"],
			}
			s.PlanetsMap[planet.Name] = PlanetRecord{
				Name:        planet.Name,
				Coordinates: coords,
				Host:        result.Host,
				Port:        result.Port,
			}
		}
		for _, cube := range result.Cubes {
			s.CubesMap[cube] = result.Host
		}
	}
}

func (s *SparseScanner) PrintSummary() {
	totalCubes := 0
	totalPlanets := 0
	successCount := 0

	fmt.Println("\n=== MULTIVERSE SUMMARY ===")
	for _, res := range s.Results {
		if res.Success {
			successCount++
			totalCubes += len(res.Cubes)
			totalPlanets += len(res.Planets)
			fmt.Printf("[%s:%d] ✅ Connected: Cubes=%d Planets=%d\n", res.Host, res.Port, len(res.Cubes), len(res.Planets))
		} else {
			fmt.Printf("[%s:%d] ❌ Failed: %s\n", res.Host, res.Port, res.Error)
		}
	}

	fmt.Printf("\n✅ Successful pods: %d / %d\n", successCount, s.NumPods*len(s.Hosts))
	fmt.Printf("🧱 Total Cubes: %d\n", totalCubes)
	fmt.Printf("🪐 Total Planets: %d\n", totalPlanets)
	fmt.Printf("🔭 Total unique planets mapped: %d\n", len(s.PlanetsMap))
}

func (s *SparseScanner) ExtractPlanetCenters() [][]float64 {
	centers := [][]float64{}
	for _, planet := range s.PlanetsMap {
		centers = append(centers, []float64{
			planet.Coordinates[0],
			planet.Coordinates[1],
			planet.Coordinates[2],
		})
	}
	return centers
}

// --- INTERNAL HELPERS ---

func (s *SparseScanner) checkPod(host string, port int) PodResult {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(s.TimeoutSec)*time.Second)
	if err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: fmt.Sprintf("Failed to connect: %v", err)}
	}
	defer conn.Close()

	if err := send(conn, s.AuthPass); err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: fmt.Sprintf("Failed to send auth: %v", err)}
	}
	authResp := read(conn)
	if !strings.Contains(authResp, "auth_success") {
		return PodResult{Host: host, Port: port, Success: false, Error: fmt.Sprintf("Authentication failed: %s", authResp)}
	}

	if err := send(conn, `{"type":"get_cube_list"}`); err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: "Failed to request cubes"}
	}
	cubesRaw := read(conn)
	var cubeData map[string]interface{}
	if err := json.Unmarshal([]byte(cubesRaw), &cubeData); err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: "Failed to parse cube list"}
	}
	cubes := toStringArray(cubeData["cubes"])

	if err := send(conn, `{"type":"get_planets"}`); err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: "Failed to request planets"}
	}
	planetsRaw := read(conn)
	var planetData map[string][]Planet
	if err := json.Unmarshal([]byte(planetsRaw), &planetData); err != nil {
		return PodResult{Host: host, Port: port, Success: false, Error: "Failed to parse planet list"}
	}
	var allPlanets []Planet
	for _, ps := range planetData {
		allPlanets = append(allPlanets, ps...)
	}

	return PodResult{
		Host:    host,
		Port:    port,
		Success: true,
		Cubes:   cubes,
		Planets: allPlanets,
	}
}

// --- GLOBAL HELPERS ---

func send(conn net.Conn, msg string) error {
	_, err := conn.Write([]byte(msg + endMarker))
	return err
}

func read(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	conn.SetReadDeadline(time.Now().Add(timeoutSec * time.Second))
	var buf bytes.Buffer
	chunk := make([]byte, 1024)
	for {
		n, err := reader.Read(chunk)
		if err != nil && err != io.EOF {
			break
		}
		buf.Write(chunk[:n])
		if strings.HasSuffix(buf.String(), endMarker) {
			break
		}
		if err == io.EOF {
			break
		}
	}
	msg := buf.String()
	if len(msg) >= len(endMarker) && strings.HasSuffix(msg, endMarker) {
		return msg[:len(msg)-len(endMarker)]
	}
	return msg
}

func (s *SparseScanner) ScanSinglePod(host string, port int) PodResult {
	result := s.checkPod(host, port)
	if result.Success {
		for _, planet := range result.Planets {
			coords := [3]float64{planet.Position["x"], planet.Position["y"], planet.Position["z"]}
			s.PlanetsMap[planet.Name] = PlanetRecord{
				Name:        planet.Name,
				Coordinates: coords,
				Host:        result.Host,
				Port:        result.Port,
			}
		}
		for _, cube := range result.Cubes {
			s.CubesMap[cube] = result.Host
		}
	}
	return result // Do not append to s.Results here
}

func (s *SparseScanner) AddPodResult(result PodResult) {
	s.Results = append(s.Results, result)
	if result.Success {
		for _, planet := range result.Planets {
			coords := [3]float64{planet.Position["x"], planet.Position["y"], planet.Position["z"]}
			s.PlanetsMap[planet.Name] = PlanetRecord{
				Name:        planet.Name,
				Coordinates: coords,
				Host:        result.Host,
				Port:        result.Port,
			}
		}
		for _, cube := range result.Cubes {
			s.CubesMap[cube] = result.Host
		}
	}
}

// GetCubesByPrefix returns a list of cube names that start with the given prefix.
func (s *SparseScanner) GetCubesByPrefix(prefix string) []string {
	filteredCubes := []string{}
	for cubeName := range s.CubesMap {
		if strings.HasPrefix(cubeName, prefix) {
			filteredCubes = append(filteredCubes, cubeName)
		}
	}
	return filteredCubes
}

// GetCubesAndConnections retrieves all cubes starting with the given prefix and their connections.
func (s *SparseScanner) GetCubesAndConnections(prefix string) ([]CubeConnection, error) {
	// Step 1: Get all cubes matching the prefix
	cubes := s.GetCubesByPrefix(prefix)
	if len(cubes) == 0 {
		return nil, fmt.Errorf("no cubes found with prefix %s", prefix)
	}

	// Step 2: For each cube, get its joints and attempt to find connections
	result := make([]CubeConnection, 0, len(cubes))
	for _, cube := range cubes {
		// Get joints for this cube using the existing function
		joints := getJointsForCube(cube)

		// Prepare the list of joint information
		jointInfos := make([]JointInfo, 0, len(joints))

		// Lock globalCubeLinks to safely read it
		linkListMutex.Lock()
		for _, jointName := range joints {
			// Try to find this joint in globalCubeLinks to identify the connected cube
			connectedCube := ""
			for _, link := range globalCubeLinks {
				if link.JointName == jointName {
					// Determine which cube in the link is the "other" cube
					if link.CubeA == cube {
						connectedCube = link.CubeB
					} else if link.CubeB == cube {
						connectedCube = link.CubeA
					}
					break
				}
			}
			jointInfos = append(jointInfos, JointInfo{
				JointName:     jointName,
				ConnectedCube: connectedCube,
			})
		}
		linkListMutex.Unlock()

		// Add to result
		result = append(result, CubeConnection{
			CubeName: cube,
			Joints:   jointInfos,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no connections found for cubes with prefix %s", prefix)
	}

	return result, nil
}

// GetCubesAndConnectionsParallel retrieves all cubes starting with the given prefix and their connections,
// using a thread pool to parallelize the requests.
func (s *SparseScanner) GetCubesAndConnectionsParallel(prefix string) ([]CubeConnection, error) {
	// Step 1: Get all cubes matching the prefix
	cubes := s.GetCubesByPrefix(prefix)
	if len(cubes) == 0 {
		return nil, fmt.Errorf("no cubes found with prefix %s", prefix)
	}

	// Step 2: Set up thread pool parameters
	const maxWorkers = 10                                // Maximum number of concurrent requests to the server
	sem := make(chan struct{}, maxWorkers)               // Semaphore to limit concurrency
	var wg sync.WaitGroup                                // To wait for all goroutines to complete
	resultsChan := make(chan CubeConnection, len(cubes)) // Channel to collect results

	// Step 3: Process each cube in parallel
	for _, cube := range cubes {
		wg.Add(1)
		sem <- struct{}{} // Acquire a slot in the thread pool
		go func(cubeName string) {
			defer wg.Done()
			defer func() { <-sem }() // Release the slot when done

			// Get joints for this cube
			joints := getJointsForCube(cubeName)

			// Prepare the list of joint information
			jointInfos := make([]JointInfo, 0, len(joints))

			// Lock globalCubeLinks to safely read it
			linkListMutex.Lock()
			for _, jointName := range joints {
				// Try to find this joint in globalCubeLinks to identify the connected cube
				connectedCube := ""
				for _, link := range globalCubeLinks {
					if link.JointName == jointName {
						if link.CubeA == cubeName {
							connectedCube = link.CubeB
						} else if link.CubeB == cubeName {
							connectedCube = link.CubeA
						}
						break
					}
				}
				jointInfos = append(jointInfos, JointInfo{
					JointName:     jointName,
					ConnectedCube: connectedCube,
				})
			}
			linkListMutex.Unlock()

			// Send the result to the channel
			resultsChan <- CubeConnection{
				CubeName: cubeName,
				Joints:   jointInfos,
			}
		}(cube)
	}

	// Step 4: Wait for all goroutines to complete and close the results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Step 5: Collect results into a slice
	result := make([]CubeConnection, 0, len(cubes))
	for conn := range resultsChan {
		result = append(result, conn)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no connections found for cubes with prefix %s", prefix)
	}

	// Step 6: Sort the results by CubeName for consistency (optional)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CubeName < result[j].CubeName
	})

	return result, nil
}
