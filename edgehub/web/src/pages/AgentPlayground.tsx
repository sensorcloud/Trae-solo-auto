import React, { useState, useRef, useEffect } from 'react';

interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  toolCalls?: ToolCall[];
}

interface ToolCall {
  id: string;
  name: string;
  arguments: Record<string, any>;
  result?: string;
  status: 'pending' | 'running' | 'success' | 'error';
}

interface Sandbox {
  id: string;
  name: string;
  status: 'running' | 'stopped' | 'error';
  createdAt: string;
  runtime: string;
  memory: number;
  cpu: number;
}

interface Tool {
  id: string;
  name: string;
  description: string;
  category: '数据查询' | '设备控制' | '能源管理' | '系统操作';
  enabled: boolean;
}

const AgentPlayground: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);
  const [sandboxes, setSandboxes] = useState<Sandbox[]>([]);
  const [activeSandbox, setActiveSandbox] = useState<Sandbox | null>(null);
  const [tools, setTools] = useState<Tool[]>([]);
  const [showToolPanel, setShowToolPanel] = useState(true);
  const [codeInput, setCodeInput] = useState('');
  const [codeOutput, setCodeOutput] = useState('');
  const [codeLanguage, setCodeLanguage] = useState<'python' | 'javascript'>('python');

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const codeEditorRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    initSandboxes();
    initTools();
    addSystemMessage('欢迎使用Agent演练场！您可以在这里测试Agent的各种能力。');
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const initSandboxes = () => {
    const mockSandboxes: Sandbox[] = [
      { id: 'SBX001', name: '生产环境沙箱', status: 'running', createdAt: '2024-01-15 08:00:00', runtime: 'Python 3.11', memory: 512, cpu: 25 },
      { id: 'SBX002', name: '测试环境沙箱', status: 'running', createdAt: '2024-01-15 09:30:00', runtime: 'Node.js 20', memory: 256, cpu: 15 },
      { id: 'SBX003', name: '开发环境沙箱', status: 'stopped', createdAt: '2024-01-14 14:00:00', runtime: 'Python 3.11', memory: 0, cpu: 0 },
    ];
    setSandboxes(mockSandboxes);
    setActiveSandbox(mockSandboxes[0]);
  };

  const initTools = () => {
    const mockTools: Tool[] = [
      { id: 'T001', name: 'query_energy_data', description: '查询能源数据', category: '数据查询', enabled: true },
      { id: 'T002', name: 'get_device_status', description: '获取设备状态', category: '数据查询', enabled: true },
      { id: 'T003', name: 'control_device', description: '控制设备开关', category: '设备控制', enabled: true },
      { id: 'T004', name: 'set_temperature', description: '设置温度参数', category: '设备控制', enabled: true },
      { id: 'T005', name: 'optimize_energy', description: '优化能源调度', category: '能源管理', enabled: true },
      { id: 'T006', name: 'predict_load', description: '预测负载', category: '能源管理', enabled: true },
      { id: 'T007', name: 'execute_script', description: '执行脚本', category: '系统操作', enabled: true },
      { id: 'T008', name: 'read_file', description: '读取文件', category: '系统操作', enabled: true },
    ];
    setTools(mockTools);
  };

  const addSystemMessage = (content: string) => {
    const message: Message = {
      id: `msg-${Date.now()}`,
      role: 'system',
      content,
      timestamp: new Date().toLocaleTimeString('zh-CN'),
    };
    setMessages(prev => [...prev, message]);
  };

  const handleSendMessage = async () => {
    if (!inputValue.trim() || isExecuting) return;

    const userMessage: Message = {
      id: `msg-${Date.now()}`,
      role: 'user',
      content: inputValue,
      timestamp: new Date().toLocaleTimeString('zh-CN'),
    };
    setMessages(prev => [...prev, userMessage]);
    setInputValue('');
    setIsExecuting(true);

    await simulateAgentResponse(inputValue);

    setIsExecuting(false);
  };

  const simulateAgentResponse = async (query: string) => {
    await new Promise(resolve => setTimeout(resolve, 1000));

    let response = '';
    let toolCalls: ToolCall[] = [];

    if (query.includes('温度') || query.includes('温控')) {
      toolCalls = [
        { id: 'tc-1', name: 'get_device_status', arguments: { deviceType: 'temperature_sensor' }, status: 'success', result: '当前温度: 25.6°C' },
        { id: 'tc-2', name: 'set_temperature', arguments: { targetTemp: 24 }, status: 'success', result: '温度已设置为24°C' },
      ];
      response = '我已查询当前温度为25.6°C，并已将目标温度设置为24°C。空调系统正在调整中。';
    } else if (query.includes('能源') || query.includes('电力')) {
      toolCalls = [
        { id: 'tc-1', name: 'query_energy_data', arguments: { timeRange: '24h' }, status: 'success', result: '24小时用电量: 4,521 kWh' },
        { id: 'tc-2', name: 'optimize_energy', arguments: {}, status: 'success', result: '优化建议: 建议在谷时段(22:00-06:00)增加储能充电' },
      ];
      response = '根据过去24小时的数据分析，总用电量为4,521 kWh。我建议在谷时段增加储能充电以降低用电成本。';
    } else if (query.includes('设备')) {
      toolCalls = [
        { id: 'tc-1', name: 'get_device_status', arguments: {}, status: 'success', result: '在线设备: 12台, 离线设备: 1台, 警告设备: 2台' },
      ];
      response = '当前设备状态：在线12台，离线1台，警告2台。需要我查看具体哪个设备的状态吗？';
    } else {
      response = '我理解您的请求。请告诉我您需要执行什么操作，我可以帮助您查询数据、控制设备或优化能源使用。';
    }

    const assistantMessage: Message = {
      id: `msg-${Date.now()}`,
      role: 'assistant',
      content: response,
      timestamp: new Date().toLocaleTimeString('zh-CN'),
      toolCalls,
    };
    setMessages(prev => [...prev, assistantMessage]);
  };

  const handleExecuteCode = async () => {
    if (!codeInput.trim()) return;

    setCodeOutput('执行中...');
    await new Promise(resolve => setTimeout(resolve, 1500));

    if (codeLanguage === 'python') {
      setCodeOutput(`>>> 执行 Python 代码...\n\n结果:\n${codeInput.includes('print') ? 'Hello, Agent Playground!' : '代码执行成功'}\n\n执行时间: 0.023s\n内存使用: 12.5 MB`);
    } else {
      setCodeOutput(`>>> 执行 JavaScript 代码...\n\n结果:\n${codeInput.includes('console.log') ? 'Hello, Agent Playground!' : '代码执行成功'}\n\n执行时间: 0.015s\n内存使用: 8.2 MB`);
    }
  };

  const toggleTool = (toolId: string) => {
    setTools(prev => prev.map(t => t.id === toolId ? { ...t, enabled: !t.enabled } : t));
  };

  const createNewSandbox = () => {
    const newSandbox: Sandbox = {
      id: `SBX${String(sandboxes.length + 1).padStart(3, '0')}`,
      name: `新沙箱-${Date.now().toString(36)}`,
      status: 'running',
      createdAt: new Date().toLocaleString('zh-CN'),
      runtime: codeLanguage === 'python' ? 'Python 3.11' : 'Node.js 20',
      memory: 128,
      cpu: 5,
    };
    setSandboxes(prev => [...prev, newSandbox]);
    setActiveSandbox(newSandbox);
  };

  return (
    <div className="agent-playground">
      <div className="page-header">
        <h1 className="page-title">Agent演练场</h1>
        <div className="header-actions">
          <select
            className="form-select sandbox-select"
            value={activeSandbox?.id || ''}
            onChange={e => setActiveSandbox(sandboxes.find(s => s.id === e.target.value) || null)}
          >
            {sandboxes.map(s => (
              <option key={s.id} value={s.id}>{s.name} ({s.status === 'running' ? '运行中' : '已停止'})</option>
            ))}
          </select>
          <button className="btn btn-primary" onClick={createNewSandbox}>+ 新建沙箱</button>
        </div>
      </div>

      {activeSandbox && (
        <div className="sandbox-info">
          <div className="info-item">
            <span className="info-label">运行时:</span>
            <span className="info-value">{activeSandbox.runtime}</span>
          </div>
          <div className="info-item">
            <span className="info-label">内存:</span>
            <span className="info-value">{activeSandbox.memory} MB</span>
          </div>
          <div className="info-item">
            <span className="info-label">CPU:</span>
            <span className="info-value">{activeSandbox.cpu}%</span>
          </div>
          <div className={`status-indicator ${activeSandbox.status}`}>
            {activeSandbox.status === 'running' ? '运行中' : activeSandbox.status === 'stopped' ? '已停止' : '错误'}
          </div>
        </div>
      )}

      <div className="playground-layout">
        <div className="main-panel">
          <div className="tabs">
            <button className="tab active">对话</button>
            <button className="tab">代码执行</button>
          </div>

          <div className="chat-container">
            <div className="messages-list">
              {messages.map(message => (
                <div key={message.id} className={`message ${message.role}`}>
                  <div className="message-header">
                    <span className="message-role">
                      {message.role === 'user' ? '用户' : message.role === 'assistant' ? 'Agent' : '系统'}
                    </span>
                    <span className="message-time">{message.timestamp}</span>
                  </div>
                  <div className="message-content">{message.content}</div>
                  {message.toolCalls && message.toolCalls.length > 0 && (
                    <div className="tool-calls">
                      <div className="tool-calls-header">工具调用:</div>
                      {message.toolCalls.map(tc => (
                        <div key={tc.id} className={`tool-call ${tc.status}`}>
                          <div className="tool-name">{tc.name}</div>
                          <div className="tool-args">
                            <pre>{JSON.stringify(tc.arguments, null, 2)}</pre>
                          </div>
                          {tc.result && <div className="tool-result">{tc.result}</div>}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              ))}
              {isExecuting && (
                <div className="message assistant">
                  <div className="typing-indicator">
                    <span></span><span></span><span></span>
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>

            <div className="input-area">
              <textarea
                className="message-input"
                placeholder="输入您的指令..."
                value={inputValue}
                onChange={e => setInputValue(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleSendMessage();
                  }
                }}
              />
              <button
                className="btn btn-primary send-btn"
                onClick={handleSendMessage}
                disabled={isExecuting}
              >
                发送
              </button>
            </div>
          </div>

          <div className="code-panel">
            <div className="code-header">
              <div className="language-selector">
                <button
                  className={`lang-btn ${codeLanguage === 'python' ? 'active' : ''}`}
                  onClick={() => setCodeLanguage('python')}
                >
                  Python
                </button>
                <button
                  className={`lang-btn ${codeLanguage === 'javascript' ? 'active' : ''}`}
                  onClick={() => setCodeLanguage('javascript')}
                >
                  JavaScript
                </button>
              </div>
              <button className="btn btn-primary" onClick={handleExecuteCode}>执行代码</button>
            </div>
            <div className="code-editor-container">
              <textarea
                ref={codeEditorRef}
                className="code-editor"
                placeholder={codeLanguage === 'python' ? '# 输入Python代码...' : '// 输入JavaScript代码...'}
                value={codeInput}
                onChange={e => setCodeInput(e.target.value)}
                spellCheck={false}
              />
              <div className="code-output">
                <pre>{codeOutput || '输出将显示在这里...'}</pre>
              </div>
            </div>
          </div>
        </div>

        {showToolPanel && (
          <div className="tool-panel">
            <div className="panel-header">
              <h3>可用工具</h3>
              <button className="btn-text" onClick={() => setShowToolPanel(false)}>隐藏</button>
            </div>
            <div className="tools-list">
              {tools.map(tool => (
                <div key={tool.id} className={`tool-item ${tool.enabled ? 'enabled' : ''}`}>
                  <div className="tool-info">
                    <div className="tool-name">{tool.name}</div>
                    <div className="tool-desc">{tool.description}</div>
                    <div className="tool-category">{tool.category}</div>
                  </div>
                  <label className="toggle">
                    <input
                      type="checkbox"
                      checked={tool.enabled}
                      onChange={() => toggleTool(tool.id)}
                    />
                    <span className="toggle-slider"></span>
                  </label>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default AgentPlayground;
