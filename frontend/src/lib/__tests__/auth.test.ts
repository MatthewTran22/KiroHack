import { AuthTokenManager } from '../auth';
import { setupAuthMocks, cleanupAuthMocks, mockJWTToken, mockExpiredJWTToken, mockLocalStorage } from '../../test/auth-test-utils';

describe('AuthTokenManager', () => {
  let tokenManager: AuthTokenManager;

  beforeEach(() => {
    setupAuthMocks();
    // Get a fresh instance for each test
    tokenManager = (AuthTokenManager as any).instance = null;
    tokenManager = AuthTokenManager.getInstance();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  describe('getInstance', () => {
    it('should return a singleton instance', () => {
      const instance1 = AuthTokenManager.getInstance();
      const instance2 = AuthTokenManager.getInstance();
      expect(instance1).toBe(instance2);
    });
  });

  describe('setTokens', () => {
    it('should store tokens in memory and localStorage', () => {
      const token = 'test-token';
      const refreshToken = 'test-refresh-token';

      tokenManager.setTokens(token, refreshToken);

      expect(tokenManager.getToken()).toBe(token);
      expect(tokenManager.getRefreshToken()).toBe(refreshToken);
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith('auth_token', token);
      expect(mockLocalStorage.setItem).toHaveBeenCalledWith('refresh_token', refreshToken);
    });
  });

  describe('clearTokens', () => {
    it('should clear tokens from memory and localStorage', () => {
      tokenManager.setTokens('token', 'refresh-token');
      tokenManager.clearTokens();

      expect(tokenManager.getToken()).toBeNull();
      expect(tokenManager.getRefreshToken()).toBeNull();
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('auth_token');
      expect(mockLocalStorage.removeItem).toHaveBeenCalledWith('refresh_token');
    });
  });

  describe('isTokenValid', () => {
    it('should return false when no token is set', () => {
      expect(tokenManager.isTokenValid()).toBe(false);
    });

    it('should return true for valid token', () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      expect(tokenManager.isTokenValid()).toBe(true);
    });

    it('should return false for expired token', () => {
      tokenManager.setTokens(mockExpiredJWTToken, 'refresh-token');
      expect(tokenManager.isTokenValid()).toBe(false);
    });

    it('should return false for invalid token', () => {
      tokenManager.setTokens('invalid-token', 'refresh-token');
      expect(tokenManager.isTokenValid()).toBe(false);
    });
  });

  describe('shouldRefreshToken', () => {
    it('should return false when no token is set', () => {
      expect(tokenManager.shouldRefreshToken()).toBe(false);
    });

    it('should return false for valid token that does not need refresh', () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      expect(tokenManager.shouldRefreshToken()).toBe(false);
    });

    it('should return true for expired token', () => {
      tokenManager.setTokens(mockExpiredJWTToken, 'refresh-token');
      expect(tokenManager.shouldRefreshToken()).toBe(true);
    });
  });

  describe('getTokenPayload', () => {
    it('should return null when no token is set', () => {
      expect(tokenManager.getTokenPayload()).toBeNull();
    });

    it('should return null for invalid token', () => {
      tokenManager.setTokens('invalid-token', 'refresh-token');
      expect(tokenManager.getTokenPayload()).toBeNull();
    });

    it('should return payload for valid token', () => {
      tokenManager.setTokens(mockJWTToken, 'refresh-token');
      const payload = tokenManager.getTokenPayload();
      
      expect(payload).toEqual({
        sub: '1',
        email: 'test@example.com',
        role: 'user',
        exp: 9999999999,
        iat: 1640995200,
      });
    });
  });

  describe('loadTokensFromStorage', () => {
    it('should load tokens from localStorage on initialization', () => {
      mockLocalStorage.setItem('auth_token', 'stored-token');
      mockLocalStorage.setItem('refresh_token', 'stored-refresh-token');

      // Create new instance to test loading
      (AuthTokenManager as any).instance = null;
      const newTokenManager = AuthTokenManager.getInstance();

      expect(newTokenManager.getToken()).toBe('stored-token');
      expect(newTokenManager.getRefreshToken()).toBe('stored-refresh-token');
    });
  });
});