package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RuntimeManager struct {
	mu         sync.RWMutex
	runtimes   map[RuntimeType]*RuntimeInfo
	configPath string
	
	defaultRuntime RuntimeType
}

func NewRuntimeManager(configPath string) *RuntimeManager {
	return &RuntimeManager{
		runtimes:       make(map[RuntimeType]*RuntimeInfo),
		configPath:     configPath,
		defaultRuntime: RuntimeTypeRunsc,
	}
}

func (rm *RuntimeManager) Initialize(ctx context.Context) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if err := rm.detectRuntimes(ctx); err != nil {
		return fmt.Errorf("检测运行时失败: %w", err)
	}
	
	if err := rm.loadRuntimeConfig(); err != nil {
		return fmt.Errorf("加载运行时配置失败: %w", err)
	}
	
	return nil
}

func (rm *RuntimeManager) detectRuntimes(ctx context.Context) error {
	runtimePaths := map[RuntimeType][]string{
		RuntimeTypeRunsc: {"/usr/local/bin/runsc", "/usr/bin/runsc", "runsc"},
		RuntimeTypeKata:  {"/usr/local/bin/kata-runtime", "/usr/bin/kata-runtime", "kata-runtime"},
		RuntimeTypeRunc:  {"/usr/local/bin/runc", "/usr/bin/runc", "runc"},
	}
	
	for runtimeType, paths := range runtimePaths {
		for _, path := range paths {
			info, err := rm.probeRuntime(ctx, runtimeType, path)
			if err != nil {
				continue
			}
			
			rm.runtimes[runtimeType] = info
			break
		}
	}
	
	if len(rm.runtimes) == 0 {
		return fmt.Errorf("未找到任何可用的容器运行时")
	}
	
	return nil
}

func (rm *RuntimeManager) probeRuntime(ctx context.Context, runtimeType RuntimeType, path string) (*RuntimeInfo, error) {
	absPath, err := exec.LookPath(path)
	if err != nil {
		return nil, fmt.Errorf("查找运行时路径失败: %w", err)
	}
	
	version, err := rm.getRuntimeVersion(ctx, absPath)
	if err != nil {
		return nil, fmt.Errorf("获取运行时版本失败: %w", err)
	}
	
	info := &RuntimeInfo{
		Type:    runtimeType,
		Path:    absPath,
		Version: version,
		Status:  RuntimeStatusAvailable,
	}
	
	switch runtimeType {
	case RuntimeTypeRunsc:
		info.Capabilities = rm.getRunscCapabilities()
		info.Platform = "gvisor"
	case RuntimeTypeKata:
		info.Capabilities = rm.getKataCapabilities()
		info.Platform = "kata-containers"
	case RuntimeTypeRunc:
		info.Capabilities = rm.getRuncCapabilities()
		info.Platform = "linux"
	}
	
	return info, nil
}

func (rm *RuntimeManager) getRuntimeVersion(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	
	return "unknown", nil
}

func (rm *RuntimeManager) getRunscCapabilities() []string {
	return []string{
		"seccomp",
		"apparmor",
		"network-namespace",
		"pid-namespace",
		"ipc-namespace",
		"uts-namespace",
		"user-namespace",
		"rootless",
		"overlayfs",
		"secure-execution",
	}
}

func (rm *RuntimeManager) getKataCapabilities() []string {
	return []string{
		"seccomp",
		"apparmor",
		"network-namespace",
		"pid-namespace",
		"ipc-namespace",
		"uts-namespace",
		"hardware-virtualization",
		"secure-execution",
		"hypervisor",
	}
}

func (rm *RuntimeManager) getRuncCapabilities() []string {
	return []string{
		"seccomp",
		"apparmor",
		"network-namespace",
		"pid-namespace",
		"ipc-namespace",
		"uts-namespace",
		"user-namespace",
		"rootless",
		"cgroup",
	}
}

func (rm *RuntimeManager) loadRuntimeConfig() error {
	if rm.configPath == "" {
		return nil
	}
	
	data, err := os.ReadFile(rm.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	var configs map[string]RuntimeConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return err
	}
	
	for name, config := range configs {
		rt := RuntimeType(name)
		if info, ok := rm.runtimes[rt]; ok {
			info.Config = config
		}
	}
	
	return nil
}

func (rm *RuntimeManager) GetRuntime(runtimeType RuntimeType) (*RuntimeInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	info, ok := rm.runtimes[runtimeType]
	if !ok {
		return nil, fmt.Errorf("运行时 %s 不可用", runtimeType)
	}
	
	return info, nil
}

func (rm *RuntimeManager) GetAvailableRuntimes() []RuntimeType {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	runtimes := make([]RuntimeType, 0, len(rm.runtimes))
	for rt := range rm.runtimes {
		runtimes = append(runtimes, rt)
	}
	return runtimes
}

