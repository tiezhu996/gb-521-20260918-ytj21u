import { create } from 'zustand';
import * as authApi from '../api/auth';
import { getStoredToken, setStoredToken } from '../api/client';
import type { AuthUser } from '../types/auth';

interface AuthState {
  user: AuthUser | null;
  ready: boolean;
  busy: boolean;
  login(email: string, password: string): Promise<void>;
  restore(): Promise<void>;
  logout(): void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  ready: false,
  busy: false,
  async login(email, password) {
    set({ busy: true });
    try {
      const result = await authApi.login(email, password);
      setStoredToken(result.token);
      set({ user: result.user, ready: true });
    } finally {
      set({ busy: false });
    }
  },
  async restore() {
    if (!getStoredToken()) {
      set({ ready: true, user: null });
      return;
    }
    try {
      set({ user: await authApi.getMe() });
    } catch {
      setStoredToken(null);
      set({ user: null });
    } finally {
      set({ ready: true });
    }
  },
  logout() {
    setStoredToken(null);
    set({ user: null, ready: true });
  },
}));
