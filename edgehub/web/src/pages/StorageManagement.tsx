import React, { useEffect, useState, useRef } from 'react';
import * as echarts from 'echarts';

interface StorageDevice {
  id: string;
  name: string;
  type: '锂电池' | '铅酸电池' | '液流电池' | '超级电容';
  capacity: number;
  currentSoc: number;
  maxPower: number;
  status: 'charging' | 'discharging' | 'idle' | 'offline';
  temperature: number;
  health: number;
  location: string;
}

interface ScheduleTask {
  id: string;
  deviceId: string;
  deviceName: string;
  action: 'charge' | 'discharge';
  power: number;
  startTime: string;
  endTime: string;
  status: 'pending' | 'running' | 'completed' | 'cancelled';
}

const StorageManagement: React.FC = () => {
  const [devices, setDevices] = useState<StorageDevice[]>([]);
  const [schedules, setSchedules] = useState<ScheduleTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedDevice, setSelectedDevice] = useState<StorageDevice | null>(null);
  const [showScheduleModal, setShowScheduleModal] = useState(false);

  const socChartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchDevices();
    fetchSchedules();
    setLoading(false);
  }, []);

  useEffect(() => {
    if (socChartRef.current && devices.length > 0) {
      initSocChart();
    }
  }, [devices]);

  const fetchDevices = () => {
    const mockDevices: StorageDevice[] = [
      {
        id: '1',
        name: '储能单元-A01',
        type: '锂电池',
        capacity: 500,
        currentSoc: 78,
        maxPower: 100,
        status: 'charging',
        temperature: 28,
        health: 95,
        location: '数据中心A区',
      },
      {
        id: '2',
        name: '储能单元-A02',
        type: '锂电池',
        capacity: 500,
        currentSoc: 45,
        maxPower: 100,
        status: 'discharging',
        temperature: 32,
        health: 88,
        location: '数据中心A区',
      },
      {
        id: '3',
        name: '储能单元-B01',
        type: '液流电池',
        capacity: 1000,
        currentSoc: 62,
        maxPower: 200,
        status: 'idle',
        temperature: 25,
        health: 92,
        location: '数据中心B区',
      },
      {
        id: '4',
        name: '储能单元-B02',
        type: '铅酸电池',
        capacity: 300,
        currentSoc: 30,
        maxPower: 50,
        status: 'offline',
        temperature: 0,
        health: 75,
        location: '数据中心B区',
      },
      {
        id: '5',
        name: '超级电容-C01',
        type: '超级电容',
        capacity: 50,
        currentSoc: 95,
        maxPower: 500,
        status: 'idle',
        temperature: 22,
        health: 98,
        location: '数据中心C区',
      },
    ];
    setDevices(mockDevices);
  };

  const fetchSchedules = () => {
    const mockSchedules: ScheduleTask[] = [
      {
        id: '1',
        deviceId: '1',
        deviceName: '储能单元-A01',
        action: 'charge',
        power: 80,
        startTime: '2024-01-15 02:00',
        endTime: '2024-01-15 06:00',
        status: 'running',
      },
      {
        id: '2',
        deviceId: '2',
        deviceName: '储能单元-A02',
        action: 'discharge',
        power: 60,
        startTime: '2024-01-15 10:00',
        endTime: '2024-01-15 12:00',
        status: 'pending',
      },
      {
        id: '3',
        deviceId: '3',
        deviceName: '储能单元-B01',
        action: 'discharge',
        power: 150,
        startTime: '2024-01-15 14:00',
        endTime: '2024-01-15 18:00',
        status: 'pending',
      },
    ];
    setSchedules(mockSchedules);
  };

  const initSocChart = () => {
    const chart = echarts.init(socChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '3%',
        top: '10%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: devices.map(d => d.name),
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10, rotate: 30 },
      },
      yAxis: {
        type: 'value',
        max: 100,
        name: 'SOC (%)',
        nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
        splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
      },
      series: [
        {
          name: 'SOC',
          type: 'bar',
          barWidth: '50%',
          itemStyle: {
            color: (params: any) => {
              const value = params.value;
              if (value >= 80) return '#10b981';
              if (value >= 50) return '#3b82f6';
              if (value >= 20) return '#f59e0b';
              return '#ef4444';
            },
            borderRadius: [4, 4, 0, 0],
          },
          data: devices.map(d => d.currentSoc),
        },
      ],
    };

    chart.setOption(option);

    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      chart.dispose();
    };
  };

  const getStatusColor = (status: StorageDevice['status']) => {
    const colors = {
      charging: '#10b981',
      discharging: '#3b82f6',
      idle: '#64748b',
      offline: '#ef4444',
    };
    return colors[status];
  };

  const getStatusText = (status: StorageDevice['status']) => {
    const texts = {
      charging: '充电中',
      discharging: '放电中',
      idle: '空闲',
      offline: '离线',
    };
    return texts[status];
  };

  const getSocColor = (soc: number) => {
    if (soc >= 80) return '#10b981';
    if (soc >= 50) return '#3b82f6';
    if (soc >= 20) return '#f59e0b';
    return '#ef4444';
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="storage-management">
      <div className="page-header">
        <h1 className="page-title">储能管理</h1>
        <button className="btn btn-primary" onClick={() => setShowScheduleModal(true)}>
          + 新建调度
        </button>
      </div>

      <div className="stats-row">
        <div className="card stat-card">
          <div className="stat-value">{devices.length}</div>
          <div className="stat-label">储能设备总数</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value success">{devices.filter(d => d.status !== 'offline').length}</div>
          <div className="stat-label">在线设备</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value">{devices.reduce((sum, d) => sum + d.capacity, 0)} kWh</div>
          <div className="stat-label">总容量</div>
        </div>
        <div className="card stat-card">
          <div className="stat-value">
            {(
              devices.reduce((sum, d) => sum + d.capacity * d.currentSoc * 0.01, 0)
            ).toFixed(0)} kWh
          </div>
          <div className="stat-label">当前储能</div>
        </div>
      </div>

      <div className="card chart-card">
        <div className="card-header">
          <h3>SOC状态概览</h3>
        </div>
        <div ref={socChartRef} className="chart-container" />
      </div>

      <div className="card devices-card">
        <div className="card-header">
          <h3>设备列表</h3>
        </div>
        <div className="devices-grid">
          {devices.map(device => (
            <div
              key={device.id}
              className={`device-item ${selectedDevice?.id === device.id ? 'selected' : ''}`}
              onClick={() => setSelectedDevice(device)}
            >
              <div className="device-header">
                <span className="device-name">{device.name}</span>
                <span
                  className="device-status"
                  style={{ backgroundColor: getStatusColor(device.status) }}
                >
                  {getStatusText(device.status)}
                </span>
              </div>
              <div className="device-info">
                <div className="info-row">
                  <span className="info-label">类型:</span>
                  <span className="info-value">{device.type}</span>
                </div>
                <div className="info-row">
                  <span className="info-label">容量:</span>
                  <span className="info-value">{device.capacity} kWh</span>
                </div>
                <div className="info-row">
                  <span className="info-label">位置:</span>
                  <span className="info-value">{device.location}</span>
                </div>
              </div>
              <div className="soc-bar-container">
                <div className="soc-bar">
                  <div
                    className="soc-fill"
                    style={{
                      width: `${device.currentSoc}%`,
                      backgroundColor: getSocColor(device.currentSoc),
                    }}
                  />
                </div>
                <span className="soc-text" style={{ color: getSocColor(device.currentSoc) }}>
                  {device.currentSoc}%
                </span>
              </div>
              <div className="device-metrics">
                <div className="metric">
                  <span className="metric-label">温度</span>
                  <span className="metric-value">{device.temperature}°C</span>
                </div>
                <div className="metric">
                  <span className="metric-label">健康度</span>
                  <span className="metric-value">{device.health}%</span>
                </div>
                <div className="metric">
                  <span className="metric-label">最大功率</span>
                  <span className="metric-value">{device.maxPower} kW</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="card schedules-card">
        <div className="card-header">
          <h3>充放电调度</h3>
        </div>
        <div className="schedules-table">
          <div className="table-header">
            <div className="table-cell">设备名称</div>
            <div className="table-cell">动作</div>
            <div className="table-cell">功率</div>
            <div className="table-cell">开始时间</div>
            <div className="table-cell">结束时间</div>
            <div className="table-cell">状态</div>
            <div className="table-cell">操作</div>
          </div>
          {schedules.map(schedule => (
            <div key={schedule.id} className="table-row">
              <div className="table-cell">{schedule.deviceName}</div>
              <div className="table-cell">
                <span className={`action-badge ${schedule.action}`}>
                  {schedule.action === 'charge' ? '充电' : '放电'}
                </span>
              </div>
              <div className="table-cell">{schedule.power} kW</div>
              <div className="table-cell">{schedule.startTime}</div>
              <div className="table-cell">{schedule.endTime}</div>
              <div className="table-cell">
                <span className={`status-badge ${schedule.status}`}>
                  {schedule.status === 'pending' && '待执行'}
                  {schedule.status === 'running' && '执行中'}
                  {schedule.status === 'completed' && '已完成'}
                  {schedule.status === 'cancelled' && '已取消'}
                </span>
              </div>
              <div className="table-cell">
                <button className="btn-text">编辑</button>
                <button className="btn-text danger">删除</button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {showScheduleModal && (
        <div className="modal-overlay" onClick={() => setShowScheduleModal(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>新建调度任务</h3>
              <button className="modal-close" onClick={() => setShowScheduleModal(false)}>
                ×
              </button>
            </div>
            <div className="modal-body">
              <div className="form-group">
                <label>选择设备</label>
                <select className="form-select">
                  {devices.filter(d => d.status !== 'offline').map(d => (
                    <option key={d.id} value={d.id}>
                      {d.name} (SOC: {d.currentSoc}%)
                    </option>
                  ))}
                </select>
              </div>
              <div className="form-group">
                <label>调度动作</label>
                <select className="form-select">
                  <option value="charge">充电</option>
                  <option value="discharge">放电</option>
                </select>
              </div>
              <div className="form-group">
                <label>功率 (kW)</label>
                <input type="number" className="form-input" placeholder="输入功率" />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>开始时间</label>
                  <input type="datetime-local" className="form-input" />
                </div>
                <div className="form-group">
                  <label>结束时间</label>
                  <input type="datetime-local" className="form-input" />
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button className="btn btn-secondary" onClick={() => setShowScheduleModal(false)}>
                取消
              </button>
              <button className="btn btn-primary">创建任务</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default StorageManagement;
