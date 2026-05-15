import React, { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import toast from 'react-hot-toast';
import './Settings.css';

const Settings: React.FC = () => {
  useAuth();
  const [activeTab, setActiveTab] = useState('general');

  const handleSave = () => {
    toast.success('设置已保存');
  };

  return (
    <div className="settings-page">
      <h1 className="page-title">系统设置</h1>

      <div className="settings-layout">
        <div className="settings-nav">
          <button 
            className={`settings-nav-item ${activeTab === 'general' ? 'active' : ''}`}
            onClick={() => setActiveTab('general')}
          >
            通用设置
          </button>
          <button 
            className={`settings-nav-item ${activeTab === 'cluster' ? 'active' : ''}`}
            onClick={() => setActiveTab('cluster')}
          >
            集群配置
          </button>
          <button 
            className={`settings-nav-item ${activeTab === 'scheduler' ? 'active' : ''}`}
            onClick={() => setActiveTab('scheduler')}
          >
            调度配置
          </button>
          <button 
            className={`settings-nav-item ${activeTab === 'security' ? 'active' : ''}`}
            onClick={() => setActiveTab('security')}
          >
            安全设置
          </button>
        </div>

        <div className="settings-content card">
          {activeTab === 'general' && (
            <div className="settings-section">
              <h2>通用设置</h2>
              <div className="form-group">
                <label>平台名称</label>
                <input type="text" className="input" defaultValue="边缘算力聚合平台" />
              </div>
              <div className="form-group">
                <label>API地址</label>
                <input type="text" className="input" defaultValue="http://localhost:8080" />
              </div>
              <div className="form-group">
                <label>刷新间隔</label>
                <select className="input">
                  <option value="10">10秒</option>
                  <option value="30">30秒</option>
                  <option value="60">1分钟</option>
                </select>
              </div>
              <button className="btn btn-primary" onClick={handleSave}>保存设置</button>
            </div>
          )}

          {activeTab === 'cluster' && (
            <div className="settings-section">
              <h2>集群配置</h2>
              <div className="form-group">
                <label>默认集群</label>
                <select className="input">
                  <option value="beijing">华北集群</option>
                  <option value="shanghai">华东集群</option>
                </select>
              </div>
              <div className="form-group">
                <label>服务网格</label>
                <select className="input">
                  <option value="linkerd">Linkerd</option>
                  <option value="istio">Istio</option>
                </select>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" defaultChecked />
                  启用自动故障转移
                </label>
              </div>
              <button className="btn btn-primary" onClick={handleSave}>保存设置</button>
            </div>
          )}

          {activeTab === 'scheduler' && (
            <div className="settings-section">
              <h2>调度配置</h2>
              <div className="form-group">
                <label>调度器类型</label>
                <select className="input">
                  <option value="kueue">Kueue</option>
                  <option value="volcano">Volcano</option>
                </select>
              </div>
              <div className="form-group">
                <label>默认优先级</label>
                <select className="input">
                  <option value="low">低</option>
                  <option value="normal">普通</option>
                  <option value="high">高</option>
                </select>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" defaultChecked />
                  启用GPU调度
                </label>
              </div>
              <button className="btn btn-primary" onClick={handleSave}>保存设置</button>
            </div>
          )}

          {activeTab === 'security' && (
            <div className="settings-section">
              <h2>安全设置</h2>
              <div className="form-group">
                <label>会话超时</label>
                <select className="input">
                  <option value="30">30分钟</option>
                  <option value="60">1小时</option>
                  <option value="120">2小时</option>
                </select>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" defaultChecked />
                  启用双因素认证
                </label>
              </div>
              <div className="form-group">
                <label className="checkbox-label">
                  <input type="checkbox" defaultChecked />
                  记录审计日志
                </label>
              </div>
              <button className="btn btn-primary" onClick={handleSave}>保存设置</button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Settings;
