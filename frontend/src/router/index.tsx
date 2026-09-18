import { lazy, Suspense, type ReactNode } from 'react';
import { Spin } from 'antd';
import { Navigate, createBrowserRouter } from 'react-router-dom';
import { AppShell } from '../App';
import { useAuthStore } from '../stores/authStore';
import { getStoredToken } from '../api/client';

const LoginPage = lazy(() => import('../pages/LoginPage').then((module) => ({ default: module.LoginPage })));
const NetworkPage = lazy(() => import('../pages/NetworkPage').then((module) => ({ default: module.NetworkPage })));
const ScenariosPage = lazy(() => import('../pages/ScenariosPage').then((module) => ({ default: module.ScenariosPage })));
const SimulationsPage = lazy(() => import('../pages/SimulationsPage').then((module) => ({ default: module.SimulationsPage })));
const InterlocksPage = lazy(() => import('../pages/InterlocksPage').then((module) => ({ default: module.InterlocksPage })));
const AuditPage = lazy(() => import('../pages/AuditPage').then((module) => ({ default: module.AuditPage })));

function DeferredPage({ children }: { children: ReactNode }) {
  return <Suspense fallback={<div className="route-loading" role="status"><Spin /><span>正在加载工作区</span></div>}>{children}</Suspense>;
}

function RequireAuth() {
  const user = useAuthStore((state) => state.user);
  return user && getStoredToken() ? <AppShell /> : <Navigate to="/login" replace />;
}

function RequireAuditRole() {
  const role = useAuthStore((state) => state.user?.role);
  return role === 'reviewer' || role === 'admin' ? <DeferredPage><AuditPage /></DeferredPage> : <Navigate to="/network" replace />;
}

export const router = createBrowserRouter([
  { path: '/login', element: <DeferredPage><LoginPage /></DeferredPage> },
  {
    path: '/', element: <RequireAuth />,
    children: [
      { index: true, element: <Navigate to="/network" replace /> },
      { path: 'network', element: <DeferredPage><NetworkPage /></DeferredPage> },
      { path: 'scenarios', element: <DeferredPage><ScenariosPage /></DeferredPage> },
      { path: 'simulations', element: <DeferredPage><SimulationsPage /></DeferredPage> },
      { path: 'interlocks', element: <DeferredPage><InterlocksPage /></DeferredPage> },
      { path: 'audit', element: <RequireAuditRole /> },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
]);
