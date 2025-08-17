import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';
import { tokenManager } from '@/lib/auth';
import { ROUTES } from '@/lib/constants';

export function useAuth() {
  const {
    user,
    isAuthenticated,
    isLoading,
    error,
    loginAttempts,
    isLocked,
    login,
    logout,
    refreshToken,
    getCurrentUser,
    clearError,
    resetLoginAttempts,
  } = useAuthStore();

  // Initialize auth state on mount
  useEffect(() => {
    const initializeAuth = async () => {
      if (tokenManager.isTokenValid()) {
        try {
          await getCurrentUser();
        } catch (error) {
          console.error('Failed to initialize auth:', error);
        }
      }
    };

    initializeAuth();
  }, [getCurrentUser]);

  // Auto-refresh token when needed
  useEffect(() => {
    if (!isAuthenticated) return;

    const checkTokenRefresh = async () => {
      if (tokenManager.shouldRefreshToken()) {
        try {
          await refreshToken();
        } catch (error) {
          console.error('Token refresh failed:', error);
        }
      }
    };

    // Check every minute
    const interval = setInterval(checkTokenRefresh, 60 * 1000);
    
    return () => clearInterval(interval);
  }, [isAuthenticated, refreshToken]);

  return {
    user,
    isAuthenticated,
    isLoading,
    error,
    loginAttempts,
    isLocked,
    login,
    logout,
    clearError,
    resetLoginAttempts,
  };
}

export function useRequireAuth() {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push(ROUTES.LOGIN);
    }
  }, [isAuthenticated, isLoading, router]);

  return { isAuthenticated, isLoading };
}

export function useRedirectIfAuthenticated() {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      router.push(ROUTES.DASHBOARD);
    }
  }, [isAuthenticated, isLoading, router]);

  return { isAuthenticated, isLoading };
}