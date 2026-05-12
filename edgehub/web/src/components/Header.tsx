import React from 'react';
import { useAuth } from '../contexts/AuthContext';
import './Header.css';

const Header: React.FC = () => {
  const { user, logout } = useAuth();

  return (
    <header className="header">
      <div className="header-left">
        <h2 className="header-title">边缘算力聚合平台</h2>
      </div>
      <div className="header-right">
        <div className="header-user">
          <span className="user-name">{user?.name || user?.email}</span>
          <button onClick={logout} className="btn btn-secondary btn-sm">
            退出
          </button>
        </div>
      </div>
    </header>
  );
};

export default Header;
