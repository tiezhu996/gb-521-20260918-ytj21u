import { useEffect } from 'react';
import { Alert, Button, Form, Input } from 'antd';
import { LockKeyhole, Mail, ShieldCheck, Wind } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../hooks/useAuth';
import { reportError } from '../utils/errors';

export function LoginPage() {
  const { user, busy, login } = useAuth();
  const navigate = useNavigate();
  useEffect(() => { if (user) navigate('/network', { replace: true }); }, [navigate, user]);
  const submit = async (values: { email: string; password: string }) => {
    try {
      await login(values.email, values.password);
      navigate('/network', { replace: true });
    } catch (error) {
      reportError(error, '登录未完成');
    }
  };
  return (
    <main className="login-page">
      <section className="login-identity">
        <div className="login-wordmark"><span><Wind size={25} /></span>VentLock 风网联锁推演台</div>
        <div className="login-title"><small>离线工程工作台 / MVNS 01</small><h1>复核每一条风路，<br />留下每一次决策证据。</h1></div>
        <div className="login-boundary"><ShieldCheck size={21} /><p><strong>系统边界</strong><br />本系统不连接 PLC、风机或风门。计算结果仅用于离线比较，不能替代现场规程与人工批准。</p></div>
      </section>
      <section className="login-panel" aria-labelledby="login-heading">
        <div className="login-form-wrap">
          <span className="page-eyebrow">授权访问</span>
          <h2 id="login-heading">进入推演工作台</h2>
          <p>使用分配给你的工程、复核或管理账号。</p>
          <Form layout="vertical" initialValues={{ email: 'engineer@mine.local', password: 'Ventilate!2026' }} onFinish={submit} requiredMark={false}>
            <Form.Item name="email" label="工作邮箱" rules={[{ required: true, message: '请输入工作邮箱' }, { type: 'email', message: '邮箱格式需要包含 @' }]}>
              <Input prefix={<Mail size={17} />} autoComplete="username" />
            </Form.Item>
            <Form.Item name="password" label="密码" rules={[{ required: true, message: '请输入密码' }]}>
              <Input.Password prefix={<LockKeyhole size={17} />} autoComplete="current-password" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={busy} block>登录工作台</Button>
          </Form>
          <Alert className="login-note" type="info" showIcon message="账号角色决定审批与风险确认权限" />
        </div>
      </section>
    </main>
  );
}
