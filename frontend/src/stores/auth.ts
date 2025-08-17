import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { User, LoginCredentials, AuthResponse } from '@/types';
import { apiClient } from '@/lib/api';
import { tokenManager } from '@/lib/auth';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  loginAttempts: number;
  isLocked: boolean;
}

interface AuthActions {
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<void>;
  getCurrentUser: () => Promise<void>;
  checkAuth: () => Promise<void>;
  clearError: () => void;
  resetLoginAttempts: () => void;
  setLoading: (loading: boolean) => void;
}

type AuthStore = AuthState & AuthActions;

const initialState: AuthState = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
  error: null,
  loginAttempts: 0,
  isLocked: false,
};

export const useAuthStore = create<AuthStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      login: async (credentials: LoginCredentials) => {
        const { loginAttempts, isLocked } = get();
        
        if (isLocked) {
          set({ error: 'Account is temporarily locked due to too many failed attempts' });
          return;
        }

        set({ isLoading: true, error: null });

        try {
          const response: AuthResponse = await apiClient.login(credentials);
          
          tokenManager.setTokens(response.token, response.refreshToken);
          
          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            loginAttempts: 0,
            error: null,
          });
        } catch (error: any) {
          const newAttempts = loginAttempts + 1;
          const shouldLock = newAttempts >= 3;
          
          set({
            isLoading: false,
            error: error.message || 'Login failed',
            loginAttempts: newAttempts,
            isLocked: shouldLock,
          });

          if (shouldLock) {
            // Auto-unlock after 15 minutes
            setTimeout(() => {
              set({ isLocked: false, loginAttempts: 0 });
            }, 15 * 60 * 1000);
          }

          throw error;
        }
      },

      logout: async () => {
        set({ isLoading: true });
        
        try {
          await apiClient.logout();
        } catch (error) {
          console.error('Logout error:', error);
        } finally {
          tokenManager.clearTokens();
          set({
            ...initialState,
            isLoading: false,
          });
        }
      },

      refreshToken: async () => {
        try {
          const response: AuthResponse = await apiClient.refreshToken();
          tokenManager.setTokens(response.token, response.refreshToken);
          
          set({
            user: response.user,
            isAuthenticated: true,
          });
        } catch (error) {
          // If refresh fails, logout the user
          get().logout();
          throw error;
        }
      },

      getCurrentUser: async () => {
        if (!tokenManager.isTokenValid()) {
          set({ isAuthenticated: false, user: null });
          return;
        }

        set({ isLoading: true });

        try {
          const user = await apiClient.getCurrentUser();
          set({
            user,
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (error: any) {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: error.message,
          });
          tokenManager.clearTokens();
        }
      },

      checkAuth: async () => {
        const token = tokenManager.getToken();
        if (!token) {
          set({ isAuthenticated: false, user: null });
          return;
        }

        if (tokenManager.isTokenValid()) {
          try {
            await get().getCurrentUser();
          } catch (error) {
            // getCurrentUser already handles the error state
          }
        } else {
          // Try to refresh the token
          try {
            await get().refreshToken();
          } catch (error) {
            set({ isAuthenticated: false, user: null });
          }
        }
      },

      clearError: () => set({ error: null }),
      
      resetLoginAttempts: () => set({ loginAttempts: 0, isLocked: false }),
      
      setLoading: (loading: boolean) => set({ isLoading: loading }),
    }),
    {
      name: 'auth-store',
    }
  )
);