package market

func CalculatePricePerHour(gpuCount int, gpuModel string) float64 {
	prices := map[string]float64{
		"A100": 8.5,
		"H100": 15.0,
		"V100": 5.0,
	}
	price, exists := prices[gpuModel]
	if !exists {
		price = 10.0
	}
	return price * float64(gpuCount)
}

func ValidateGPUCount(count int) bool {
	return count >= 1 && count <= 64
}
