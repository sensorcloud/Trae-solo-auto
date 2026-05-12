import React, { useEffect, useState } from 'react';
import api from '../utils/api';
import toast from 'react-hot-toast';
import './Workloads.css';

interface Workload {
  name: string;
  namespace: string;
  type: string;
  replicas: number;
  available: number;
  status: string;
}

const Workloads: React.FC = () => {
  const [workloads, setWorkloads] = useState<Workload[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchWorkloads();
  }, []);

  const fetchWorkloads = async () => {
    try {
      const response = await api.get('/api/v1/workloads');
      setWorkloads(response.data);
    } catch (error) {
      toast.error('获取工作负载失败');
      setWorkloads([
        { name: 'api-server', namespace: 'default', type: 'Deployment', replicas: 3, available: 3, status: 'Running' },
        { name: 'scheduler', namespace: 'default', type: 'Deployment', replicas: 2, available: 2, status: 'Running' },
        { name: 'node-agent', namespace: 'kube-system', type: 'DaemonSet', replicas: 10, available: 10, status: 'Running' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="workloads-page">
      <h1 className="page-title">工作负载</h1>

      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>命名空间</th>
              <th>类型</th>
              <th>副本</th>
              <th>可用</th>
              <th>状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {workloads.map((workload, index) => (
              <tr key={index}>
                <td className="font-medium">{workload.name}</td>
                <td>{workload.namespace}</td>
                <td>
                  <span className="type-badge">{workload.type}</span>
                </td>
                <td>{workload.replicas}</td>
                <td>{workload.available}</td>
                <td>
                  <span className={`status-text ${workload.status.toLowerCase()}`}>
                    {workload.status}
                  </span>
                </td>
                <td>
                  <button className="btn btn-secondary btn-sm">详情</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default Workloads;
