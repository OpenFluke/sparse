package main

import (
	"fmt"
	"sort"
	"strings"
)

// GetChainsJointTable returns the chains, joint type, and joint parameters as a table.
// Columns are: item1, item2, jointtype, followed by all joint parameters as key:value pairs.
// Each row is an array of strings, e.g., ["head", "body", "hinge", "limit_upper:0.0", ...].
func (c *Construct) GetChainsJointTable() [][]string {
	// First, count the total number of pairs in all chains
	totalPairs := 0
	for _, chain := range c.Config.Chains {
		if len(chain) > 1 {
			totalPairs += len(chain) - 1 // Number of pairs is len(chain) - 1
		}
	}

	// If no pairs, return an empty slice
	if totalPairs == 0 {
		return [][]string{}
	}

	// Sort joint parameter keys for consistent ordering
	paramKeys := make([]string, 0, len(c.Config.JointParams))
	for key := range c.Config.JointParams {
		paramKeys = append(paramKeys, key)
	}
	sort.Strings(paramKeys)

	// Create the rows
	rows := make([][]string, 0, totalPairs)
	for _, chain := range c.Config.Chains {
		// Skip chains with fewer than 2 elements
		if len(chain) < 2 {
			continue
		}
		// Generate pairs from the chain
		for i := 0; i < len(chain)-1; i++ {
			item1 := chain[i]
			item2 := chain[i+1]

			// Start the row with item1, item2, and jointtype
			row := []string{item1, item2, c.Config.JointType}

			// Append all joint parameters in sorted order
			for _, key := range paramKeys {
				value := c.Config.JointParams[key]
				row = append(row, fmt.Sprintf("%s:%g", key, value))
			}

			rows = append(rows, row)
		}
	}

	return rows
}

// GetCubesTable returns the cubes data as a table with columns: name, x, y, z, rx, ry, rz.
// Each row is an array of strings, e.g., ["head", "0", "3.6", "0", "0", "0", "0"].
func (c *Construct) GetCubesTable() [][]string {
	rows := make([][]string, len(c.Config.Cubes))
	for i, cube := range c.Config.Cubes {
		// Extract position components (x, y, z)
		x := "0"
		y := "0"
		z := "0"
		if len(cube.Position) == 3 {
			x = fmt.Sprintf("%g", cube.Position[0])
			y = fmt.Sprintf("%g", cube.Position[1])
			z = fmt.Sprintf("%g", cube.Position[2])
		}

		// Default rotation to [0,0,0] since it's not stored in Cube
		rx := "0"
		ry := "0"
		rz := "0"

		// Create the row with separate columns for x, y, z, rx, ry, rz
		rows[i] = []string{cube.Name, x, y, z, rx, ry, rz}
	}
	return rows
}

// PrintCubesTable prints the cubes data as a table with columns: name, x, y, z, rx, ry, rz.
// Each row is printed as a comma-separated string, e.g., "head,0,3.6,0,0,0,0".
func (c *Construct) PrintCubesTable() {
	fmt.Printf("Cubes Table for %s (name,x,y,z,rx,ry,rz):\n", c.unitName)
	for _, cube := range c.Config.Cubes {
		// Extract position components (x, y, z)
		x := "0"
		y := "0"
		z := "0"
		if len(cube.Position) == 3 {
			x = fmt.Sprintf("%g", cube.Position[0])
			y = fmt.Sprintf("%g", cube.Position[1])
			z = fmt.Sprintf("%g", cube.Position[2])
		}

		// Default rotation to [0,0,0] since it's not stored in Cube
		rx := "0"
		ry := "0"
		rz := "0"

		// Create the row and print it
		row := []string{cube.Name, x, y, z, rx, ry, rz}
		fmt.Println(strings.Join(row, ","))
	}
}

// PrintChainsJointTable prints the chains, joint type, and joint parameters as a table.
// Columns are: item1, item2, jointtype, followed by all joint parameters as key:value pairs.
// Each row is printed as a comma-separated string, e.g., "head,body,hinge,limit_upper:0.0,...".
func (c *Construct) PrintChainsJointTable() {
	// Count the total number of pairs in all chains
	totalPairs := 0
	for _, chain := range c.Config.Chains {
		if len(chain) > 1 {
			totalPairs += len(chain) - 1
		}
	}

	// If no pairs, print a message and return
	if totalPairs == 0 {
		fmt.Printf("Chains/Joint Table for %s: No chains with pairs to display.\n", c.unitName)
		return
	}

	// Sort joint parameter keys for consistent ordering
	paramKeys := make([]string, 0, len(c.Config.JointParams))
	for key := range c.Config.JointParams {
		paramKeys = append(paramKeys, key)
	}
	sort.Strings(paramKeys)

	fmt.Printf("Chains/Joint Table for %s (item1,item2,jointtype,jointparams):\n", c.unitName)
	for _, chain := range c.Config.Chains {
		// Skip chains with fewer than 2 elements
		if len(chain) < 2 {
			continue
		}
		// Generate pairs from the chain
		for i := 0; i < len(chain)-1; i++ {
			item1 := chain[i]
			item2 := chain[i+1]

			// Start the row with item1, item2, and jointtype
			row := []string{item1, item2, c.Config.JointType}

			// Append all joint parameters in sorted order
			for _, key := range paramKeys {
				value := c.Config.JointParams[key]
				row = append(row, fmt.Sprintf("%s:%g", key, value))
			}

			// Print the row
			fmt.Println(strings.Join(row, ","))
		}
	}
}
