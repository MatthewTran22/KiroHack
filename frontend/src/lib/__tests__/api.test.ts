import { apiClient, APIError } from '../api';
import { setupAuthMocks, cleanupAuthMocks, mockFetch, mockAPIResponses, mockJWTToken } from '../../test/auth-test-utils';
import { tokenManager } from '../auth';

describe('APIClient', () => {
  beforeEach(() => {
    setupAuthMocks();
    mockFetch.mockClear();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  describe('login', () => {
    it('should successfully login with valid credentials', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.login.success,
      });

      const credentials = {
        email: 'test@example.com',
        password: 'password123',
      };

      const result = await apiClient.login(credentials);

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/login',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify(credentials),
        })
      );

      expect(result).toEqual(mockAPIResponses.login.success);
    });

    it('should throw APIError for invalid credentials', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => mockAPIResponses.login.invalidCredentials,
      });

      const credentials = {
        email: 'test@example.com',
        password: 'wrongpassword',
      };

      await expect(apiClient.login(credentials)).rejects.toThrow(APIError);
    });

    it('should handle MFA required response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => mockAPIResponses.login.mfaRequired,
      });

      const credentials = {
        email: 'test@example.com',
        password: 'password123',
      };

      try {
        await apiClient.login(credentials);
      } catch (error) {
        expect(error).toBeInstanceOf(APIError);
        expect((error as APIError).code).toBe('MFA_REQUIRED');
      }
    });
  });

  describe('logout', () => {
    it('should successfully logout', async () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
      });

      await apiClient.logout();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/logout',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ refreshToken: 'refresh-token' }),
        })
      );
    });

    it('should clear tokens even if logout request fails', async () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await apiClient.logout();

      expect(tokenManager.getToken()).toBeNull();
      expect(tokenManager.getRefreshToken()).toBeNull();
    });
  });

  describe('refreshToken', () => {
    it('should successfully refresh token', async () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.refreshToken.success,
      });

      const result = await apiClient.refreshToken();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/refresh',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ refreshToken: 'refresh-token' }),
        })
      );

      expect(result).toEqual(mockAPIResponses.refreshToken.success);
    });

    it('should throw error when no refresh token is available', async () => {
      tokenManager.clearTokens();

      await expect(apiClient.refreshToken()).rejects.toThrow(APIError);
      await expect(apiClient.refreshToken()).rejects.toThrow('No refresh token available');
    });
  });

  describe('getCurrentUser', () => {
    it('should successfully get current user', async () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.getCurrentUser.success,
      });

      const result = await apiClient.getCurrentUser();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/me',
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': `Bearer ${mockJWTToken}`,
          }),
        })
      );

      expect(result).toEqual(mockAPIResponses.getCurrentUser.success);
    });

    it('should include authorization header when token is available', async () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.getCurrentUser.success,
      });

      await apiClient.getCurrentUser();

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': `Bearer ${mockJWTToken}`,
          }),
        })
      );
    });
  });

  describe('MFA endpoints', () => {
    beforeEach(() => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
    });

    it('should setup MFA', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.mfaSetup.success,
      });

      const result = await apiClient.setupMFA();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/mfa/setup',
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result).toEqual(mockAPIResponses.mfaSetup.success);
    });

    it('should verify MFA code', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.mfaVerify.success,
      });

      const result = await apiClient.verifyMFA('123456');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/mfa/verify',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ code: '123456' }),
        })
      );

      expect(result).toEqual(mockAPIResponses.mfaVerify.success);
    });

    it('should disable MFA', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockAPIResponses.mfaVerify.success,
      });

      const result = await apiClient.disableMFA('123456');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/mfa/disable',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ code: '123456' }),
        })
      );

      expect(result).toEqual(mockAPIResponses.mfaVerify.success);
    });
  });

  describe('error handling', () => {
    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(apiClient.getCurrentUser()).rejects.toThrow(APIError);
      await expect(apiClient.getCurrentUser()).rejects.toThrow('Network error');
    });

    it('should handle HTTP errors with JSON response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({
          message: 'Bad request',
          code: 'VALIDATION_ERROR',
          details: { field: 'email' },
        }),
      });

      try {
        await apiClient.getCurrentUser();
      } catch (error) {
        expect(error).toBeInstanceOf(APIError);
        expect((error as APIError).status).toBe(400);
        expect((error as APIError).code).toBe('VALIDATION_ERROR');
        expect((error as APIError).details).toEqual({ field: 'email' });
      }
    });

    it('should handle HTTP errors without JSON response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      try {
        await apiClient.getCurrentUser();
      } catch (error) {
        expect(error).toBeInstanceOf(APIError);
        expect((error as APIError).status).toBe(500);
        expect((error as APIError).message).toBe('HTTP 500');
      }
    });
  });
});