import React, { useEffect, useState } from 'react';
import api from '../utils/api';
import toast from 'react-hot-toast';
import './Dashboard.css';

interface ClusterMetrics {
  totalClusters: number;
  activeClusters: number;
  totalNodes: number;
  activeNodes: number;
  totalPods: number;
  runningPods: number;
  gpuUsage: {
    total: number;
    used: number;
  };
}

const Dashboard: React.FC = () => {
  const [metrics, setMetrics] = useState<ClusterMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchMetrics();
  }, []);

  const fetchMetrics = async () => {
    try {
      const response = await api.get('/api/v1/metrics');
      setMetrics(response.data);
    } catch (error) {
      toast.error('获取指标数据失败');
      setMetrics({
        totalClusters: 3,
        activeClusters: 2,
        totalNodes: 12,
        activeNodes: 10,
        totalPods: 156,
        runningPods: 142,
        gpuUsage: { total: 16, used: 8 },
      });
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="dashboard">
      <h1 className="page-title">仪表盘</h1>

      <div className="metrics-grid">
        <div className="card metric-card">
          <div className="metric-value">{metrics?.totalClusters || 0}</div>
          <div className="metric-label">总集群数</div>
        </div>
        <div className="card metric-card">
          <div className="metric-value success">{metrics?.activeClusters || 0}</div>
          <div className="metric-label">活跃集群</div>
        </div>
        <div className="card metric-card">
          <div className="metric-value">{metrics?.totalNodes || 0}</div>
          <div className="metric-label">总节点数</div>
        </div>
        <div className="card metric-card">
          <div className="metric-value success">{metrics?.activeNodes || 0}</div>
          <div className="metric-label">活跃节点</div>
        </div>
      </div>

      <div className="metrics-row">
        <div className="card metric-card">
          <div className="metric-value">{metrics?.totalPods || 0}</div>
          <div className="metric-label">总Pod数</div>
          <div className="metric-detail">
            <span className="badge badge-success">{metrics?.runningPods || 0} 运行中</span>
          </div>
        </div>
        <div className="card metric-card">
          <div className="metric-value">{metrics?.gpuUsage?.total || 0}</div>
          <div className="metric-label">GPU总数</div>
          <div className="metric-detail">
            <span className="badge badge-info">{metrics?.gpuUsage?.used || 0} 已分配</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
