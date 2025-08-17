import { act, renderHook } from '@testing-library/react';
import { useAuthStore } from '../auth';
import { setupAuthMocks, cleanupAuthMocks, mockFetch, mockAPIResponses, mockUser } from '../../test/auth-test-utils';
import { tokenManager } from '../../lib/auth';

// Mock the API client
jest.mock('../../lib/api', () => ({
  apiClient: {
    login: jest.fn(),
    logout: jest.fn(),
    refreshToken: jest.fn(),
    getCurrentUser: jest.fn(),
  },
}));

import { apiClient } from '../../lib/api';

describe('useAuthStore', () => {
  beforeEach(() => {
    setupAuthMocks();
    // Reset store state
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
      loginAttempts: 0,
      isLocked: false,
    });
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const { result } = renderHook(() => useAuthStore());

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
      expect(result.current.loginAttempts).toBe(0);
      expect(result.current.isLocked).toBe(false);
    });
  });

  describe('login', () => {
    it('should successfully login with valid credentials', async () => {
      (apiClient.login as jest.Mock).mockResolvedValueOnce(mockAPIResponses.login.success);
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.login({
          email: 'test@example.com',
          password: 'password123',
        });
      });

      expect(result.current.user).toEqual(mockUser);
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBeNull();
      expect(result.current.loginAttempts).toBe(0);
    });

    it('should handle login failure and increment attempts', async () => {
      const error = new Error('Invalid credentials');
      (apiClient.login as jest.Mock).mockRejectedValueOnce(error);
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        try {
          await result.current.login({
            email: 'test@example.com',
            password: 'wrongpassword',
          });
        } catch (e) {
          // Expected to throw
        }
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.isLoading).toBe(false);
      expect(result.current.error).toBe('Invalid credentials');
      expect(result.current.loginAttempts).toBe(1);
      expect(result.current.isLocked).toBe(false);
    });

    it('should lock account after 3 failed attempts', async () => {
      const error = new Error('Invalid credentials');
      (apiClient.login as jest.Mock).mockRejectedValue(error);
      
      const { result } = renderHook(() => useAuthStore());

      // First two attempts
      for (let i = 0; i < 2; i++) {
        await act(async () => {
          try {
            await result.current.login({
              email: 'test@example.com',
              password: 'wrongpassword',
            });
          } catch (e) {
            // Expected to throw
          }
        });
      }

      expect(result.current.loginAttempts).toBe(2);
      expect(result.current.isLocked).toBe(false);

      // Third attempt should lock the account
      await act(async () => {
        try {
          await result.current.login({
            email: 'test@example.com',
            password: 'wrongpassword',
          });
        } catch (e) {
          // Expected to throw
        }
      });

      expect(result.current.loginAttempts).toBe(3);
      expect(result.current.isLocked).toBe(true);
    });

    it('should prevent login when account is locked', async () => {
      const { result } = renderHook(() => useAuthStore());

      // Set account as locked
      act(() => {
        useAuthStore.setState({ isLocked: true });
      });

      await act(async () => {
        await result.current.login({
          email: 'test@example.com',
          password: 'password123',
        });
      });

      expect(result.current.error).toBe('Account is temporarily locked due to too many failed attempts');
      expect(apiClient.login).not.toHaveBeenCalled();
    });
  });

  describe('logout', () => {
    it('should successfully logout', async () => {
      (apiClient.logout as jest.Mock).mockResolvedValueOnce(undefined);
      
      const { result } = renderHook(() => useAuthStore());

      // Set initial authenticated state
      act(() => {
        useAuthStore.setState({
          user: mockUser,
          isAuthenticated: true,
        });
      });

      await act(async () => {
        await result.current.logout();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.isLoading).toBe(false);
      expect(apiClient.logout).toHaveBeenCalled();
    });

    it('should clear state even if logout API call fails', async () => {
      (apiClient.logout as jest.Mock).mockRejectedValueOnce(new Error('Network error'));
      
      const { result } = renderHook(() => useAuthStore());

      // Set initial authenticated state
      act(() => {
        useAuthStore.setState({
          user: mockUser,
          isAuthenticated: true,
        });
      });

      await act(async () => {
        await result.current.logout();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.isLoading).toBe(false);
    });
  });

  describe('refreshToken', () => {
    it('should successfully refresh token', async () => {
      (apiClient.refreshToken as jest.Mock).mockResolvedValueOnce(mockAPIResponses.refreshToken.success);
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.refreshToken();
      });

      expect(result.current.user).toEqual(mockUser);
      expect(result.current.isAuthenticated).toBe(true);
      expect(apiClient.refreshToken).toHaveBeenCalled();
    });

    it('should logout user if refresh token fails', async () => {
      const error = new Error('Invalid refresh token');
      (apiClient.refreshToken as jest.Mock).mockRejectedValueOnce(error);
      (apiClient.logout as jest.Mock).mockResolvedValueOnce(undefined);
      
      const { result } = renderHook(() => useAuthStore());

      // Set initial authenticated state
      act(() => {
        useAuthStore.setState({
          user: mockUser,
          isAuthenticated: true,
        });
      });

      await act(async () => {
        try {
          await result.current.refreshToken();
        } catch (e) {
          // Expected to throw
        }
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
    });
  });

  describe('getCurrentUser', () => {
    it('should successfully get current user', async () => {
      (apiClient.getCurrentUser as jest.Mock).mockResolvedValueOnce(mockUser);
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(true);
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.getCurrentUser();
      });

      expect(result.current.user).toEqual(mockUser);
      expect(result.current.isAuthenticated).toBe(true);
      expect(result.current.isLoading).toBe(false);
    });

    it('should clear auth state if token is invalid', async () => {
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(false);
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.getCurrentUser();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(apiClient.getCurrentUser).not.toHaveBeenCalled();
    });

    it('should handle API error and clear tokens', async () => {
      const error = new Error('Unauthorized');
      (apiClient.getCurrentUser as jest.Mock).mockRejectedValueOnce(error);
      jest.spyOn(tokenManager, 'isTokenValid').mockReturnValue(true);
      jest.spyOn(tokenManager, 'clearTokens').mockImplementation();
      
      const { result } = renderHook(() => useAuthStore());

      await act(async () => {
        await result.current.getCurrentUser();
      });

      expect(result.current.user).toBeNull();
      expect(result.current.isAuthenticated).toBe(false);
      expect(result.current.error).toBe('Unauthorized');
      expect(tokenManager.clearTokens).toHaveBeenCalled();
    });
  });

  describe('utility actions', () => {
    it('should clear error', () => {
      const { result } = renderHook(() => useAuthStore());

      act(() => {
        useAuthStore.setState({ error: 'Some error' });
      });

      expect(result.current.error).toBe('Some error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });

    it('should reset login attempts', () => {
      const { result } = renderHook(() => useAuthStore());

      act(() => {
        useAuthStore.setState({ loginAttempts: 3, isLocked: true });
      });

      expect(result.current.loginAttempts).toBe(3);
      expect(result.current.isLocked).toBe(true);

      act(() => {
        result.current.resetLoginAttempts();
      });

      expect(result.current.loginAttempts).toBe(0);
      expect(result.current.isLocked).toBe(false);
    });

    it('should set loading state', () => {
      const { result } = renderHook(() => useAuthStore());

      expect(result.current.isLoading).toBe(false);

      act(() => {
        result.current.setLoading(true);
      });

      expect(result.current.isLoading).toBe(true);

      act(() => {
        result.current.setLoading(false);
      });

      expect(result.current.isLoading).toBe(false);
    });
  });
});