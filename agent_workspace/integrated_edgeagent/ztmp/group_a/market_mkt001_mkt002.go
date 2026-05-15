package market

func CalculatePricePerHour(gpuCount int, gpuModel string) float64 {
	var basePrice float64
	switch gpuModel {
	case "A100":
		basePrice = 8.5
	case "H100":
		basePrice = 15.0
	case "V100":
		basePrice = 5.0
	default:
		basePrice = 10.0
	}
	return basePrice * float64(gpuCount)
}

func ValidateGPUCount(count int) bool {
	if count < 1 || count > 64 {
		return false
	}
	return true
}
