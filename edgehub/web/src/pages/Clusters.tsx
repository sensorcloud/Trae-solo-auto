import React, { useEffect, useState } from 'react';
import api from '../utils/api';
import toast from 'react-hot-toast';
import './Clusters.css';

interface Cluster {
  id: string;
  name: string;
  region: string;
  provider: string;
  status: string;
  nodes: number;
  pods: number;
}

const Clusters: React.FC = () => {
  const [clusters, setClusters] = useState<Cluster[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchClusters();
  }, []);

  const fetchClusters = async () => {
    try {
      const response = await api.get('/api/v1/clusters');
      setClusters(response.data);
    } catch (error) {
      toast.error('获取集群列表失败');
      setClusters([
        { id: '1', name: '边缘-华北集群', region: '华北', provider: '私有云', status: '在线', nodes: 5, pods: 68 },
        { id: '2', name: '边缘-华南集群', region: '华南', provider: '私有云', status: '在线', nodes: 4, pods: 52 },
        { id: '3', name: '边缘-华东集群', region: '华东', provider: '公有云', status: '离线', nodes: 3, pods: 36 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="clusters-page">
      <div className="page-header">
        <h1 className="page-title">集群管理</h1>
        <button className="btn btn-primary">添加集群</button>
      </div>

      <div className="clusters-grid">
        {clusters.map((cluster) => (
          <div key={cluster.id} className="card cluster-card">
            <div className="cluster-header">
              <h3 className="cluster-name">{cluster.name}</h3>
              <span className={`status-badge ${cluster.status === '在线' ? 'online' : 'offline'}`}>
                {cluster.status}
              </span>
            </div>
            <div className="cluster-info">
              <div className="info-row">
                <span className="info-label">区域:</span>
                <span className="info-value">{cluster.region}</span>
              </div>
              <div className="info-row">
                <span className="info-label">提供商:</span>
                <span className="info-value">{cluster.provider}</span>
              </div>
              <div className="info-row">
                <span className="info-label">节点数:</span>
                <span className="info-value">{cluster.nodes}</span>
              </div>
              <div className="info-row">
                <span className="info-label">Pod数:</span>
                <span className="info-value">{cluster.pods}</span>
              </div>
            </div>
            <div className="cluster-actions">
              <button className="btn btn-secondary btn-sm">详情</button>
              <button className="btn btn-secondary btn-sm">配置</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Clusters;
