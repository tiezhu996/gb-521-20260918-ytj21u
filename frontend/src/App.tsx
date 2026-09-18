import { useEffect, useState } from 'react';
import { Button, Drawer, Spin } from 'antd';
import { Activity, ClipboardCheck, FileClock, GitBranch, LogOut, Menu, ShieldAlert, Wind } from 'lucide-react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useAuth } from './hooks/useAuth';
import { roleLabel } from './utils/format';

const navItems = [
  { to: '/network', label: '通风网络', Icon: GitBranch },
  { to: '/scenarios', label: '风机方案', Icon: ClipboardCheck },
  { to: '/simulations', label: '推演工作台', Icon: Activity },
  { to: '/interlocks', label: '联锁风险', Icon: ShieldAlert },
  { to: '/audit', label: '审计追踪', Icon: FileClock, restricted: true },
];

function Navigation({ close }: { close?: () => void }) {
  const { user } = useAuth();
  return <nav className="primary-nav" aria-label="主要导航">{navItems.filter((item) => !item.restricted || Boolean(user && user.role !== 'engineer')).map(({ to, label, Icon }) => <NavLink key={to} to={to} onClick={close}><Icon size={18} aria-hidden="true" /><span>{label}</span></NavLink>)}</nav>;
}

export function AppShell() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const signOut = () => {
    logout();
    navigate('/login', { replace: true });
  };
  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">跳到主要内容</a>
      <aside className="sidebar">
        <div className="brand"><span className="brand-mark"><Wind size={22} /></span><span><strong>VentLock 风网联锁推演台</strong><small>离线决策支持</small></span></div>
        <Navigation />
        <div className="sidebar-boundary"><ShieldAlert size={17} /><span>不连接现场控制系统<br />结果须经人工复核</span></div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <Button className="mobile-menu" type="text" icon={<Menu size={21} />} aria-label="打开导航" onClick={() => setDrawerOpen(true)} />
          <div className="system-state"><i />离线模型环境</div>
          <div className="operator"><span><strong>{user?.display_name}</strong><small>{user ? roleLabel[user.role] : ''}</small></span><Button type="text" icon={<LogOut size={17} />} aria-label="退出登录" title="退出登录" onClick={signOut} /></div>
        </header>
        <main id="main-content" className="main-content"><Outlet /></main>
      </div>
      <Drawer className="mobile-drawer" placement="left" width={280} open={drawerOpen} onClose={() => setDrawerOpen(false)} title="VentLock 风网联锁推演台"><Navigation close={() => setDrawerOpen(false)} /></Drawer>
    </div>
  );
}

export function AuthBootstrap({ children }: { children: React.ReactNode }) {
  const { ready, restore } = useAuth();
  useEffect(() => { void restore(); }, [restore]);
  if (!ready) return <div className="boot-screen"><Spin size="large" /><span>正在校验工作台会话</span></div>;
  return children;
}
