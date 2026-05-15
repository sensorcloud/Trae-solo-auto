import React, { useEffect, useState, useRef } from 'react';
import * as echarts from 'echarts';

interface ComputeResource {
  id: string;
  name: string;
  type: 'GPU集群' | 'CPU集群' | '边缘节点';
  power: number;
  utilization: number;
  status: 'active' | 'idle' | 'offline';
  location: string;
}

interface PowerResource {
  id: string;
  name: string;
  type: '市电' | '储能' | '光伏' | '风电';
  capacity: number;
  current: number;
  cost: number;
  status: 'available' | 'limited' | 'unavailable';
}

interface OptimizationSuggestion {
  id: string;
  type: 'load_shift' | 'energy_saving' | 'cost_reduction' | 'performance';
  priority: 'high' | 'medium' | 'low';
  title: string;
  description: string;
  potentialSaving: number;
  impact: string;
  status: 'pending' | 'applied' | 'dismissed';
}

interface CoordinationMetrics {
  computeEfficiency: number;
  energyEfficiency: number;
  costEfficiency: number;
  carbonFootprint: number;
  pue: number;
  greenEnergyRatio: number;
}

const CoordinationDashboard: React.FC = () => {
  const [computeResources, setComputeResources] = useState<ComputeResource[]>([]);
  const [powerResources, setPowerResources] = useState<PowerResource[]>([]);
  const [suggestions, setSuggestions] = useState<OptimizationSuggestion[]>([]);
  const [metrics, setMetrics] = useState<CoordinationMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedTimeRange, setSelectedTimeRange] = useState<'24h' | '7d' | '30d'>('24h');

  const correlationChartRef = useRef<HTMLDivElement>(null);
  const efficiencyChartRef = useRef<HTMLDivElement>(null);
  const costChartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchData();
    setLoading(false);
  }, [selectedTimeRange]);

  useEffect(() => {
    if (correlationChartRef.current) {
      initCorrelationChart();
    }
    if (efficiencyChartRef.current) {
      initEfficiencyChart();
    }
    if (costChartRef.current) {
      initCostChart();
    }
  }, [computeResources, powerResources, metrics]);

  const fetchData = () => {
    setComputeResources([
      { id: 'CR001', name: 'GPU集群-A', type: 'GPU集群', power: 45.2, utilization: 78, status: 'active', location: '数据中心A区' },
      { id: 'CR002', name: 'GPU集群-B', type: 'GPU集群', power: 38.5, utilization: 62, status: 'active', location: '数据中心B区' },
      { id: 'CR003', name: 'CPU集群-C', type: 'CPU集群', power: 12.8, utilization: 45, status: 'active', location: '数据中心C区' },
      { id: 'CR004', name: '边缘节点-D1', type: '边缘节点', power: 2.5, utilization: 32, status: 'idle', location: '边缘站点D1' },
      { id: 'CR005', name: '边缘节点-D2', type: '边缘节点', power: 2.1, utilization: 28, status: 'active', location: '边缘站点D2' },
    ]);

    setPowerResources([
      { id: 'PR001', name: '市电主供', type: '市电', capacity: 200, current: 85.6, cost: 0.45, status: 'available' },
      { id: 'PR002', name: '储能系统', type: '储能', capacity: 50, current: 12.5, cost: 0.35, status: 'available' },
      { id: 'PR003', name: '光伏电站', type: '光伏', capacity: 30, current: 8.2, cost: 0.28, status: 'available' },
      { id: 'PR004', name: '风电场', type: '风电', capacity: 20, current: 5.8, cost: 0.32, status: 'limited' },
    ]);

    setSuggestions([
      {
        id: 'SUG001',
        type: 'load_shift',
        priority: 'high',
        title: '负载时段迁移建议',
        description: '建议将GPU集群B的批量训练任务迁移至谷时段(22:00-06:00)执行，可降低用电成本约15%',
        potentialSaving: 1250,
        impact: '成本降低15%，碳排放减少8%',
        status: 'pending',
      },
      {
        id: 'SUG002',
        type: 'energy_saving',
        priority: 'medium',
        title: '闲置资源休眠',
        description: '边缘节点D1当前利用率仅32%，建议启用动态休眠策略，预计可节省电力消耗',
        potentialSaving: 420,
        impact: '电力节省18%',
        status: 'pending',
      },
      {
        id: 'SUG003',
        type: 'cost_reduction',
        priority: 'high',
        title: '绿电优先调度',
        description: '当前光伏发电充足，建议优先使用绿电供电，可降低碳排放和用电成本',
        potentialSaving: 680,
        impact: '碳排放减少12%，成本降低8%',
        status: 'applied',
      },
      {
        id: 'SUG004',
        type: 'performance',
        priority: 'low',
        title: '算力负载均衡',
        description: 'GPU集群A负载较高，建议将部分任务调度至GPU集群B以提升整体效率',
        potentialSaving: 0,
        impact: '性能提升10%',
        status: 'pending',
      },
    ]);

    setMetrics({
      computeEfficiency: 85.6,
      energyEfficiency: 78.2,
      costEfficiency: 92.4,
      carbonFootprint: 125.8,
      pue: 1.35,
      greenEnergyRatio: 28.5,
    });
  };

  const initCorrelationChart = () => {
    const chart = echarts.init(correlationChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`);
    const computeData = hours.map(() => Math.floor(Math.random() * 40) + 60);
    const powerData = hours.map((_, i) => {
      const base = 80 + Math.sin(i / 4) * 20;
      return Math.floor(base + Math.random() * 10);
    });

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: { trigger: 'axis' },
      legend: {
        data: ['算力利用率', '电力消耗'],
        textStyle: { color: isDark ? '#a0aec0' : '#64748b' },
      },
      grid: { left: '3%', right: '4%', bottom: '3%', top: '15%', containLabel: true },
      xAxis: {
        type: 'category',
        data: hours,
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
      },
      yAxis: [
        {
          type: 'value',
          name: '利用率 (%)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
        },
        {
          type: 'value',
          name: '电力 (kW)',
          nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
          axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
          axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: '算力利用率',
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: { color: '#3b82f6', width: 2 },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(59, 130, 246, 0.3)' },
            { offset: 1, color: 'rgba(59, 130, 246, 0.05)' },
          ])},
          data: computeData,
        },
        {
          name: '电力消耗',
          type: 'bar',
          yAxisIndex: 1,
          barWidth: '40%',
          itemStyle: { color: '#10b981', borderRadius: [4, 4, 0, 0] },
          data: powerData,
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const initEfficiencyChart = () => {
    const chart = echarts.init(efficiencyChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {},
      radar: {
        indicator: [
          { name: '算力效率', max: 100 },
          { name: '能源效率', max: 100 },
          { name: '成本效率', max: 100 },
          { name: '绿色占比', max: 100 },
          { name: 'PUE优化', max: 100 },
        ],
        axisName: { color: isDark ? '#a0aec0' : '#64748b' },
        splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#e2e8f0' } },
        splitArea: { areaStyle: { color: isDark ? ['rgba(255,255,255,0.02)', 'rgba(255,255,255,0.05)'] : ['rgba(0,0,0,0.02)', 'rgba(0,0,0,0.05)'] } },
      },
      series: [
        {
          type: 'radar',
          data: [
            {
              value: [metrics?.computeEfficiency || 85, metrics?.energyEfficiency || 78, metrics?.costEfficiency || 92, (metrics?.greenEnergyRatio || 28) * 2, 100 - ((metrics?.pue || 1.35) - 1) * 100],
              name: '当前状态',
              areaStyle: { color: 'rgba(59, 130, 246, 0.3)' },
              lineStyle: { color: '#3b82f6' },
              itemStyle: { color: '#3b82f6' },
            },
          ],
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const initCostChart = () => {
    const chart = echarts.init(costChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: { trigger: 'item' },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
        textStyle: { color: isDark ? '#a0aec0' : '#64748b' },
      },
      series: [
        {
          name: '成本构成',
          type: 'pie',
          radius: ['40%', '70%'],
          center: ['35%', '50%'],
          avoidLabelOverlap: false,
          itemStyle: {
            borderRadius: 8,
            borderColor: isDark ? '#1a202c' : '#fff',
            borderWidth: 2,
          },
          label: { show: false },
          emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold' } },
          labelLine: { show: false },
          data: [
            { value: 45, name: '市电成本', itemStyle: { color: '#64748b' } },
            { value: 25, name: '储能成本', itemStyle: { color: '#3b82f6' } },
            { value: 18, name: '光伏成本', itemStyle: { color: '#10b981' } },
            { value: 12, name: '风电成本', itemStyle: { color: '#06b6d4' } },
          ],
        },
      ],
    };

    chart.setOption(option);
    const handleResize = () => chart.resize();
    window.addEventListener('resize', handleResize);
    return () => { window.removeEventListener('resize', handleResize); chart.dispose(); };
  };

  const getPriorityColor = (priority: OptimizationSuggestion['priority']) => {
    const colors = { high: '#ef4444', medium: '#f59e0b', low: '#10b981' };
    return colors[priority];
  };

  const getTypeText = (type: OptimizationSuggestion['type']) => {
    const texts = {
      load_shift: '负载迁移',
      energy_saving: '节能优化',
      cost_reduction: '成本优化',
      performance: '性能优化',
    };
    return texts[type];
  };

  const getStatusText = (status: OptimizationSuggestion['status']) => {
    const texts = { pending: '待处理', applied: '已应用', dismissed: '已忽略' };
    return texts[status];
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="coordination-dashboard">
      <div className="page-header">
        <h1 className="page-title">算电协同仪表盘</h1>
        <div className="time-range-selector">
          {(['24h', '7d', '30d'] as const).map(range => (
            <button
              key={range}
              className={`time-btn ${selectedTimeRange === range ? 'active' : ''}`}
              onClick={() => setSelectedTimeRange(range)}
            >
              {range === '24h' ? '24小时' : range === '7d' ? '7天' : '30天'}
            </button>
          ))}
        </div>
      </div>

      <div className="metrics-row">
        <div className="card metric-card">
          <div className="metric-header">
            <span className="metric-title">算力效率</span>
          </div>
          <div className="metric-body">
            <span className="metric-value">{metrics?.computeEfficiency.toFixed(1)}%</span>
          </div>
          <div className="metric-progress">
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${metrics?.computeEfficiency}%`, backgroundColor: '#3b82f6' }} />
            </div>
          </div>
        </div>
        <div className="card metric-card">
          <div className="metric-header">
            <span className="metric-title">能源效率</span>
          </div>
          <div className="metric-body">
            <span className="metric-value">{metrics?.energyEfficiency.toFixed(1)}%</span>
          </div>
          <div className="metric-progress">
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${metrics?.energyEfficiency}%`, backgroundColor: '#10b981' }} />
            </div>
          </div>
        </div>
        <div className="card metric-card">
          <div className="metric-header">
            <span className="metric-title">PUE</span>
          </div>
          <div className="metric-body">
            <span className="metric-value">{metrics?.pue.toFixed(2)}</span>
          </div>
          <div className="metric-sub">目标: &lt;1.3</div>
        </div>
        <div className="card metric-card">
          <div className="metric-header">
            <span className="metric-title">绿电占比</span>
          </div>
          <div className="metric-body">
            <span className="metric-value">{metrics?.greenEnergyRatio.toFixed(1)}%</span>
          </div>
          <div className="metric-progress">
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${metrics?.greenEnergyRatio}%`, backgroundColor: '#10b981' }} />
            </div>
          </div>
        </div>
        <div className="card metric-card">
          <div className="metric-header">
            <span className="metric-title">碳排放</span>
          </div>
          <div className="metric-body">
            <span className="metric-value">{metrics?.carbonFootprint.toFixed(1)}</span>
            <span className="metric-unit">kg CO₂/h</span>
          </div>
        </div>
      </div>

      <div className="resources-row">
        <div className="card resources-card">
          <div className="card-header">
            <h3>算力资源</h3>
          </div>
          <div className="resources-list">
            {computeResources.map(resource => (
              <div key={resource.id} className="resource-item">
                <div className="resource-info">
                  <span className="resource-name">{resource.name}</span>
                  <span className="resource-type">{resource.type}</span>
                </div>
                <div className="resource-metrics">
                  <div className="resource-metric">
                    <span className="metric-label">功率</span>
                    <span className="metric-value">{resource.power} kW</span>
                  </div>
                  <div className="resource-metric">
                    <span className="metric-label">利用率</span>
                    <span className="metric-value">{resource.utilization}%</span>
                  </div>
                </div>
                <div className="utilization-bar">
                  <div className="bar-fill" style={{ width: `${resource.utilization}%`, backgroundColor: resource.utilization > 70 ? '#10b981' : resource.utilization > 40 ? '#f59e0b' : '#64748b' }} />
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="card resources-card">
          <div className="card-header">
            <h3>电力资源</h3>
          </div>
          <div className="resources-list">
            {powerResources.map(resource => (
              <div key={resource.id} className="resource-item">
                <div className="resource-info">
                  <span className="resource-name">{resource.name}</span>
                  <span className="resource-type">{resource.type}</span>
                </div>
                <div className="resource-metrics">
                  <div className="resource-metric">
                    <span className="metric-label">容量</span>
                    <span className="metric-value">{resource.capacity} kW</span>
                  </div>
                  <div className="resource-metric">
                    <span className="metric-label">当前</span>
                    <span className="metric-value">{resource.current} kW</span>
                  </div>
                  <div className="resource-metric">
                    <span className="metric-label">成本</span>
                    <span className="metric-value">¥{resource.cost}/kWh</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="charts-row">
        <div className="card chart-card large">
          <div className="card-header">
            <h3>算力-电力协同曲线</h3>
          </div>
          <div ref={correlationChartRef} className="chart-container" />
        </div>
        <div className="card chart-card">
          <div className="card-header">
            <h3>效率雷达图</h3>
          </div>
          <div ref={efficiencyChartRef} className="chart-container" />
        </div>
      </div>

      <div className="bottom-row">
        <div className="card chart-card">
          <div className="card-header">
            <h3>成本构成</h3>
          </div>
          <div ref={costChartRef} className="chart-container" />
        </div>

        <div className="card suggestions-card">
          <div className="card-header">
            <h3>优化建议</h3>
            <span className="suggestion-count">{suggestions.filter(s => s.status === 'pending').length} 条待处理</span>
          </div>
          <div className="suggestions-list">
            {suggestions.map(suggestion => (
              <div key={suggestion.id} className={`suggestion-item ${suggestion.status}`}>
                <div className="suggestion-header">
                  <span className="suggestion-type">{getTypeText(suggestion.type)}</span>
                  <span className="suggestion-priority" style={{ backgroundColor: getPriorityColor(suggestion.priority) }}>
                    {suggestion.priority === 'high' ? '高' : suggestion.priority === 'medium' ? '中' : '低'}
                  </span>
                  <span className={`suggestion-status ${suggestion.status}`}>{getStatusText(suggestion.status)}</span>
                </div>
                <div className="suggestion-title">{suggestion.title}</div>
                <div className="suggestion-desc">{suggestion.description}</div>
                <div className="suggestion-footer">
                  {suggestion.potentialSaving > 0 && (
                    <span className="potential-saving">预计节省: ¥{suggestion.potentialSaving}/月</span>
                  )}
                  <span className="impact">{suggestion.impact}</span>
                </div>
                {suggestion.status === 'pending' && (
                  <div className="suggestion-actions">
                    <button className="btn btn-primary btn-sm">应用</button>
                    <button className="btn btn-secondary btn-sm">忽略</button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

export default CoordinationDashboard;
