import { request } from './client';
import type { AuthUser, LoginResult } from '../types/auth';

export const login = (email: string, password: string) =>
  request<LoginResult>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) });

export const getMe = () => request<AuthUser>('/api/v1/auth/me');
