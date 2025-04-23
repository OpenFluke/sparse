package main

import (
	"fmt"

	paragon "github.com/OpenFluke/PARAGON"
)

type ExperimentModel struct {
	Ip            string
	Port          int
	Cubes         int
	Planets       int
	ExpectedCubes int
	Template      Construct
	Model         *paragon.Network
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

	for num, res := range scannerTmp.Results {
		if res.Success {

			tmp := *NewConstruct(res.Host+":"+string(res.Port), aPass, aDel)

			// Load the JSON string for validation/storage
			if err := tmp.LoadJSONToString(jsonStr); err != nil {
				fmt.Printf("Error loading JSON string: %v\n", err)
				return
			}

			// Generate a unique unitName for this Construct (e.g., "POD_192.168.0.227_10008")
			unitNameIp := fmt.Sprintf("POD_%s_%d", res.Host, res.Port)
			unitName := generateUnitID("ARC", "openfluke.com",
				1, num) + "-" + unitNameIp

			// Load the JSON string and parse it into tmp.Config
			if err := tmp.LoadConfigFromJSONString(jsonStr, unitName); err != nil {
				fmt.Printf("Error loading JSON string for %s: %v\n", unitName, err)
				return
			}

			fmt.Println(unitName + " expecting " + fmt.Sprintf("%d", len(tmp.Config.Cubes)) + " cubes")

			expModel := ExperimentModel{
				Ip:            res.Host, // Join the list of IPs into a single string
				Port:          res.Port,
				Cubes:         len(res.Cubes),
				Planets:       len(res.Planets),
				Template:      tmp,
				ExpectedCubes: len(tmp.Config.Cubes),
			}

			ExperimentModels = append(ExperimentModels, expModel)

		}
	}

}