func (rm *RuntimeManager) SetDefaultRuntime(runtimeType RuntimeType) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, ok := rm.runtimes[runtimeType]; !ok {
		return fmt.Errorf("运行时 %s 不可用", runtimeType)
	}
	
	rm.defaultRuntime = runtimeType
	return nil
}

func (rm *RuntimeManager) GetDefaultRuntime() RuntimeType {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.defaultRuntime
}

type ContainerSpec struct {
	ID          string
	Name        string
	Image       string
	Runtime     RuntimeType
	
	Command     []string
	Args        []string
	Env         map[string]string
	WorkingDir  string
	
	Resources   ResourceSpec
	Network     NetworkConfig
	Security    SecurityConfig
	
	Rootfs      string
	RootfsType  string
	
	Stdin       bool
	Tty         bool
	
	Labels      map[string]string
	Annotations map[string]string
}

type ContainerStatus struct {
	ID          string
	Name        string
	Status      string
	Runtime     RuntimeType
	Pid         int
	ExitCode    int
	
	CreatedAt   time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
	
	IPAddress   string
	MacAddress  string
	
	Metrics     ContainerMetrics
}

type ContainerMetrics struct {
	CPUUsage    float64
	MemoryUsage int64
	MemoryMax   int64
	NetworkIn   int64
	NetworkOut  int64
	BlockRead   int64
	BlockWrite  int64
}

type ContainerManager struct {
	runtimeManager *RuntimeManager
	stateDir       string
	rootDir        string
	
	mu         sync.RWMutex
	containers map[string]*ContainerStatus
}

func NewContainerManager(runtimeManager *RuntimeManager, stateDir, rootDir string) *ContainerManager {
	return &ContainerManager{
		runtimeManager: runtimeManager,
		stateDir:       stateDir,
		rootDir:        rootDir,
		containers:     make(map[string]*ContainerStatus),
	}
}

