import React from 'react';
import { NavLink } from 'react-router-dom';
import './Sidebar.css';

const Sidebar: React.FC = () => {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h1 className="sidebar-logo">EdgeHub</h1>
        <span className="sidebar-version">v1.1.0</span>
      </div>
      <nav className="sidebar-nav">
        <NavLink to="/dashboard" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">📊</span>
          <span>仪表盘</span>
        </NavLink>
        <NavLink to="/clusters" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">🖥️</span>
          <span>集群管理</span>
        </NavLink>
        <NavLink to="/workloads" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">📦</span>
          <span>工作负载</span>
        </NavLink>
        <NavLink to="/jobs" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">⚡</span>
          <span>批处理任务</span>
        </NavLink>
        <NavLink to="/gpu" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">🎮</span>
          <span>GPU管理</span>
        </NavLink>
        <NavLink to="/settings" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
          <span className="nav-icon">⚙️</span>
          <span>系统设置</span>
        </NavLink>
      </nav>
    </aside>
  );
};

export default Sidebar;
