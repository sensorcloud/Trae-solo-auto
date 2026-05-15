package agent

func GetActiveAgents() []uint {
	var ids []uint
	for id := range sandboxInstances {
		ids = append(ids, id)
	}
	return ids
}