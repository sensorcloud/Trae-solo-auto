import React, { useEffect, useState } from 'react';
import api from '../utils/api';
import toast from 'react-hot-toast';
import './GPU.css';

interface GPUNode {
  name: string;
  gpuCount: number;
  usedGpu: number;
  memory: string;
  status: string;
}

const GPU: React.FC = () => {
  const [gpuNodes, setGpuNodes] = useState<GPUNode[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchGPUInfo();
  }, []);

  const fetchGPUInfo = async () => {
    try {
      const response = await api.get('/api/v1/gpu');
      setGpuNodes(response.data);
    } catch (error) {
      toast.error('获取GPU信息失败');
      setGpuNodes([
        { name: 'gpu-node-01', gpuCount: 4, usedGpu: 2, memory: '32Gi', status: '在线' },
        { name: 'gpu-node-02', gpuCount: 4, usedGpu: 4, memory: '32Gi', status: '在线' },
        { name: 'gpu-node-03', gpuCount: 8, usedGpu: 6, memory: '64Gi', status: '在线' },
        { name: 'gpu-node-04', gpuCount: 8, usedGpu: 0, memory: '64Gi', status: '离线' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="gpu-page">
      <h1 className="page-title">GPU管理</h1>

      <div className="gpu-summary">
        <div className="card metric-card">
          <div className="metric-value">
            {gpuNodes.reduce((sum, node) => sum + node.gpuCount, 0)}
          </div>
          <div className="metric-label">GPU总数</div>
        </div>
        <div className="card metric-card">
          <div className="metric-value">
            {gpuNodes.reduce((sum, node) => sum + node.usedGpu, 0)}
          </div>
          <div className="metric-label">已使用</div>
        </div>
        <div className="card metric-card">
          <div className="metric-value">
            {gpuNodes.filter(n => n.status === '在线').length}
          </div>
          <div className="metric-label">在线节点</div>
        </div>
      </div>

      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>节点名称</th>
              <th>GPU数量</th>
              <th>已使用</th>
              <th>显存</th>
              <th>使用率</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            {gpuNodes.map((node, index) => {
              const usagePercent = node.gpuCount > 0 
                ? Math.round((node.usedGpu / node.gpuCount) * 100) 
                : 0;
              return (
                <tr key={index}>
                  <td className="font-medium">{node.name}</td>
                  <td>{node.gpuCount}</td>
                  <td>{node.usedGpu}</td>
                  <td>{node.memory}</td>
                  <td>
                    <div className="usage-bar">
                      <div className="usage-fill" style={{ width: `${usagePercent}%` }}></div>
                    </div>
                    <span className="usage-text">{usagePercent}%</span>
                  </td>
                  <td>
                    <span className={`status-text ${node.status === '在线' ? 'online' : 'offline'}`}>
                      {node.status}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default GPU;
