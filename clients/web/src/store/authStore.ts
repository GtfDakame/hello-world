import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  isAuthenticated: boolean;
  user: {
    id: string;
    phone?: string;
    email?: string;
    displayName?: string;
    avatarUrl?: string;
  } | null;
  token: string | null;
  refreshToken: string | null;
  login: (token: string, refreshToken: string, user: AuthState['user']) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      isAuthenticated: false,
      user: null,
      token: null,
      refreshToken: null,
      
      login: (token, refreshToken, user) => set({
        isAuthenticated: true,
        token,
        refreshToken,
        user,
      }),
      
      logout: () => set({
        isAuthenticated: false,
        user: null,
        token: null,
        refreshToken: null,
      }),
    }),
    {
      name: 'auth-storage',
    }
  )
);
