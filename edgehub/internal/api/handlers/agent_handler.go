package handlers

import (
	"github.com/edgehub/edgehub/internal/agent"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentHandler struct {
	agentSvc agent.AgentService
}

func NewAgentHandler(agentSvc agent.AgentService) *AgentHandler {
	return &AgentHandler{
		agentSvc: agentSvc,
	}
}

// ==================== 沙箱管理 API ====================

// ListSandboxes 获取沙箱列表
// @Summary 获取沙箱列表
// @Description 获取所有沙箱列表，支持分页和过滤
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param status query string false "沙箱状态 (pending, creating, running, paused, stopping, stopped, error, terminated)"
// @Param runtime query string false "运行时类型 (runsc, kata, runc)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/agents/sandboxes [get]
func (h *AgentHandler) ListSandboxes(c *gin.Context) {
	pagination := GetPagination(c)

	req := &agent.ListSandboxesRequest{
		TenantID: GetTenantID(c),
		Status:   agent.AgentStatus(c.Query("status")),
		Runtime:  agent.SandboxRuntime(c.Query("runtime")),
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	sandboxes, err := h.agentSvc.ListSandboxes(c.Request.Context(), req)
	if err != nil {
		InternalError(c, "获取沙箱列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, sandboxes, int64(len(sandboxes)), pagination.Page, pagination.PageSize)
}

// CreateSandbox 创建沙箱
// @Summary 创建沙箱
// @Description 创建新的沙箱环境
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param sandbox body CreateSandboxRequest true "沙箱配置"
// @Success 201 {object} Response
// @Router /api/v1/agents/sandboxes [post]
func (h *AgentHandler) CreateSandbox(c *gin.Context) {
	var req agent.CreateSandboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	req.TenantID = GetTenantID(c)

	sandbox, err := h.agentSvc.CreateSandbox(c.Request.Context(), &req)
	if err != nil {
		InternalError(c, "创建沙箱失败: "+err.Error())
		return
	}

	Created(c, sandbox)
}

// GetSandbox 获取沙箱详情
// @Summary 获取沙箱详情
// @Description 根据ID获取沙箱详细信息
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{id} [get]
func (h *AgentHandler) GetSandbox(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	sandbox, err := h.agentSvc.GetSandbox(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "沙箱不存在")
		return
	}

	Success(c, sandbox)
}

// PauseSandbox 暂停沙箱
// @Summary 暂停沙箱
// @Description 暂停运行中的沙箱
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{id}/pause [post]
func (h *AgentHandler) PauseSandbox(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	if err := h.agentSvc.PauseSandbox(c.Request.Context(), id); err != nil {
		InternalError(c, "暂停沙箱失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "沙箱已暂停", nil)
}

// ResumeSandbox 恢复沙箱
// @Summary 恢复沙箱
// @Description 恢复已暂停的沙箱
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{id}/resume [post]
func (h *AgentHandler) ResumeSandbox(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	if err := h.agentSvc.ResumeSandbox(c.Request.Context(), id); err != nil {
		InternalError(c, "恢复沙箱失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "沙箱已恢复", nil)
}

// StopSandbox 停止沙箱
// @Summary 停止沙箱
// @Description 停止运行中的沙箱
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Param force query bool false "是否强制停止" default(false)
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{id}/stop [post]
func (h *AgentHandler) StopSandbox(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	force := ParseBool(c.Query("force"))

	if err := h.agentSvc.StopSandbox(c.Request.Context(), id, force); err != nil {
		InternalError(c, "停止沙箱失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "沙箱已停止", nil)
}

// DeleteSandbox 删除沙箱
// @Summary 删除沙箱
// @Description 删除沙箱及其所有资源
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Success 204 "无内容"
// @Router /api/v1/agents/sandboxes/{id} [delete]
func (h *AgentHandler) DeleteSandbox(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	if err := h.agentSvc.DeleteSandbox(c.Request.Context(), id); err != nil {
		InternalError(c, "删除沙箱失败: "+err.Error())
		return
	}

	NoContent(c)
}

// GetSandboxMetrics 获取沙箱指标
// @Summary 获取沙箱指标
// @Description 获取沙箱的资源使用指标
// @Tags Agent沙箱管理
// @Accept json
// @Produce json
// @Param id path string true "沙箱ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{id}/metrics [get]
func (h *AgentHandler) GetSandboxMetrics(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	metrics, err := h.agentSvc.GetMetrics(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "获取沙箱指标失败: "+err.Error())
		return
	}

	Success(c, metrics)
}

// ==================== Agent管理 API ====================

// ListAgents 获取Agent列表
// @Summary 获取Agent列表
// @Description 获取所有Agent列表
// @Tags Agent管理
// @Accept json
// @Produce json
// @Param sandbox_id query string false "沙箱ID"
// @Param status query string false "Agent状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/agents [get]
func (h *AgentHandler) ListAgents(c *gin.Context) {
	pagination := GetPagination(c)

	req := &agent.ListAgentsRequest{
		TenantID: GetTenantID(c),
		Status:   agent.AgentStatus(c.Query("status")),
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	agents, err := h.agentSvc.ListAgents(c.Request.Context(), req)
	if err != nil {
		InternalError(c, "获取Agent列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, agents, int64(len(agents)), pagination.Page, pagination.PageSize)
}

// CreateAgent 创建Agent
// @Summary 创建Agent
// @Description 在指定沙箱中创建Agent
// @Tags Agent管理
// @Accept json
// @Produce json
// @Param sandbox_id path string true "沙箱ID"
// @Param agent body agent.CreateAgentRequest true "Agent配置"
// @Success 201 {object} Response
// @Router /api/v1/agents/sandboxes/{sandbox_id}/agents [post]
func (h *AgentHandler) CreateAgent(c *gin.Context) {
	sandboxID := ParseUUID(c.Param("sandbox_id"))
	if sandboxID == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	var req agent.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	req.UserID = GetUserID(c)

	agent, err := h.agentSvc.CreateAgent(c.Request.Context(), sandboxID, &req)
	if err != nil {
		InternalError(c, "创建Agent失败: "+err.Error())
		return
	}

	Created(c, agent)
}

// GetAgent 获取Agent详情
// @Summary 获取Agent详情
// @Description 根据ID获取Agent详细信息
// @Tags Agent管理
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/{id} [get]
func (h *AgentHandler) GetAgent(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	agent, err := h.agentSvc.GetAgent(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "Agent不存在")
		return
	}

	Success(c, agent)
}

// DeleteAgent 删除Agent
// @Summary 删除Agent
// @Description 删除指定的Agent
// @Tags Agent管理
// @Accept json
// @Produce json
// @Param id path string true "Agent ID"
// @Success 204 "无内容"
// @Router /api/v1/agents/{id} [delete]
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	if err := h.agentSvc.DeleteAgent(c.Request.Context(), id); err != nil {
		InternalError(c, "删除Agent失败: "+err.Error())
		return
	}

	NoContent(c)
}

// ==================== 执行管理 API ====================

// Execute 执行任务
// @Summary 执行任务
// @Description 在Agent中执行任务
// @Tags Agent执行
// @Accept json
// @Produce json
// @Param request body agent.ExecuteRequest true "执行请求"
// @Success 200 {object} Response
// @Router /api/v1/agents/execute [post]
func (h *AgentHandler) Execute(c *gin.Context) {
	var req agent.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	response, err := h.agentSvc.Execute(c.Request.Context(), &req)
	if err != nil {
		InternalError(c, "执行任务失败: "+err.Error())
		return
	}

	Success(c, response)
}

// ExecuteCode 执行代码
// @Summary 执行代码
// @Description 在Agent沙箱中执行代码
// @Tags Agent执行
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID"
// @Param code body CodeExecuteRequest true "代码执行请求"
// @Success 200 {object} Response
// @Router /api/v1/agents/{agent_id}/execute/code [post]
func (h *AgentHandler) ExecuteCode(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	var req CodeExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	execReq := &agent.ExecuteRequest{
		AgentID:  agentID,
		Type:     agent.ExecutionTypeCode,
		Code:     req.Code,
		Language: req.Language,
		Timeout:  req.Timeout,
	}

	response, err := h.agentSvc.Execute(c.Request.Context(), execReq)
	if err != nil {
		InternalError(c, "执行代码失败: "+err.Error())
		return
	}

	Success(c, response)
}

// ExecuteShell 执行Shell命令
// @Summary 执行Shell命令
// @Description 在Agent沙箱中执行Shell命令
// @Tags Agent执行
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID"
// @Param command body ShellExecuteRequest true "Shell执行请求"
// @Success 200 {object} Response
// @Router /api/v1/agents/{agent_id}/execute/shell [post]
func (h *AgentHandler) ExecuteShell(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	var req ShellExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	execReq := &agent.ExecuteRequest{
		AgentID: agentID,
		Type:    agent.ExecutionTypeShell,
		Command: req.Command,
		Timeout: req.Timeout,
	}

	response, err := h.agentSvc.Execute(c.Request.Context(), execReq)
	if err != nil {
		InternalError(c, "执行Shell命令失败: "+err.Error())
		return
	}

	Success(c, response)
}

// GetExecution 获取执行详情
// @Summary 获取执行详情
// @Description 根据ID获取执行详细信息
// @Tags Agent执行
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/executions/{id} [get]
func (h *AgentHandler) GetExecution(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的执行ID")
		return
	}

	execution, err := h.agentSvc.GetExecution(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "执行记录不存在")
		return
	}

	Success(c, execution)
}

// CancelExecution 取消执行
// @Summary 取消执行
// @Description 取消正在执行的任务
// @Tags Agent执行
// @Accept json
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/executions/{id}/cancel [post]
func (h *AgentHandler) CancelExecution(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的执行ID")
		return
	}

	if err := h.agentSvc.CancelExecution(c.Request.Context(), id); err != nil {
		InternalError(c, "取消执行失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "执行已取消", nil)
}

// ==================== 工具管理 API ====================

// ListTools 获取工具列表
// @Summary 获取工具列表
// @Description 获取Agent可用的工具列表
// @Tags Agent工具
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID"
// @Success 200 {object} Response
// @Router /api/v1/agents/{agent_id}/tools [get]
func (h *AgentHandler) ListTools(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	tools, err := h.agentSvc.ListTools(c.Request.Context(), agentID)
	if err != nil {
		InternalError(c, "获取工具列表失败: "+err.Error())
		return
	}

	Success(c, tools)
}

// RegisterTool 注册工具
// @Summary 注册工具
// @Description 为Agent注册新工具
// @Tags Agent工具
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID"
// @Param tool body agent.CreateToolRequest true "工具配置"
// @Success 201 {object} Response
// @Router /api/v1/agents/{agent_id}/tools [post]
func (h *AgentHandler) RegisterTool(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}

	var req agent.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	tool, err := h.agentSvc.RegisterTool(c.Request.Context(), agentID, &req)
	if err != nil {
		InternalError(c, "注册工具失败: "+err.Error())
		return
	}

	Created(c, tool)
}

// UnregisterTool 注销工具
// @Summary 注销工具
// @Description 注销Agent的工具
// @Tags Agent工具
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID"
// @Param tool_name path string true "工具名称"
// @Success 200 {object} Response
// @Router /api/v1/agents/{agent_id}/tools/{tool_name} [delete]
func (h *AgentHandler) UnregisterTool(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	toolName := c.Param("tool_name")

	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}
	if toolName == "" {
		BadRequest(c, "工具名称不能为空")
		return
	}

	if err := h.agentSvc.UnregisterTool(c.Request.Context(), agentID, toolName); err != nil {
		InternalError(c, "注销工具失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "工具已注销", nil)
}

// ExecuteTool 执行工具
// @Summary 执行工具
// @Description 执行指定的工具
// @Tags Agent工具
// @Accept json
// @Param agent_id path string true "Agent ID"
// @Param tool_name path string true "工具名称"
// @Param params body map[string]interface{} true "工具参数"
// @Success 200 {object} Response
// @Router /api/v1/agents/{agent_id}/tools/{tool_name}/execute [post]
func (h *AgentHandler) ExecuteTool(c *gin.Context) {
	agentID := ParseUUID(c.Param("agent_id"))
	toolName := c.Param("tool_name")

	if agentID == uuid.Nil {
		BadRequest(c, "无效的Agent ID")
		return
	}
	if toolName == "" {
		BadRequest(c, "工具名称不能为空")
		return
	}

	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	execReq := &agent.ExecuteRequest{
		AgentID: agentID,
		Type:    agent.ExecutionTypeTool,
		Input:   toolName,
	}

	response, err := h.agentSvc.Execute(c.Request.Context(), execReq)
	if err != nil {
		InternalError(c, "执行工具失败: "+err.Error())
		return
	}

	Success(c, response)
}

// ==================== 安全策略 API ====================

// ApplySecurityPolicy 应用安全策略
// @Summary 应用安全策略
// @Description 为沙箱应用安全策略
// @Tags Agent安全
// @Accept json
// @Produce json
// @Param sandbox_id path string true "沙箱ID"
// @Param policy body agent.SecurityPolicy true "安全策略"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{sandbox_id}/security [put]
func (h *AgentHandler) ApplySecurityPolicy(c *gin.Context) {
	sandboxID := ParseUUID(c.Param("sandbox_id"))
	if sandboxID == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	var policy agent.SecurityPolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.agentSvc.ApplySecurityPolicy(c.Request.Context(), sandboxID, &policy); err != nil {
		InternalError(c, "应用安全策略失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "安全策略已应用", nil)
}

// ApplyNetworkPolicy 应用网络策略
// @Summary 应用网络策略
// @Description 为沙箱应用网络策略
// @Tags Agent安全
// @Accept json
// @Produce json
// @Param sandbox_id path string true "沙箱ID"
// @Param policy body agent.NetworkPolicySpec true "网络策略"
// @Success 200 {object} Response
// @Router /api/v1/agents/sandboxes/{sandbox_id}/network-policy [put]
func (h *AgentHandler) ApplyNetworkPolicy(c *gin.Context) {
	sandboxID := ParseUUID(c.Param("sandbox_id"))
	if sandboxID == uuid.Nil {
		BadRequest(c, "无效的沙箱ID")
		return
	}

	var policy agent.NetworkPolicySpec
	if err := c.ShouldBindJSON(&policy); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.agentSvc.ApplyNetworkPolicy(c.Request.Context(), sandboxID, &policy); err != nil {
		InternalError(c, "应用网络策略失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "网络策略已应用", nil)
}

// ==================== 服务统计 API ====================

// GetStatistics 获取服务统计
// @Summary 获取服务统计
// @Description 获取Agent服务的统计信息
// @Tags Agent服务
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/agents/statistics [get]
func (h *AgentHandler) GetStatistics(c *gin.Context) {
	stats, err := h.agentSvc.GetStatistics(c.Request.Context())
	if err != nil {
		InternalError(c, "获取统计信息失败: "+err.Error())
		return
	}

	Success(c, stats)
}

// 请求结构体定义

type CodeExecuteRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language"`
	Timeout  int    `json:"timeout"`
}

type ShellExecuteRequest struct {
	Command []string `json:"command" binding:"required"`
	Timeout int      `json:"timeout"`
}
