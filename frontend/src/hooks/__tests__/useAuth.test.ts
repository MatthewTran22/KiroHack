import { renderHook, act } from '@testing-library/react';
import { useAuth, useRequireAuth, useRedirectIfAuthenticated } from '../useAuth';
import { useAuthStore } from '../../stores/auth';
import { setupAuthMocks, cleanupAuthMocks } from '../../test/auth-test-utils';
import { tokenManager } from '../../lib/auth';

// Mock next/navigation
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

// Mock the auth store
jest.mock('../../stores/auth');

describe('useAuth', () => {
  const mockAuthStore = {
    user: null,
    isAuthenticated: false,
    isLoading: false,
    error: null,
    loginAttempts: 0,
    isLocked: false,
    login: jest.fn(),
    logout: jest.fn(),
    refreshToken: jest.fn(),
    getCurrentUser: jest.fn(),
    clearError: jest.fn(),
    resetLoginAttempts: jest.fn(),
  };

  beforeEach(() => {
    setupAuthMocks();
    (useAuthStore as jest.Mock).mockReturnValue(mockAuthStore);
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  describe('useAuth hook', () => {
    it('should return auth state and actions', () => {
      const { result } = renderHook(() => useAuth());

      expect(result.current).toEqual({
        user: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,
        loginAttempts: 0,
        isLocked: false,
        login: expect.any(Function),
        logout: expect.any(Function),
        clearError: expect.any(Function),
        resetLoginAttempts: expect.any(Function),
      });
    });

    it('should initialize auth state on mount when token is valid', async () => {
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(true);
      
      renderHook(() => useAuth());

      // Wait for useEffect to run
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      expect(mockAuthStore.getCurrentUser).toHaveBeenCalled();
    });

    it('should not initialize auth state when token is invalid', async () => {
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(false);
      
      renderHook(() => useAuth());

      // Wait for useEffect to run
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      expect(mockAuthStore.getCurrentUser).not.toHaveBeenCalled();
    });

    it('should handle initialization error gracefully', async () => {
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(true);
      mockAuthStore.getCurrentUser.mockRejectedValueOnce(new Error('Network error'));
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      renderHook(() => useAuth());

      // Wait for useEffect to run
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      expect(consoleSpy).toHaveBeenCalledWith('Failed to initialize auth:', expect.any(Error));
      consoleSpy.mockRestore();
    });

    it('should set up token refresh interval when authenticated', async () => {
      jest.useFakeTimers();
      jest.spyOn(tokenManager, 'shouldRefreshToken').mockReturnValue(true);
      
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      renderHook(() => useAuth());

      // Fast-forward time to trigger interval
      act(() => {
        jest.advanceTimersByTime(60 * 1000); // 1 minute
      });

      expect(authenticatedStore.refreshToken).toHaveBeenCalled();

      jest.useRealTimers();
    });

    it('should handle token refresh error gracefully', async () => {
      jest.useFakeTimers();
      jest.spyOn(tokenManager, 'shouldRefreshToken').mockReturnValue(true);
      mockAuthStore.refreshToken.mockRejectedValueOnce(new Error('Refresh failed'));
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
      
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      renderHook(() => useAuth());

      // Fast-forward time to trigger interval
      await act(async () => {
        jest.advanceTimersByTime(60 * 1000); // 1 minute
      });

      expect(consoleSpy).toHaveBeenCalledWith('Token refresh failed:', expect.any(Error));
      consoleSpy.mockRestore();
      jest.useRealTimers();
    });
  });

  describe('useRequireAuth hook', () => {
    it('should redirect to login when not authenticated', () => {
      const unauthenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: false,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(unauthenticatedStore);

      renderHook(() => useRequireAuth());

      expect(mockPush).toHaveBeenCalledWith('/login');
    });

    it('should not redirect when authenticated', () => {
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      renderHook(() => useRequireAuth());

      expect(mockPush).not.toHaveBeenCalled();
    });

    it('should not redirect when loading', () => {
      const loadingStore = {
        ...mockAuthStore,
        isAuthenticated: false,
        isLoading: true,
      };
      (useAuthStore as jest.Mock).mockReturnValue(loadingStore);

      renderHook(() => useRequireAuth());

      expect(mockPush).not.toHaveBeenCalled();
    });

    it('should return auth state', () => {
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      const { result } = renderHook(() => useRequireAuth());

      expect(result.current).toEqual({
        isAuthenticated: true,
        isLoading: false,
      });
    });
  });

  describe('useRedirectIfAuthenticated hook', () => {
    it('should redirect to dashboard when authenticated', () => {
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      renderHook(() => useRedirectIfAuthenticated());

      expect(mockPush).toHaveBeenCalledWith('/dashboard');
    });

    it('should not redirect when not authenticated', () => {
      const unauthenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: false,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(unauthenticatedStore);

      renderHook(() => useRedirectIfAuthenticated());

      expect(mockPush).not.toHaveBeenCalled();
    });

    it('should not redirect when loading', () => {
      const loadingStore = {
        ...mockAuthStore,
        isAuthenticated: true,
        isLoading: true,
      };
      (useAuthStore as jest.Mock).mockReturnValue(loadingStore);

      renderHook(() => useRedirectIfAuthenticated());

      expect(mockPush).not.toHaveBeenCalled();
    });

    it('should return auth state', () => {
      const authenticatedStore = {
        ...mockAuthStore,
        isAuthenticated: true,
        isLoading: false,
      };
      (useAuthStore as jest.Mock).mockReturnValue(authenticatedStore);

      const { result } = renderHook(() => useRedirectIfAuthenticated());

      expect(result.current).toEqual({
        isAuthenticated: true,
        isLoading: false,
      });
    });
  });
});