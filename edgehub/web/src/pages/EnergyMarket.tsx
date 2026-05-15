import React, { useEffect, useState, useRef } from 'react';
import * as echarts from 'echarts';
import './EnergyMarket.css';

interface PriceData {
  time: string;
  price: number;
}

interface MarketOverview {
  currentPrice: number;
  priceChange: number;
  dailyVolume: number;
  greenEnergyRatio: number;
  peakPrice: number;
  valleyPrice: number;
}

const EnergyMarket: React.FC = () => {
  const [overview] = useState<MarketOverview>({
    currentPrice: 0.45,
    priceChange: 2.3,
    dailyVolume: 1250000,
    greenEnergyRatio: 35.6,
    peakPrice: 0.68,
    valleyPrice: 0.28,
  });
  const [priceData, setPriceData] = useState<PriceData[]>([]);
  const [loading, setLoading] = useState(true);
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d'>('24h');

  const priceChartRef = useRef<HTMLDivElement>(null);
  const greenChartRef = useRef<HTMLDivElement>(null);
  const volumeChartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    generateMockData();
    setLoading(false);
  }, [timeRange]);

  useEffect(() => {
    if (priceChartRef.current && priceData.length > 0) {
      initPriceChart();
    }
    if (greenChartRef.current) {
      initGreenChart();
    }
    if (volumeChartRef.current) {
      initVolumeChart();
    }
  }, [priceData, overview]);

  const generateMockData = () => {
    const data: PriceData[] = [];
    const now = new Date();
    const points = timeRange === '24h' ? 24 : timeRange === '7d' ? 168 : 720;

    for (let i = points; i >= 0; i--) {
      const time = new Date(now.getTime() - i * (timeRange === '24h' ? 3600000 : 3600000));
      const basePrice = 0.35 + Math.sin(i / 4) * 0.15;
      const randomFactor = (Math.random() - 0.5) * 0.1;
      data.push({
        time: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        price: Math.max(0.2, Math.min(0.8, basePrice + randomFactor)),
      });
    }
    setPriceData(data);
  };

  const initPriceChart = () => {
    const chart = echarts.init(priceChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => {
          const data = params[0];
          return `${data.axisValue}<br/>电价: ¥${data.value.toFixed(2)}/kWh`;
        },
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
        data: priceData.map(d => d.time),
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
        splitLine: { show: false },
      },
      yAxis: {
        type: 'value',
        name: '¥/kWh',
        nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
        splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
      },
      series: [
        {
          name: '电价',
          type: 'line',
          smooth: true,
          symbol: 'none',
          lineStyle: { width: 2, color: '#2563eb' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(37, 99, 235, 0.3)' },
              { offset: 1, color: 'rgba(37, 99, 235, 0.05)' },
            ]),
          },
          data: priceData.map(d => d.price),
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

  const initGreenChart = () => {
    const chart = echarts.init(greenChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'item',
        formatter: '{b}: {c}% ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: '5%',
        top: 'center',
        textStyle: { color: isDark ? '#a0aec0' : '#64748b' },
      },
      series: [
        {
          name: '能源构成',
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
          emphasis: {
            label: { show: true, fontSize: 14, fontWeight: 'bold' },
          },
          labelLine: { show: false },
          data: [
            { value: 35.6, name: '绿电', itemStyle: { color: '#10b981' } },
            { value: 28.4, name: '火电', itemStyle: { color: '#f59e0b' } },
            { value: 20.0, name: '水电', itemStyle: { color: '#3b82f6' } },
            { value: 16.0, name: '核电', itemStyle: { color: '#8b5cf6' } },
          ],
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

  const initVolumeChart = () => {
    const chart = echarts.init(volumeChartRef.current!);
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';

    const hours = Array.from({ length: 24 }, (_, i) => `${i}:00`);
    const volumes = hours.map(() => Math.floor(Math.random() * 50000) + 30000);

    const option: echarts.EChartsOption = {
      backgroundColor: 'transparent',
      tooltip: {
        trigger: 'axis',
        formatter: (params: any) => `${params[0].axisValue}<br/>交易量: ${(params[0].value / 1000).toFixed(1)} MWh`,
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
        data: hours,
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b', fontSize: 10 },
      },
      yAxis: {
        type: 'value',
        name: 'kWh',
        nameTextStyle: { color: isDark ? '#a0aec0' : '#64748b' },
        axisLine: { lineStyle: { color: isDark ? '#4a5568' : '#e2e8f0' } },
        axisLabel: { color: isDark ? '#a0aec0' : '#64748b' },
        splitLine: { lineStyle: { color: isDark ? '#2d3748' : '#f1f5f9' } },
      },
      series: [
        {
          name: '交易量',
          type: 'bar',
          barWidth: '60%',
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#3b82f6' },
              { offset: 1, color: '#1d4ed8' },
            ]),
            borderRadius: [4, 4, 0, 0],
          },
          data: volumes,
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

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="energy-market">
      <div className="page-header">
        <h1 className="page-title">能源市场</h1>
        <div className="time-range-selector">
          {(['24h', '7d', '30d'] as const).map(range => (
            <button
              key={range}
              className={`time-btn ${timeRange === range ? 'active' : ''}`}
              onClick={() => setTimeRange(range)}
            >
              {range === '24h' ? '24小时' : range === '7d' ? '7天' : '30天'}
            </button>
          ))}
        </div>
      </div>

      <div className="overview-cards">
        <div className="card overview-card">
          <div className="card-icon price-icon">¥</div>
          <div className="card-content">
            <div className="card-value">¥{overview.currentPrice.toFixed(2)}</div>
            <div className="card-label">实时电价/kWh</div>
            <div className={`card-change ${overview.priceChange >= 0 ? 'up' : 'down'}`}>
              {overview.priceChange >= 0 ? '↑' : '↓'} {Math.abs(overview.priceChange).toFixed(1)}%
            </div>
          </div>
        </div>

        <div className="card overview-card">
          <div className="card-icon volume-icon">⚡</div>
          <div className="card-content">
            <div className="card-value">{(overview.dailyVolume / 1000).toFixed(0)} MWh</div>
            <div className="card-label">今日交易量</div>
          </div>
        </div>

        <div className="card overview-card">
          <div className="card-icon green-icon">🌱</div>
          <div className="card-content">
            <div className="card-value">{overview.greenEnergyRatio.toFixed(1)}%</div>
            <div className="card-label">绿电占比</div>
          </div>
        </div>

        <div className="card overview-card">
          <div className="card-icon peak-icon">📊</div>
          <div className="card-content">
            <div className="card-value">¥{overview.peakPrice.toFixed(2)}</div>
            <div className="card-label">峰时电价</div>
            <div className="card-sub">谷时: ¥{overview.valleyPrice.toFixed(2)}</div>
          </div>
        </div>
      </div>

      <div className="charts-row">
        <div className="card chart-card large">
          <div className="card-header">
            <h3>电价走势</h3>
          </div>
          <div ref={priceChartRef} className="chart-container" />
        </div>

        <div className="card chart-card">
          <div className="card-header">
            <h3>能源构成</h3>
          </div>
          <div ref={greenChartRef} className="chart-container" />
        </div>
      </div>

      <div className="card chart-card full">
        <div className="card-header">
          <h3>交易量分布</h3>
        </div>
        <div ref={volumeChartRef} className="chart-container tall" />
      </div>
    </div>
  );
};

export default EnergyMarket;
