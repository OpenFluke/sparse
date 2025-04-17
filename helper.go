package main

import "math"

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
