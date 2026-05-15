import React, { useEffect, useState, useRef } from 'react';
import * as echarts from 'echarts';

interface TelemetryData {
  timestamp: string;
  temperature: number;
  humidity: number;
  power: number;
  voltage: number;
  current: number;
}

interface Alert {
  id: string;
  deviceId: string;
  deviceName: string;
  type: 'temperature' | 'power' | 'voltage' | 'connectivity' | 'battery';
  severity: 'critical' | 'warning' | 'info';
  message: string;
  timestamp: string;
  acknowledged: boolean;
}

interface RealtimeMetric {
  name: string;
  value: number;
  unit: string;
  trend: 'up' | 'down' | 'stable';
  change: number;
}

const TelemetryDashboard: React.FC = () => {
  const [telemetryData, setTelemetryData] = useState<TelemetryData[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedTimeRange, setSelectedTimeRange] = useState<'1h' | '6h' | '24h' | '7d'>('1h');
  const [realtimeMetrics, setRealtimeMetrics] = useState<RealtimeMetric[]>([]);

  const temperatureChartRef = useRef<HTMLDivElement>(null);
  const powerChartRef = useRef<HTMLDivElement>(null);
  const gaugeChartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchTelemetryData();
    fetchAlerts();
    fetchRealtimeMetrics();
    setLoading(false);

    const interval = setInterval(() => {
      updateRealtimeData();
    }, 5000);

    return () => clearInterval(interval);
  }, [selectedTimeRange]);

  useEffect(() => {
    if (temperatureChartRef.current && telemetryData.length > 0) {
      initTemperatureChart();
    }
    if (powerChartRef.current && telemetryData.length > 0) {
      initPowerChart();
    }
    if (gaugeChartRef.current) {
      initGaugeChart();
    }
  }, [telemetryData, realtimeMetrics]);

  const fetchTelemetryData = () => {
    const data: TelemetryData[] = [];
    const now = new Date();
    const points = selectedTimeRange === '1h' ? 60 : selectedTimeRange === '6h' ? 72 : selectedTimeRange === '24h' ? 96 : 168;

    for (let i = points; i >= 0; i--) {
      const time = new Date(now.getTime() - i * (selectedTimeRange === '1h' ? 60000 : selectedTimeRange === '6h' ? 300000 : selectedTimeRange === '24h' ? 900000 : 3600000));
      data.push({
        timestamp: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        temperature: 22 + Math.random() * 8,
        humidity: 40 + Math.random() * 20,
        power: 150 + Math.random() * 50,
        voltage: 218 + Math.random() * 10,
        current: 0.6 + Math.random() * 0.3,
      });
    }
    setTelemetryData(data);
  };

  const fetchAlerts = () => {
    const mockAlerts: Alert[] = [
      {
        id: 'ALT001',
        deviceId: 'DEV003',
        deviceName: '空调控制器-C01',
        type: 'temperature',
        severity: 'critical',
        message: '温度超过阈值 (45°C > 40°C)',
        timestamp: '2024-01-15 12:28:30',
        acknowledged: false,
      },
      {
        id: 'ALT002',
        deviceId: 'DEV006',
        deviceName: '湿度传感器-A02',
        type: 'battery',
        severity: 'warning',
        message: '电池电量低 (5%)',
        timestamp: '2024-01-15 11:45:10',
        acknowledged: false,
      },
      {
        id: 'ALT003',
        deviceId: 'DEV005',
        deviceName: 'UPS执行器-E01',
        type: 'connectivity',
        severity: 'critical',
        message: '设备离线超过2小时',
        timestamp: '2024-01-15 10:15:22',
        acknowledged: true,
      },
      {
        id: 'ALT004',
        deviceId: 'DEV002',
        deviceName: '智能电表-B01',
        type: 'voltage',
        severity: 'warning',
        message: '电压波动异常 (±5%)',
        timestamp: '2024-01-15 09:30:45',
        acknowledged: false,
      },
    ];
    setAlerts(mockAlerts);
  };

  const fetchRealtimeMetrics = () => {
    setRealtimeMetrics([
      { name: '平均温度', value: 25.6, unit: '°C', trend: 'up', change: 2.3 },
      { name: '平均湿度', value: 52.4, unit: '%', trend: 'stable', change: 0.1 },
      { name: '总功率', value: 185.2, unit: 'kW', trend: 'down', change: -3.5 },
      { name: '平均电压', value: 222.5, unit: 'V', trend: 'stable', change: 0.2 },
    ]);
  };

  const updateRealtimeData = () => {
    setRealtimeMetrics(prev => prev.map(m => ({
      ...m,
      value: m.value + (Math.random() - 0.5) * 2,
      change: (Math.random() - 0.5) * 5,
    })));
  };

  const initTemperatureChart = () => {
    const chart = echarts.init(temperatureChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: { trigger: 'axis' },
      legend: {
        data: ['温度', '湿度'],
        textStyle: { color: isDark ? '#a0aec0' : '#64748b' },
      },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '15%', containLabel: true },
      xAxis: {
        type: 'category',
        data: telemetryData.map(d => d.timestamp),
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
      },
      yAxis: [
        {
          type: 'value',
          name: '温度 (°C)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
        },
        {
          type: 'value',
          name: '湿度 (%)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: '温度',
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: { color: '#ef4444' },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(239, 68, 68, 0.3)' },
            { offset: 1, color: 'rgba(239, 68, 68, 0.05)' },
          ])},
          data: telemetryData.map(d => d.temperature.toFixed(1)),
        },
        {
          name: '湿度',
          type: 'line',
          smooth: true,
          symbol: 'none',
          yAxisIndex: 1,
          lineStyle: { color: '#3b82f6' },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
            { offset: 1, color: 'rgba(59, 130, 246, 0.05)' },
          ])},
          data: telemetryData.map(d => d.humidity.toFixed(1)),
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const initPowerChart = () => {
    const chart = echarts.init(powerChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: { trigger: 'axis' },
      legend: {
        data: ['功率', '电压'],
        textStyle: { color: isDark ? '#a0aec0' : '#64748b' },
      },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '15%', containLabel: true },
      xAxis: {
        type: 'category',
        data: telemetryData.map(d => d.timestamp),
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
      },
      yAxis: [
        {
          type: 'value',
          name: '功率 (kW)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
        },
        {
          type: 'value',
          name: '电压 (V)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: '功率',
          type: 'bar',
          barWidth: '40%',
          itemStyle: { color: '#10b981', borderRadius: [4, 4, 0, 0] },
          data: telemetryData.map(d => d.power.toFixed(1)),
        },
        {
          name: '电压',
          type: 'line',
          smooth: true,
          symbol: 'none',
          yAxisIndex: 1,
          lineStyle: { color: '#f59e0b' },
          data: telemetryData.map(d => d.voltage.toFixed(1)),
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const initGaugeChart = () => {
    const chart = echarts.init(gaugeChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      series: [
        {
          type: 'gauge',
          center: ['50%', '60%'],
          radius: '80%',
          startAngle: 200,
          endAngle: -20,
          min: 0,
          max: 300,
          splitNumber: 6,
          itemStyle: { color: '#10b981' },
          progress: { show: true, width: 20 },
          pointer: { show: false },
          axisLine: { lineStyle: { width: 20, color: [[1, isDark ? '#2d3748' : '#e2e8f0']] } },
          axisTick: { show: false },
          splitLine: { show: false },
          axisLabel: { show: false },
          anchor: { show: false },
          title: { show: false },
          detail: {
            valueAnimation: true,
            fontSize: 32,
            fontWeight: 'bold',
            color: isDark ? '#e2e8f0' : '#1e293b',
            formatter: '{value} kW',
            offsetCenter: [0, '10%'],
          },
          data: [{ value: realtimeMetrics[2]?.value || 185.2 }],
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const getSeverityColor = (severity: Alert['severity']) => {
    const colors = { critical: '#ef4444', warning: '#f59e0b', info: '#3b82f6' };
    return colors[severity];
  };

  const getSeverityText = (severity: Alert['severity']) => {
    const texts = { critical: '严重', warning: '警告', info: '信息' };
    return texts[severity];
  };

  const getTrendIcon = (trend: RealtimeMetric['trend']) => {
    const icons = { up: '↑', down: '↓', stable: '→' };
    return icons[trend];
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="telemetry-dashboard">
      <div className="page-header">
        <h1 className="page-title">遥测仪表盘</h1>
        <div className="time-range-selector">
          {(['1h', '6h', '24h', '7d'] as const).map(range => (
            <button
              key={range}
              className={`time-btn ${selectedTimeRange === range ? 'active' : ''}`}
              onClick={() => setSelectedTimeRange(range)}
            >
              {range === '1h' ? '1小时' : range === '6h' ? '6小时' : range === '24h' ? '24小时' : '7天'}
            </button>
          ))}
        </div>
      </div>

      <div className="realtime-metrics">
        {realtimeMetrics.map((metric, idx) => (
          <div key={idx} className="card metric-card">
            <div className="metric-header">
              <span className="metric-name">{metric.name}</span>
              <span className={`metric-trend ${metric.trend}`}>
                {getTrendIcon(metric.trend)} {Math.abs(metric.change).toFixed(1)}%
              </span>
            </div>
            <div className="metric-body">
              <span className="metric-value">{metric.value.toFixed(1)}</span>
              <span className="metric-unit">{metric.unit}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="charts-row">
        <div className="card chart-card">
          <div className="card-header">
            <h3>温湿度趋势</h3>
          </div>
          <div ref={temperatureChartRef} className="chart-container" />
        </div>
        <div className="card chart-card">
          <div className="card-header">
            <h3>功率/电压趋势</h3>
          </div>
          <div ref={powerChartRef} className="chart-container" />
        </div>
      </div>

      <div className="bottom-row">
        <div className="card gauge-card">
          <div className="card-header">
            <h3>实时功率</h3>
          </div>
          <div ref={gaugeChartRef} className="gauge-container" />
        </div>

        <div className="card alerts-card">
          <div className="card-header">
            <h3>告警列表</h3>
            <span className="alert-count">{alerts.filter(a => !a.acknowledged).length} 未处理</span>
          </div>
          <div className="alerts-list">
            {alerts.map(alert => (
              <div key={alert.id} className={`alert-item ${alert.severity}`}>
                <div className="alert-indicator" style={{ backgroundColor: getSeverityColor(alert.severity) }} />
                <div className="alert-content">
                  <div className="alert-header">
                    <span className="alert-device">{alert.deviceName}</span>
                    <span className={`alert-severity ${alert.severity}`}>
                      {getSeverityText(alert.severity)}
                    </span>
                  </div>
                  <div className="alert-message">{alert.message}</div>
                  <div className="alert-time">{alert.timestamp}</div>
                </div>
                {!alert.acknowledged && (
                  <button className="btn-text acknowledge-btn">确认</button>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default TelemetryDashboard;
