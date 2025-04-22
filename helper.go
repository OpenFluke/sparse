package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

func calculateRotationOutward(center, position []float64) float64 {
	dx := position[0] - center[0]
	dz := position[2] - center[2]
	angle := math.Atan2(dz, dx) * (180.0 / math.Pi)
	return angle
}

func normalize(vec []float64) []float64 {
	mag := math.Sqrt(vec[0]*vec[0] + vec[1]*vec[1] + vec[2]*vec[2])
	if mag == 0 {
		return []float64{0, 1, 0}
	}
	return []float64{vec[0] / mag, vec[1] / mag, vec[2] / mag}
}

func appendUnitSafely(slice []string, unitName string) []string {
	allUnitsMutex.Lock()
	defer allUnitsMutex.Unlock()
	return append(slice, unitName)
}

func toStringArray(v interface{}) []string {
	arr := []string{}
	if v == nil {
		return arr
	}
	switch vv := v.(type) {
	case []interface{}:
		for _, item := range vv {
			if str, ok := item.(string); ok {
				arr = append(arr, str)
			}
		}
	}
	return arr
}

// fibonacciSphere generates n points evenly distributed on a sphere
func fibonacciSphere(n int, radius float64, center []float64) [][]float64 {
	points := make([][]float64, n)
	if n == 0 {
		return points
	}
	if n == 1 {
		// For n=1, place the point along the positive X-axis at the specified radius
		points[0] = []float64{
			center[0] + radius,
			center[1],
			center[2],
		}
		return points
	}

	phi := math.Pi * (3 - math.Sqrt(5)) // Golden angle in radians
	for i := 0; i < n; i++ {
		y := 1 - (float64(i)/float64(n-1))*2 // y goes from 1 to -1
		r := math.Sqrt(1 - y*y)              // radius at y
		theta := phi * float64(i)            // golden angle increment
		x := math.Cos(theta) * r
		z := math.Sin(theta) * r
		// Scale by radius and offset by center
		points[i] = []float64{
			center[0] + x*radius,
			center[1] + y*radius,
			center[2] + z*radius,
		}
	}
	return points
}

func generateUnitID(role string, domain string, gen int, version int) string {
	domainParts := strings.Split(domain, ".")
	projectCode := ""
	for _, part := range domainParts {
		if len(part) > 0 {
			projectCode += strings.ToUpper(string(part[0]))
		}
	}
	return fmt.Sprintf("[%s]-%s-gen%d-v%d", strings.ToUpper(role), projectCode, gen, version)
}

// LoadJSONFileToString reads a JSON file and returns its contents as a string after validating it.
func LoadJSONFileToString(filename string) (string, error) {
	// Read the file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read JSON file %s: %v", filename, err)
	}

	// Validate that the content is valid JSON by attempting to unmarshal it into a generic interface
	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return "", fmt.Errorf("failed to validate JSON in file %s: %v", filename, err)
	}

	// Return the JSON content as a string
	return string(data), nil
}
