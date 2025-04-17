package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

func targetedDespawnAllCubes(unitName string) {
	var wg sync.WaitGroup
	for _, cube := range globalCubeList {
		if strings.HasPrefix(cube, unitName+"_") {
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				conn, err := net.Dial("tcp", serverAddr)
				if err != nil {
					fmt.Printf("[%s] [Despawn] Connection failed: %v\n", unitName, err)
					return
				}
				defer conn.Close()

				if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
					return
				}
				_, _ = readResponse(conn)

				despawn := Message{
					"type":      "despawn_cube",
					"cube_name": name,
				}
				sendJSONMessage(conn, despawn)
			}(cube)
		}
	}
	wg.Wait()
	fmt.Printf("🧹 [%s] All cubes despawned.\n", unitName)
}

func despawnAllCubes() {
	var wg sync.WaitGroup
	for _, cube := range globalCubeList {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				fmt.Println("[Despawn] Failed to connect:", err)
				return
			}
			defer conn.Close()

			if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
				return
			}
			_, _ = readResponse(conn)

			despawn := Message{
				"type":      "despawn_cube",
				"cube_name": name,
			}
			sendJSONMessage(conn, despawn)
		}(cube)
	}
	wg.Wait()
}

// nukeAllCubes asks the server for ALL active cubes and despawns them brutally.
func nukeAllCubes() {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("[Nuke] Failed to connect:", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(authPass + delimiter)); err != nil {
		fmt.Println("[Nuke] Failed to auth:", err)
		return
	}
	_, _ = readResponse(conn)

	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Request all cubes
		if err := sendJSONMessage(conn, Message{"type": "get_cube_list"}); err != nil {
			fmt.Println("[Nuke] Failed to request cube list:", err)
			return
		}
		raw, err := readResponse(conn)
		if err != nil {
			fmt.Println("[Nuke] Failed to read cube list:", err)
			return
		}

		var cubeData map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &cubeData); err != nil {
			fmt.Println("[Nuke] JSON unmarshal error:", err)
			return
		}

		cubes := toStringArray(cubeData["cubes"])
		if len(cubes) == 0 {
			fmt.Println("[Nuke] All cubes cleared.")
			break
		}

		for _, cube := range cubes {
			if err := sendJSONMessage(conn, Message{
				"type":      "despawn_cube",
				"cube_name": cube,
			}); err != nil {
				fmt.Printf("[Nuke] Failed to despawn cube %s: %v\n", cube, err)
			}
		}

		fmt.Printf("[Nuke] NUKED %d cubes (pass %d)\n", len(cubes), attempt)
		time.Sleep(500 * time.Millisecond) // Give server time to process
	}

	fmt.Println("[Nuke] Finished.")
}
