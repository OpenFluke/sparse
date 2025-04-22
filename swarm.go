package main

import "fmt"

type ExperimentModel struct {
	Ip            string
	Port          int
	Cubes         int
	Planets       int
	ExpectedCubes int
	Template      Construct
}

var ExperimentModels []ExperimentModel // Public array to store experiment models

// tmpSweep scans the multiverse and returns the total number of detected cubes.
func QuickScan(quick []string, port int) int {
	scannerTmp := &SparseScanner{}
	scannerTmp.InitSparseScanner(quick, port) // starting port
	scannerTmp.ScanAllPods()
	scannerTmp.PrintSummary()

	// Calculate total cubes detected
	totalCubes := 0
	for _, res := range scannerTmp.Results {
		if res.Success {
			totalCubes += len(res.Cubes)
		}
	}
	return totalCubes
}

func StartEMLst(quick []string, port int, aPass string, aDel string) {
	scannerTmp := &SparseScanner{}
	scannerTmp.InitSparseScanner(quick, port) // starting port
	scannerTmp.ScanAllPods()
	//scannerTmp.PrintSummary()

	// Load JSON from a file into a string
	jsonStr, err := LoadJSONFileToString("construct_config.json")
	if err != nil {
		fmt.Printf("Error loading JSON file: %v\n", err)
		return
	}
	fmt.Printf("JSON string loaded:\n%s\n", jsonStr)

	for _, res := range scannerTmp.Results {
		if res.Success {

			tmp := *NewConstruct(res.Host+":"+string(res.Port), aPass, aDel)

			// Load the JSON string for validation/storage
			if err := tmp.LoadJSONToString(jsonStr); err != nil {
				fmt.Printf("Error loading JSON string: %v\n", err)
				return
			}

			expModel := ExperimentModel{
				Ip:       res.Host, // Join the list of IPs into a single string
				Port:     res.Port,
				Cubes:    len(res.Cubes),
				Planets:  len(res.Planets),
				Template: tmp,
			}

			ExperimentModels = append(ExperimentModels, expModel)

		}
	}

}
