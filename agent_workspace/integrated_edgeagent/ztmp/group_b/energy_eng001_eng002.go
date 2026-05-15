package energy

func CalculateCarbonIntensity(powerSource string) int {
	switch powerSource {
	case "solar":
		return 15
	case "wind":
		return 12
	case "gas":
		return 450
	case "storage":
		return 0
	default:
		return 100
	}
}

func IsValidPowerSource(source string) bool {
	validSources := []string{"solar", "wind", "gas", "storage"}
	for _, s := range validSources {
		if s == source {
			return true
		}
	}
	return false
}
