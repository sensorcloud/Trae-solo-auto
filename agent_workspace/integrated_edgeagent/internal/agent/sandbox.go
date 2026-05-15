package agent

import (
	"fmt"
	"os/exec"
	"time"
)

type SandboxInstance struct {
	AgentID  uint
	Process  *exec.Cmd
	Port     int
	StartedAt time.Time
}

var sandboxInstances = make(map[uint]*SandboxInstance)

func startAgentSandbox(agent *Agent) error {
	if _, exists := sandboxInstances[agent.ID]; exists {
		return fmt.Errorf("Agent already running")
	}

	var cmd *exec.Cmd
	switch agent.Runtime {
	case "python":
		cmd = exec.Command("python", "-c", agent.Code)
	case "node":
		cmd = exec.Command("node", "-e", agent.Code)
	default:
		return fmt.Errorf("unsupported runtime: %s", agent.Runtime)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	sandboxInstances[agent.ID] = &SandboxInstance{
		AgentID:  agent.ID,
		Process:  cmd,
		StartedAt: time.Now(),
	}

	go monitorSandbox(agent.ID)

	return nil
}

func stopAgentSandbox(agent *Agent) error {
	instance, exists := sandboxInstances[agent.ID]
	if !exists {
		return fmt.Errorf("Agent not running")
	}

	if err := instance.Process.Process.Kill(); err != nil {
		return err
	}

	instance.Process.Wait()
	delete(sandboxInstances, agent.ID)

	return nil
}

func monitorSandbox(agentID uint) {
	instance := sandboxInstances[agentID]
	if instance == nil {
		return
	}

	instance.Process.Wait()

	delete(sandboxInstances, agentID)
}

func executeAgent(agent *Agent, req ExecuteRequest) (*ExecuteResponse, error) {
	_, exists := sandboxInstances[agent.ID]
	if !exists {
		return nil, fmt.Errorf("Agent not running")
	}

	result := &ExecuteResponse{
		Output:   "Execution completed",
		Status:   "success",
		Context:  req.Context,
		ToolCalls: req.ToolCalls,
	}

	return result, nil
}