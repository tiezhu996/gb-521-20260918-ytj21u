export type UserRole = 'engineer' | 'reviewer' | 'admin';

export interface AuthUser {
  id: number;
  email: string;
  display_name: string;
  role: UserRole;
}

export interface LoginResult {
  token: string;
  expires_at: string;
  user: AuthUser;
}
