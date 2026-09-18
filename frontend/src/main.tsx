import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider } from 'antd';
import { RouterProvider } from 'react-router-dom';
import zhCN from 'antd/locale/zh_CN';
import { AuthBootstrap } from './App';
import { router } from './router';
import './styles.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#196b50', colorInfo: '#326b7a', colorSuccess: '#196b50', colorWarning: '#b18416', colorError: '#b44b35', borderRadius: 5, fontFamily: 'Aptos, "Segoe UI Variable", "Noto Sans SC", sans-serif', controlHeight: 40 }, components: { Table: { headerBg: '#e8eeeb', headerColor: '#25302c', rowHoverBg: '#f0f5f2' }, Button: { primaryShadow: 'none' }, Modal: { borderRadiusLG: 6 } } }}>
      <AuthBootstrap><RouterProvider router={router} /></AuthBootstrap>
    </ConfigProvider>
  </React.StrictMode>,
);