func (cm *ContainerManager) Initialize(ctx context.Context) error {
	if err := os.MkdirAll(cm.stateDir, 0755); err != nil {
		return fmt.Errorf("创建状态目录失败: %w", err)
	}
	if err := os.MkdirAll(cm.rootDir, 0755); err != nil {
		return fmt.Errorf("创建根目录失败: %w", err)
	}
	
	if err := cm.restoreContainers(ctx); err != nil {
		return fmt.Errorf("恢复容器状态失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) restoreContainers(ctx context.Context) error {
	entries, err := os.ReadDir(cm.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		containerID := entry.Name()
		stateFile := filepath.Join(cm.stateDir, containerID, "state.json")
		
		data, err := os.ReadFile(stateFile)
		if err != nil {
			continue
		}
		
		var status ContainerStatus
		if err := json.Unmarshal(data, &status); err != nil {
			continue
		}
		
		cm.containers[containerID] = &status
	}
	
	return nil
}

func (cm *ContainerManager) CreateContainer(ctx context.Context, spec *ContainerSpec) (*ContainerStatus, error) {
	runtimeInfo, err := cm.runtimeManager.GetRuntime(spec.Runtime)
	if err != nil {
		return nil, fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	containerID := spec.ID
	if containerID == "" {
		containerID = generateContainerID()
	}
	
	containerDir := filepath.Join(cm.stateDir, containerID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return nil, fmt.Errorf("创建容器目录失败: %w", err)
	}
	
	rootfsDir := filepath.Join(cm.rootDir, containerID, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建rootfs目录失败: %w", err)
	}
	
	bundleDir := filepath.Join(cm.rootDir, containerID)
	if err := cm.createBundle(ctx, bundleDir, spec); err != nil {
		return nil, fmt.Errorf("创建bundle失败: %w", err)
	}
	
	status := &ContainerStatus{
		ID:          containerID,
		Name:        spec.Name,
		Status:      "created",
		Runtime:     spec.Runtime,
		CreatedAt:   time.Now(),
	}
	
	switch spec.Runtime {
	case RuntimeTypeRunsc:
		err = cm.createRunscContainer(ctx, runtimeInfo.Path, containerID, bundleDir, spec)
	case RuntimeTypeKata:
		err = cm.createKataContainer(ctx, runtimeInfo.Path, containerID, bundleDir, spec)
	case RuntimeTypeRunc:
		err = cm.createRuncContainer(ctx, runtimeInfo.Path, containerID, bundleDir, spec)
	default:
		err = fmt.Errorf("不支持的运行时类型: %s", spec.Runtime)
	}
	
	if err != nil {
		os.RemoveAll(containerDir)
		os.RemoveAll(filepath.Join(cm.rootDir, containerID))
		return nil, fmt.Errorf("创建容器失败: %w", err)
	}
	
	cm.mu.Lock()
	cm.containers[containerID] = status
	cm.mu.Unlock()
	
	if err := cm.saveContainerState(containerID, status); err != nil {
		return nil, fmt.Errorf("保存容器状态失败: %w", err)
	}
	
	return status, nil
}

func (cm *ContainerManager) createBundle(ctx context.Context, bundleDir string, spec *ContainerSpec) error {
	config := cm.buildRuntimeSpec(spec)
	
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	
	configPath := filepath.Join(bundleDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) buildRuntimeSpec(spec *ContainerSpec) map[string]interface{} {
	config := map[string]interface{}{
		"ociVersion": "1.0.2",
		"process": map[string]interface{}{
			"terminal": spec.Tty,
			"user": map[string]interface{}{
				"uid": 0,
				"gid": 0,
			},
			"args":    append(spec.Command, spec.Args...),
			"env":     cm.buildEnv(spec.Env),
			"cwd":     spec.WorkingDir,
			"capabilities": map[string]interface{}{
				"bounding":    []string{},
				"effective":   []string{},
				"inheritable": []string{},
				"permitted":   []string{},
				"ambient":     []string{},
			},
			"noNewPrivileges": spec.Security.NoNewPrivileges,
		},
		"root": map[string]interface{}{
			"path":     "rootfs",
			"readonly": spec.Security.ReadOnlyRootFS,
		},
		"hostname": spec.Name,
		"mounts":   cm.buildMounts(spec),
		"linux": map[string]interface{}{
			"uidMappings": []map[string]interface{}{
				{"containerID": 0, "hostID": 0, "size": 1},
			},
			"gidMappings": []map[string]interface{}{
				{"containerID": 0, "hostID": 0, "size": 1},
			},
			"namespaces": cm.buildNamespaces(spec),
			"resources":  cm.buildResources(spec),
		},
	}
	
	return config
}

func (cm *ContainerManager) buildEnv(env map[string]string) []string {
	envList := []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"TERM=xterm",
	}
	
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	
	return envList
}

func (cm *ContainerManager) buildMounts(spec *ContainerSpec) []map[string]interface{} {
	mounts := []map[string]interface{}{
		{
			"destination": "/proc",
			"type":        "proc",
			"source":      "proc",
			"options":     []string{"nosuid", "noexec", "nodev"},
		},
		{
			"destination": "/dev",
			"type":        "tmpfs",
			"source":      "tmpfs",
			"options":     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
		},
		{
			"destination": "/dev/pts",
			"type":        "devpts",
			"source":      "devpts",
			"options":     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
		},
		{
			"destination": "/dev/shm",
			"type":        "tmpfs",
			"source":      "shm",
			"options":     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
		},
		{
			"destination": "/dev/mqueue",
			"type":        "mqueue",
			"source":      "mqueue",
			"options":     []string{"nosuid", "noexec", "nodev"},
		},
		{
			"destination": "/sys",
			"type":        "sysfs",
			"source":      "sysfs",
			"options":     []string{"nosuid", "noexec", "nodev", "ro"},
		},
	}
	
	return mounts
}

func (cm *ContainerManager) buildNamespaces(spec *ContainerSpec) []map[string]interface{} {
	namespaces := []map[string]interface{}{
		{"type": "pid"},
		{"type": "network"},
		{"type": "ipc"},
		{"type": "uts"},
	}
	
	if spec.Security.UserNamespace {
		namespaces = append(namespaces, map[string]interface{}{
			"type": "user",
		})
	}
	
	return namespaces
}

func (cm *ContainerManager) buildResources(spec *ContainerSpec) map[string]interface{} {
	resources := map[string]interface{}{
		"memory": map[string]interface{}{
			"limit": spec.Resources.MemoryLimit,
		},
		"cpu": map[string]interface{}{
			"shares": 1024,
			"quota":  100000,
			"period": 100000,
		},
		"pids": map[string]interface{}{
			"limit": spec.Resources.ProcessLimit,
		},
	}
	
	return resources
}

func (cm *ContainerManager) createRunscContainer(ctx context.Context, runtimePath, containerID, bundleDir string, spec *ContainerSpec) error {
	args := []string{
		"create",
		"--bundle", bundleDir,
		"--root", cm.stateDir,
		containerID,
	}
	
	if spec.Network.Policy == NetworkPolicyIsolated {
		args = append([]string{"--network=none"}, args...)
	}
	
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimePath, args...)
	cmd.Dir = bundleDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runsc create失败: %w, output: %s", err, string(output))
	}
	
	return nil
}

func (cm *ContainerManager) createKataContainer(ctx context.Context, runtimePath, containerID, bundleDir string, spec *ContainerSpec) error {
	args := []string{
		"create",
		"--bundle", bundleDir,
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimePath, args...)
	cmd.Dir = bundleDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kata-runtime create失败: %w, output: %s", err, string(output))
	}
	
	return nil
}

