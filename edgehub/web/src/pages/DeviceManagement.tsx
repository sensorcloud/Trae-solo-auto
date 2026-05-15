import React, { useEffect, useState } from 'react';

interface Device {
  id: string;
  name: string;
  type: '传感器' | '控制器' | '网关' | '执行器' | '智能电表';
  model: string;
  status: 'online' | 'offline' | 'warning' | 'error';
  location: string;
  ipAddress: string;
  lastSeen: string;
  firmware: string;
  metrics: {
    cpu?: number;
    memory?: number;
    temperature?: number;
    battery?: number;
  };
  tags: string[];
}

interface DeviceConfig {
  samplingInterval: number;
  reportingInterval: number;
  alertThreshold: {
    temperature: number;
    battery: number;
  };
  dataRetention: number;
}

const DeviceManagement: React.FC = () => {
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [config, setConfig] = useState<DeviceConfig>({
    samplingInterval: 1000,
    reportingInterval: 60000,
    alertThreshold: { temperature: 60, battery: 20 },
    dataRetention: 30,
  });

  useEffect(() => {
    fetchDevices();
    setLoading(false);
  }, []);

  const fetchDevices = () => {
    const mockDevices: Device[] = [
      {
        id: 'DEV001',
        name: '温度传感器-A01',
        type: '传感器',
        model: 'TS-2000',
        status: 'online',
        location: '数据中心A区-机柜01',
        ipAddress: '192.168.1.101',
        lastSeen: '2024-01-15 12:30:45',
        firmware: 'v2.1.3',
        metrics: { temperature: 28.5, battery: 85 },
        tags: ['温度', '环境监测', 'A区'],
      },
      {
        id: 'DEV002',
        name: '智能电表-B01',
        type: '智能电表',
        model: 'SM-500',
        status: 'online',
        location: '数据中心B区-配电室',
        ipAddress: '192.168.1.102',
        lastSeen: '2024-01-15 12:30:42',
        firmware: 'v3.0.1',
        metrics: { cpu: 45, memory: 62 },
        tags: ['电力', '计量', 'B区'],
      },
      {
        id: 'DEV003',
        name: '空调控制器-C01',
        type: '控制器',
        model: 'AC-CTRL-100',
        status: 'warning',
        location: '数据中心C区-空调机房',
        ipAddress: '192.168.1.103',
        lastSeen: '2024-01-15 12:28:30',
        firmware: 'v1.8.5',
        metrics: { temperature: 42, cpu: 78 },
        tags: ['空调', '温控', 'C区'],
      },
      {
        id: 'DEV004',
        name: '边缘网关-D01',
        type: '网关',
        model: 'EG-1000',
        status: 'online',
        location: '数据中心D区-核心机房',
        ipAddress: '192.168.1.104',
        lastSeen: '2024-01-15 12:30:48',
        firmware: 'v4.2.0',
        metrics: { cpu: 32, memory: 45, temperature: 35 },
        tags: ['网关', '边缘计算', 'D区'],
      },
      {
        id: 'DEV005',
        name: 'UPS执行器-E01',
        type: '执行器',
        model: 'UPS-ACT-200',
        status: 'offline',
        location: '数据中心E区-UPS室',
        ipAddress: '192.168.1.105',
        lastSeen: '2024-01-15 10:15:22',
        firmware: 'v2.0.0',
        metrics: {},
        tags: ['UPS', '电源', 'E区'],
      },
      {
        id: 'DEV006',
        name: '湿度传感器-A02',
        type: '传感器',
        model: 'HS-100',
        status: 'error',
        location: '数据中心A区-机柜02',
        ipAddress: '192.168.1.106',
        lastSeen: '2024-01-15 11:45:10',
        firmware: 'v1.5.2',
        metrics: { battery: 5 },
        tags: ['湿度', '环境监测', 'A区'],
      },
    ];
    setDevices(mockDevices);
  };

  const filteredDevices = devices.filter(device => {
    const matchesSearch =
      device.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      device.id.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = filterType === 'all' || device.type === filterType;
    const matchesStatus = filterStatus === 'all' || device.status === filterStatus;
    return matchesSearch && matchesType && matchesStatus;
  });

  const getStatusColor = (status: Device['status']) => {
    const colors = {
      online: '#10b981',
      offline: '#64748b',
      warning: '#f59e0b',
      error: '#ef4444',
    };
    return colors[status];
  };

  const getStatusText = (status: Device['status']) => {
    const texts = {
      online: '在线',
      offline: '离线',
      warning: '警告',
      error: '故障',
    };
    return texts[status];
  };

  const getTypeIcon = (type: Device['type']) => {
    const icons = {
      传感器: '📡',
      控制器: '🎛️',
      网关: '🌐',
      执行器: '⚙️',
      智能电表: '⚡',
    };
    return icons[type];
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="device-management">
      <div className="page-header">
        <h1 className="page-title">设备管理</h1>
        <button className="btn btn-primary">+ 添加设备</button>
      </div>

      <div className="stats-row">
        <div className="card stat-card">
          <div className="stat-value">{devices.length}</div>
          <div className="stat-label">设备总数</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value success">{devices.filter(d => d.status === 'online').length}</div>
          <div className="stat-label">在线设备</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value warning">{devices.filter(d => d.status === 'warning').length}</div>
          <div className="stat-label">警告设备</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value danger">{devices.filter(d => d.status === 'error').length}</div>
          <div className="stat-label">故障设备</div>
        </div>
      </div>

      <div className="card filters-card">
        <div className="filters-row">
          <div className="search-box">
            <input
              type="text"
              className="form-input"
              placeholder="搜索设备名称或ID..."
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
            />
          </div>
          <select
            className="form-select"
            value={filterType}
            onChange={e => setFilterType(e.target.value)}
          >
            <option value="all">所有类型</option>
            <option value="传感器">传感器</option>
            <option value="控制器">控制器</option>
            <option value="网关">网关</option>
            <option value="执行器">执行器</option>
            <option value="智能电表">智能电表</option>
          </select>
          <select
            className="form-select"
            value={filterStatus}
            onChange={e => setFilterStatus(e.target.value)}
          >
            <option value="all">所有状态</option>
            <option value="online">在线</option>
            <option value="offline">离线</option>
            <option value="warning">警告</option>
            <option value="error">故障</option>
          </select>
        </div>
      </div>

      <div className="devices-grid">
        {filteredDevices.map(device => (
          <div
            key={device.id}
            className={`card device-card ${selectedDevice?.id === device.id ? 'selected' : ''}`}
            onClick={() => setSelectedDevice(device)}
          >
            <div className="device-header">
              <div className="device-icon">{getTypeIcon(device.type)}</div>
              <div className="device-info">
                <div className="device-name">{device.name}</div>
                <div className="device-id">{device.id}</div>
              </div>
              <span
                className="device-status"
                style={{ backgroundColor: getStatusColor(device.status) }}
              >
                {getStatusText(device.status)}
              </span>
            </div>
            <div className="device-details">
              <div className="detail-row">
                <span className="detail-label">型号:</span>
                <span className="detail-value">{device.model}</span>
              </div>
              <div className="detail-row">
                <span className="detail-label">类型:</span>
                <span className="detail-value">{device.type}</span>
              </div>
              <div className="detail-row">
                <span className="detail-label">位置:</span>
                <span className="detail-value">{device.location}</span>
              </div>
              <div className="detail-row">
                <span className="detail-label">IP地址:</span>
                <span className="detail-value">{device.ipAddress}</span>
              </div>
            </div>
            <div className="device-metrics">
              {device.metrics.temperature !== undefined && (
                <div className="metric-item">
                  <span className="metric-label">温度</span>
                  <span className="metric-value">{device.metrics.temperature}°C</span>
                </div>
              )}
              {device.metrics.cpu !== undefined && (
                <div className="metric-item">
                  <span className="metric-label">CPU</span>
                  <span className="metric-value">{device.metrics.cpu}%</span>
                </div>
              )}
              {device.metrics.memory !== undefined && (
                <div className="metric-item">
                  <span className="metric-label">内存</span>
                  <span className="metric-value">{device.metrics.memory}%</span>
                </div>
              )}
              {device.metrics.battery !== undefined && (
                <div className="metric-item">
                  <span className="metric-label">电量</span>
                  <span className={`metric-value ${device.metrics.battery < 20 ? 'low' : ''}`}>
                    {device.metrics.battery}%
                  </span>
                </div>
              )}
            </div>
            <div className="device-tags">
              {device.tags.map(tag => (
                <span key={tag} className="tag">
                  {tag}
                </span>
              ))}
            </div>
            <div className="device-footer">
              <span className="last-seen">最后在线: {device.lastSeen}</span>
              <span className="firmware">固件: {device.firmware}</span>
            </div>
            <div className="device-actions">
              <button className="btn-text" onClick={e => { e.stopPropagation(); setShowConfigModal(true); }}>
                配置
              </button>
              <button className="btn-text">详情</button>
              <button className="btn-text danger">重启</button>
            </div>
          </div>
        ))}
      </div>

      {showConfigModal && selectedDevice && (
        <div className="modal-overlay" onClick={() => setShowConfigModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>设备配置 - {selectedDevice.name}</h3>
              <button className="modal-close" onClick={() => setShowConfigModal(false)}>
                ×
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label>采样间隔 (ms)</label>
                <input
                  type="number"
                  className="form-input"
                  value={config.samplingInterval}
                  onChange={e => setConfig({ ...config, samplingInterval: parseInt(e.target.value) })}
                />
              </div>
              <div className="form-group">
                <label>上报间隔 (ms)</label>
                <input
                  type="number"
                  className="form-input"
                  value={config.reportingInterval}
                  onChange={e => setConfig({ ...config, reportingInterval: parseInt(e.target.value) })}
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>温度告警阈值 (°C)</label>
                  <input
                    type="number"
                    className="form-input"
                    value={config.alertThreshold.temperature}
                    onChange={e => setConfig({
                      ...config,
                      alertThreshold: { ...config.alertThreshold, temperature: parseInt(e.target.value) }
                    })}
                  />
                </div>
                <div className="form-group">
                  <label>电量告警阈值 (%)</label>
                  <input
                    type="number"
                    className="form-input"
                    value={config.alertThreshold.battery}
                    onChange={e => setConfig({
                      ...config,
                      alertThreshold: { ...config.alertThreshold, battery: parseInt(e.target.value) }
                    })}
                  />
                </div>
              </div>
              <div className="form-group">
                <label>数据保留天数</label>
                <input
                  type="number"
                  className="form-input"
                  value={config.dataRetention}
                  onChange={e => setConfig({ ...config, dataRetention: parseInt(e.target.value) })}
                />
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowConfigModal(false)}>
                取消
              </button>
              <button className="btn btn-primary">保存配置</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default DeviceManagement;
