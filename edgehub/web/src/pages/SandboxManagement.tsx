import React, { useEffect, useState } from 'react';

interface Sandbox {
  id: string;
  name: string;
  description: string;
  status: 'running' | 'stopped' | 'creating' | 'error' | 'terminating';
  runtime: 'Python 3.11' | 'Node.js 20' | 'Go 1.21' | 'Rust 1.75';
  createdAt: string;
  lastActive: string;
  resources: {
    cpu: number;
    memory: number;
    storage: number;
  };
  limits: {
    maxCpu: number;
    maxMemory: number;
    maxStorage: number;
    timeout: number;
  };
  environment: {
    variables: Record<string, string>;
    packages: string[];
  };
  owner: string;
  tags: string[];
}

interface SandboxTemplate {
  id: string;
  name: string;
  runtime: Sandbox['runtime'];
  description: string;
  defaultPackages: string[];
}

const SandboxManagement: React.FC = () => {
  const [sandboxes, setSandboxes] = useState<Sandbox[]>([]);
  const [templates, setTemplates] = useState<SandboxTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [selectedSandbox, setSelectedSandbox] = useState<Sandbox | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newSandboxConfig, setNewSandboxConfig] = useState({
    name: '',
    runtime: 'Python 3.11' as Sandbox['runtime'],
    template: '',
    cpu: 1,
    memory: 512,
    timeout: 3600,
  });

  useEffect(() => {
    fetchSandboxes();
    fetchTemplates();
    setLoading(false);
  }, []);

  const fetchSandboxes = () => {
    const mockSandboxes: Sandbox[] = [
      {
        id: 'SBX001',
        name: '生产环境沙箱',
        description: '用于生产环境的Agent测试',
        status: 'running',
        runtime: 'Python 3.11',
        createdAt: '2024-01-15 08:00:00',
        lastActive: '2024-01-15 12:30:45',
        resources: { cpu: 25, memory: 512, storage: 1024 },
        limits: { maxCpu: 2, maxMemory: 2048, maxStorage: 4096, timeout: 7200 },
        environment: {
          variables: { PYTHONPATH: '/app', DEBUG: 'false' },
          packages: ['numpy', 'pandas', 'requests'],
        },
        owner: 'admin',
        tags: ['生产', 'Python', 'Agent'],
      },
      {
        id: 'SBX002',
        name: '测试环境沙箱',
        description: '用于功能测试的沙箱环境',
        status: 'running',
        runtime: 'Node.js 20',
        createdAt: '2024-01-15 09:30:00',
        lastActive: '2024-01-15 12:28:30',
        resources: { cpu: 15, memory: 256, storage: 512 },
        limits: { maxCpu: 1, maxMemory: 1024, maxStorage: 2048, timeout: 3600 },
        environment: {
          variables: { NODE_ENV: 'test' },
          packages: ['axios', 'lodash'],
        },
        owner: 'developer1',
        tags: ['测试', 'Node.js'],
      },
      {
        id: 'SBX003',
        name: '开发环境沙箱',
        description: '开发调试用沙箱',
        status: 'stopped',
        runtime: 'Python 3.11',
        createdAt: '2024-01-14 14:00:00',
        lastActive: '2024-01-14 18:30:00',
        resources: { cpu: 0, memory: 0, storage: 256 },
        limits: { maxCpu: 1, maxMemory: 512, maxStorage: 1024, timeout: 1800 },
        environment: {
          variables: { DEBUG: 'true' },
          packages: ['pytest'],
        },
        owner: 'developer2',
        tags: ['开发', 'Python'],
      },
      {
        id: 'SBX004',
        name: '高性能计算沙箱',
        description: '用于复杂计算任务',
        status: 'running',
        runtime: 'Go 1.21',
        createdAt: '2024-01-15 10:00:00',
        lastActive: '2024-01-15 12:25:10',
        resources: { cpu: 80, memory: 1536, storage: 2048 },
        limits: { maxCpu: 4, maxMemory: 4096, maxStorage: 8192, timeout: 14400 },
        environment: {
          variables: { GOMAXPROCS: '4' },
          packages: [],
        },
        owner: 'admin',
        tags: ['高性能', 'Go', '计算'],
      },
      {
        id: 'SBX005',
        name: '实验性沙箱',
        description: '用于新功能实验',
        status: 'error',
        runtime: 'Rust 1.75',
        createdAt: '2024-01-15 11:00:00',
        lastActive: '2024-01-15 11:45:30',
        resources: { cpu: 0, memory: 0, storage: 128 },
        limits: { maxCpu: 2, maxMemory: 1024, maxStorage: 2048, timeout: 3600 },
        environment: {
          variables: {},
          packages: ['tokio', 'serde'],
        },
        owner: 'researcher1',
        tags: ['实验', 'Rust'],
      },
    ];
    setSandboxes(mockSandboxes);
  };

  const fetchTemplates = () => {
    const mockTemplates: SandboxTemplate[] = [
      { id: 'TPL001', name: 'Python数据分析', runtime: 'Python 3.11', description: '预装numpy、pandas等数据分析库', defaultPackages: ['numpy', 'pandas', 'matplotlib', 'scikit-learn'] },
      { id: 'TPL002', name: 'Node.js Web服务', runtime: 'Node.js 20', description: '预装express、axios等Web开发库', defaultPackages: ['express', 'axios', 'lodash'] },
      { id: 'TPL003', name: 'Go高性能服务', runtime: 'Go 1.21', description: '适合构建高性能服务', defaultPackages: [] },
      { id: 'TPL004', name: 'Rust系统编程', runtime: 'Rust 1.75', description: '适合系统级编程', defaultPackages: ['tokio', 'serde', 'anyhow'] },
    ];
    setTemplates(mockTemplates);
  };

  const filteredSandboxes = sandboxes.filter(sandbox => {
    const matchesSearch = sandbox.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sandbox.id.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = filterStatus === 'all' || sandbox.status === filterStatus;
    return matchesSearch && matchesStatus;
  });

  const getStatusColor = (status: Sandbox['status']) => {
    const colors = {
      running: '#10b981',
      stopped: '#64748b',
      creating: '#3b82f6',
      error: '#ef4444',
      terminating: '#f59e0b',
    };
    return colors[status];
  };

  const getStatusText = (status: Sandbox['status']) => {
    const texts = {
      running: '运行中',
      stopped: '已停止',
      creating: '创建中',
      error: '错误',
      terminating: '终止中',
    };
    return texts[status];
  };

  const getRuntimeIcon = (runtime: Sandbox['runtime']) => {
    const icons = {
      'Python 3.11': '🐍',
      'Node.js 20': '🟢',
      'Go 1.21': '🐹',
      'Rust 1.75': '🦀',
    };
    return icons[runtime];
  };

  const handleCreateSandbox = () => {
    const newSandbox: Sandbox = {
      id: `SBX${String(sandboxes.length + 1).padStart(3, '0')}`,
      name: newSandboxConfig.name,
      description: '新创建的沙箱',
      status: 'creating',
      runtime: newSandboxConfig.runtime,
      createdAt: new Date().toLocaleString('zh-CN'),
      lastActive: new Date().toLocaleString('zh-CN'),
      resources: { cpu: 0, memory: 0, storage: 0 },
      limits: {
        maxCpu: newSandboxConfig.cpu,
        maxMemory: newSandboxConfig.memory,
        maxStorage: 1024,
        timeout: newSandboxConfig.timeout,
      },
      environment: { variables: {}, packages: [] },
      owner: 'current_user',
      tags: [],
    };
    setSandboxes(prev => [...prev, newSandbox]);
    setShowCreateModal(false);
    setNewSandboxConfig({ name: '', runtime: 'Python 3.11', template: '', cpu: 1, memory: 512, timeout: 3600 });

    setTimeout(() => {
      setSandboxes(prev => prev.map(s => s.id === newSandbox.id ? { ...s, status: 'running', resources: { cpu: 5, memory: 128, storage: 64 } } : s));
    }, 3000);
  };

  const handleStartSandbox = (sandboxId: string) => {
    setSandboxes(prev => prev.map(s => s.id === sandboxId ? { ...s, status: 'running', resources: { cpu: 5, memory: 128, storage: s.resources.storage } } : s));
  };

  const handleStopSandbox = (sandboxId: string) => {
    setSandboxes(prev => prev.map(s => s.id === sandboxId ? { ...s, status: 'stopped', resources: { cpu: 0, memory: 0, storage: s.resources.storage } } : s));
  };

  const handleDeleteSandbox = (sandboxId: string) => {
    setSandboxes(prev => prev.filter(s => s.id !== sandboxId));
    if (selectedSandbox?.id === sandboxId) {
      setSelectedSandbox(null);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="sandbox-management">
      <div className="page-header">
        <h1 className="page-title">沙箱管理</h1>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>+ 创建沙箱</button>
      </div>

      <div className="stats-row">
        <div className="card stat-card">
          <div className="stat-value">{sandboxes.length}</div>
          <div className="stat-label">沙箱总数</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value success">{sandboxes.filter(s => s.status === 'running').length}</div>
          <div className="stat-label">运行中</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value">{sandboxes.filter(s => s.status === 'stopped').length}</div>
          <div className="stat-label">已停止</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value danger">{sandboxes.filter(s => s.status === 'error').length}</div>
          <div className="stat-label">错误</div>
        </div>
      </div>

      <div className="card filters-card">
        <div className="filters-row">
          <div className="search-box">
            <input
              type="text"
              className="form-input"
              placeholder="搜索沙箱名称或ID..."
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
            />
          </div>
          <select
            className="form-select"
            value={filterStatus}
            onChange={e => setFilterStatus(e.target.value)}
          >
            <option value="all">所有状态</option>
            <option value="running">运行中</option>
            <option value="stopped">已停止</option>
            <option value="creating">创建中</option>
            <option value="error">错误</option>
          </select>
        </div>
      </div>

      <div className="content-layout">
        <div className="sandboxes-list">
          {filteredSandboxes.map(sandbox => (
            <div
              key={sandbox.id}
              className={`card sandbox-card ${selectedSandbox?.id === sandbox.id ? 'selected' : ''}`}
              onClick={() => setSelectedSandbox(sandbox)}
            >
              <div className="sandbox-header">
                <div className="sandbox-icon">{getRuntimeIcon(sandbox.runtime)}</div>
                <div className="sandbox-info">
                  <div className="sandbox-name">{sandbox.name}</div>
                  <div className="sandbox-id">{sandbox.id}</div>
                </div>
                <span className="sandbox-status" style={{ backgroundColor: getStatusColor(sandbox.status) }}>
                  {getStatusText(sandbox.status)}
                </span>
              </div>
              <div className="sandbox-details">
                <div className="detail-row">
                  <span className="detail-label">运行时:</span>
                  <span className="detail-value">{sandbox.runtime}</span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">创建时间:</span>
                  <span className="detail-value">{sandbox.createdAt}</span>
                </div>
              </div>
              <div className="resource-bars">
                <div className="resource-bar">
                  <span className="resource-label">CPU</span>
                  <div className="bar">
                    <div className="bar-fill" style={{ width: `${(sandbox.resources.cpu / sandbox.limits.maxCpu / 100) * 100}%` }} />
                  </div>
                  <span className="resource-value">{sandbox.resources.cpu}%</span>
                </div>
                <div className="resource-bar">
                  <span className="resource-label">内存</span>
                  <div className="bar">
                    <div className="bar-fill" style={{ width: `${(sandbox.resources.memory / sandbox.limits.maxMemory) * 100}%` }} />
                  </div>
                  <span className="resource-value">{sandbox.resources.memory}MB</span>
                </div>
              </div>
              <div className="sandbox-tags">
                {sandbox.tags.map(tag => (
                  <span key={tag} className="tag">{tag}</span>
                ))}
              </div>
              <div className="sandbox-actions">
                {sandbox.status === 'stopped' && (
                  <button className="btn-text" onClick={e => { e.stopPropagation(); handleStartSandbox(sandbox.id); }}>启动</button>
                )}
                {sandbox.status === 'running' && (
                  <button className="btn-text warning" onClick={e => { e.stopPropagation(); handleStopSandbox(sandbox.id); }}>停止</button>
                )}
                <button className="btn-text">详情</button>
                <button className="btn-text danger" onClick={e => { e.stopPropagation(); handleDeleteSandbox(sandbox.id); }}>删除</button>
              </div>
            </div>
          ))}
        </div>

        {selectedSandbox && (
          <div className="card detail-panel">
            <div className="panel-header">
              <h3>{selectedSandbox.name}</h3>
              <span className="sandbox-status" style={{ backgroundColor: getStatusColor(selectedSandbox.status) }}>
                {getStatusText(selectedSandbox.status)}
              </span>
            </div>
            <div className="panel-section">
              <h4>基本信息</h4>
              <div className="info-grid">
                <div className="info-item">
                  <span className="info-label">ID</span>
                  <span className="info-value">{selectedSandbox.id}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">运行时</span>
                  <span className="info-value">{selectedSandbox.runtime}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">创建时间</span>
                  <span className="info-value">{selectedSandbox.createdAt}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">最后活动</span>
                  <span className="info-value">{selectedSandbox.lastActive}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">所有者</span>
                  <span className="info-value">{selectedSandbox.owner}</span>
                </div>
                <div className="info-item">
                  <span className="info-label">超时时间</span>
                  <span className="info-value">{selectedSandbox.limits.timeout}秒</span>
                </div>
              </div>
            </div>
            <div className="panel-section">
              <h4>资源配置</h4>
              <div className="limits-grid">
                <div className="limit-item">
                  <span className="limit-label">CPU限制</span>
                  <span className="limit-value">{selectedSandbox.limits.maxCpu} 核</span>
                </div>
                <div className="limit-item">
                  <span className="limit-label">内存限制</span>
                  <span className="limit-value">{selectedSandbox.limits.maxMemory} MB</span>
                </div>
                <div className="limit-item">
                  <span className="limit-label">存储限制</span>
                  <span className="limit-value">{selectedSandbox.limits.maxStorage} MB</span>
                </div>
              </div>
            </div>
            <div className="panel-section">
              <h4>已安装包</h4>
              <div className="packages-list">
                {selectedSandbox.environment.packages.length > 0 ? (
                  selectedSandbox.environment.packages.map(pkg => (
                    <span key={pkg} className="package-tag">{pkg}</span>
                  ))
                ) : (
                  <span className="no-packages">暂无已安装的包</span>
                )}
              </div>
            </div>
            <div className="panel-section">
              <h4>环境变量</h4>
              <div className="env-vars">
                {Object.entries(selectedSandbox.environment.variables).map(([key, value]) => (
                  <div key={key} className="env-var">
                    <span className="env-key">{key}</span>
                    <span className="env-value">= {value}</span>
                  </div>
                ))}
                {Object.keys(selectedSandbox.environment.variables).length === 0 && (
                  <span className="no-env">暂无环境变量</span>
                )}
              </div>
            </div>
          </div>
        )}
      </div>

      {showCreateModal && (
        <div className="modal-overlay" onClick={() => setShowCreateModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>创建新沙箱</h3>
              <button className="modal-close" onClick={() => setShowCreateModal(false)}>×</button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label>沙箱名称</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder="输入沙箱名称"
                  value={newSandboxConfig.name}
                  onChange={e => setNewSandboxConfig({ ...newSandboxConfig, name: e.target.value })}
                />
              </div>
              <div className="form-group">
                <label>运行时环境</label>
                <select
                  className="form-select"
                  value={newSandboxConfig.runtime}
                  onChange={e => setNewSandboxConfig({ ...newSandboxConfig, runtime: e.target.value as Sandbox['runtime'] })}
                >
                  <option value="Python 3.11">Python 3.11</option>
                  <option value="Node.js 20">Node.js 20</option>
                  <option value="Go 1.21">Go 1.21</option>
                  <option value="Rust 1.75">Rust 1.75</option>
                </select>
              </div>
              <div className="form-group">
                <label>使用模板</label>
                <select
                  className="form-select"
                  value={newSandboxConfig.template}
                  onChange={e => setNewSandboxConfig({ ...newSandboxConfig, template: e.target.value })}
                >
                  <option value="">不使用模板</option>
                  {templates.map(t => (
                    <option key={t.id} value={t.id}>{t.name}</option>
                  ))}
                </select>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>CPU核心数</label>
                  <input
                    type="number"
                    className="form-input"
                    min="1"
                    max="8"
                    value={newSandboxConfig.cpu}
                    onChange={e => setNewSandboxConfig({ ...newSandboxConfig, cpu: parseInt(e.target.value) })}
                  />
                </div>
                <div className="form-group">
                  <label>内存 (MB)</label>
                  <input
                    type="number"
                    className="form-input"
                    min="128"
                    step="128"
                    value={newSandboxConfig.memory}
                    onChange={e => setNewSandboxConfig({ ...newSandboxConfig, memory: parseInt(e.target.value) })}
                  />
                </div>
              </div>
              <div className="form-group">
                <label>超时时间 (秒)</label>
                <input
                  type="number"
                  className="form-input"
                  min="60"
                  step="60"
                  value={newSandboxConfig.timeout}
                  onChange={e => setNewSandboxConfig({ ...newSandboxConfig, timeout: parseInt(e.target.value) })}
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowCreateModal(false)}>取消</button>
              <button className="btn btn-primary" onClick={handleCreateSandbox} disabled={!newSandboxConfig.name}>创建</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SandboxManagement;