func (cm *ContainerManager) createRuncContainer(ctx context.Context, runtimePath, containerID, bundleDir string, spec *ContainerSpec) error {
	args := []string{
		"create",
		"--bundle", bundleDir,
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimePath, args...)
	cmd.Dir = bundleDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("runc create失败: %w, output: %s", err, string(output))
	}
	
	return nil
}

func (cm *ContainerManager) StartContainer(ctx context.Context, containerID string) error {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"start",
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("启动容器失败: %w, output: %s", err, string(output))
	}
	
	cm.mu.Lock()
	status.Status = "running"
	status.StartedAt = time.Now()
	cm.mu.Unlock()
	
	if err := cm.saveContainerState(containerID, status); err != nil {
		return fmt.Errorf("保存容器状态失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string, timeout int) error {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"kill",
		"--root", cm.stateDir,
		containerID,
		"SIGTERM",
	}
	
	killCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(killCtx, runtimeInfo.Path, args...)
	_ = cmd.Run()
	
	time.Sleep(time.Duration(timeout) * time.Second)
	
	cm.mu.Lock()
	status.Status = "stopped"
	status.FinishedAt = time.Now()
	cm.mu.Unlock()
	
	if err := cm.saveContainerState(containerID, status); err != nil {
		return fmt.Errorf("保存容器状态失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) DeleteContainer(ctx context.Context, containerID string) error {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"delete",
		"--root", cm.stateDir,
		"--force",
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除容器失败: %w, output: %s", err, string(output))
	}
	
	containerDir := filepath.Join(cm.stateDir, containerID)
	os.RemoveAll(containerDir)
	
	rootfsDir := filepath.Join(cm.rootDir, containerID)
	os.RemoveAll(rootfsDir)
	
	cm.mu.Lock()
	delete(cm.containers, containerID)
	cm.mu.Unlock()
	
	return nil
}

func (cm *ContainerManager) GetContainerStatus(ctx context.Context, containerID string) (*ContainerStatus, error) {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return nil, fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"state",
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取容器状态失败: %w", err)
	}
	
	var state struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Pid    int    `json:"pid"`
	}
	
	if err := json.Unmarshal(output, &state); err != nil {
		return nil, fmt.Errorf("解析容器状态失败: %w", err)
	}
	
	cm.mu.Lock()
	status.Status = state.Status
	status.Pid = state.Pid
	cm.mu.Unlock()
	
	return status, nil
}

func (cm *ContainerManager) ExecInContainer(ctx context.Context, containerID string, cmd []string, stdin []byte) ([]byte, error) {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return nil, fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"exec",
		"--root", cm.stateDir,
		"-d",
		containerID,
	}
	args = append(args, cmd...)
	
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	
	execCmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	if len(stdin) > 0 {
		execCmd.Stdin = strings.NewReader(string(stdin))
	}
	
	output, err := execCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("容器执行失败: %w, output: %s", err, string(output))
	}
	
	return output, nil
}

func (cm *ContainerManager) PauseContainer(ctx context.Context, containerID string) error {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"pause",
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("暂停容器失败: %w, output: %s", err, string(output))
	}
	
	cm.mu.Lock()
	status.Status = "paused"
	cm.mu.Unlock()
	
	if err := cm.saveContainerState(containerID, status); err != nil {
		return fmt.Errorf("保存容器状态失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) ResumeContainer(ctx context.Context, containerID string) error {
	cm.mu.RLock()
	status, ok := cm.containers[containerID]
	cm.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("容器 %s 不存在", containerID)
	}
	
	runtimeInfo, err := cm.runtimeManager.GetRuntime(status.Runtime)
	if err != nil {
		return fmt.Errorf("获取运行时信息失败: %w", err)
	}
	
	args := []string{
		"resume",
		"--root", cm.stateDir,
		containerID,
	}
	
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, runtimeInfo.Path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("恢复容器失败: %w, output: %s", err, string(output))
	}
	
	cm.mu.Lock()
	status.Status = "running"
	cm.mu.Unlock()
	
	if err := cm.saveContainerState(containerID, status); err != nil {
		return fmt.Errorf("保存容器状态失败: %w", err)
	}
	
	return nil
}

func (cm *ContainerManager) saveContainerState(containerID string, status *ContainerStatus) error {
	stateFile := filepath.Join(cm.stateDir, containerID, "state.json")
	
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(stateFile, data, 0644)
}

func (cm *ContainerManager) ListContainers() []*ContainerStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	containers := make([]*ContainerStatus, 0, len(cm.containers))
	for _, status := range cm.containers {
		containers = append(containers, status)
	}
	return containers
}

func generateContainerID() string {
	return fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
}
