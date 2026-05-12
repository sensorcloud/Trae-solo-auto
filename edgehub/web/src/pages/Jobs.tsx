import React, { useEffect, useState } from 'react';
import api from '../utils/api';
import toast from 'react-hot-toast';
import './Jobs.css';

interface Job {
  id: string;
  name: string;
  queue: string;
  priority: string;
  status: string;
  submissionTime: string;
  runTime: string;
}

const Jobs: React.FC = () => {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchJobs();
  }, []);

  const fetchJobs = async () => {
    try {
      const response = await api.get('/api/v1/jobs');
      setJobs(response.data);
    } catch (error) {
      toast.error('获取任务列表失败');
      setJobs([
        { id: '1', name: 'training-job-001', queue: 'default', priority: 'High', status: 'Running', submissionTime: '2024-01-15 10:30', runTime: '2h 15m' },
        { id: '2', name: 'inference-job-001', queue: 'inference', priority: 'Normal', status: 'Pending', submissionTime: '2024-01-15 11:00', runTime: '-' },
        { id: '3', name: 'batch-process-001', queue: 'batch', priority: 'Low', status: 'Completed', submissionTime: '2024-01-15 09:00', runTime: '1h 30m' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="jobs-page">
      <div className="page-header">
        <h1 className="page-title">批处理任务</h1>
        <button className="btn btn-primary">提交任务</button>
      </div>

      <div className="jobs-stats">
        <div className="stat-item">
          <div className="stat-value">{jobs.filter(j => j.status === 'Running').length}</div>
          <div className="stat-label">运行中</div>
        </div>
        <div className="stat-item">
          <div className="stat-value">{jobs.filter(j => j.status === 'Pending').length}</div>
          <div className="stat-label">排队中</div>
        </div>
        <div className="stat-item">
          <div className="stat-value">{jobs.filter(j => j.status === 'Completed').length}</div>
          <div className="stat-label">已完成</div>
        </div>
      </div>

      <div className="card">
        <table className="data-table">
          <thead>
            <tr>
              <th>任务名称</th>
              <th>队列</th>
              <th>优先级</th>
              <th>状态</th>
              <th>提交时间</th>
              <th>运行时长</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.id}>
                <td className="font-medium">{job.name}</td>
                <td>{job.queue}</td>
                <td>
                  <span className={`priority-badge ${job.priority.toLowerCase()}`}>
                    {job.priority}
                  </span>
                </td>
                <td>
                  <span className={`status-text ${job.status.toLowerCase()}`}>
                    {job.status}
                  </span>
                </td>
                <td>{job.submissionTime}</td>
                <td>{job.runTime}</td>
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

export default Jobs;
