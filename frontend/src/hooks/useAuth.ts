import { useMemo } from 'react';
import { useAuthStore } from '../stores/authStore';
import type { UserRole } from '../types/auth';

export function useAuth() {
  const state = useAuthStore();
  return useMemo(() => ({
    ...state,
    hasRole: (...roles: UserRole[]) => state.user ? roles.includes(state.user.role) : false,
  }), [state]);
}
