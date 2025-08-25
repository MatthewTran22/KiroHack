import { jwtDecode } from 'jwt-decode';
import { TokenPayload } from '@/types';
import { TOKEN_STORAGE_KEY, REFRESH_TOKEN_STORAGE_KEY, TOKEN_REFRESH_THRESHOLD } from './constants';

export class AuthTokenManager {
  private static instance: AuthTokenManager;
  private token: string | null = null;
  private refreshToken: string | null = null;

  private constructor() {
    this.loadTokensFromStorage();
  }

  static getInstance(): AuthTokenManager {
    if (!AuthTokenManager.instance) {
      AuthTokenManager.instance = new AuthTokenManager();
    }
    return AuthTokenManager.instance;
  }

  private loadTokensFromStorage(): void {
    if (typeof window !== 'undefined') {
      // Try localStorage first
      this.token = localStorage.getItem(TOKEN_STORAGE_KEY);
      this.refreshToken = localStorage.getItem(REFRESH_TOKEN_STORAGE_KEY);

      // If not in localStorage, try cookies
      if (!this.token) {
        this.token = this.getCookie(TOKEN_STORAGE_KEY);
      }
      if (!this.refreshToken) {
        this.refreshToken = this.getCookie(REFRESH_TOKEN_STORAGE_KEY);
      }
    }
  }

  private getCookie(name: string): string | null {
    if (typeof document === 'undefined') return null;

    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) {
      return parts.pop()?.split(';').shift() || null;
    }
    return null;
  }

  setTokens(token: string, refreshToken: string): void {
    this.token = token;
    this.refreshToken = refreshToken;

    if (typeof window !== 'undefined') {
      // Store in localStorage
      localStorage.setItem(TOKEN_STORAGE_KEY, token);
      localStorage.setItem(REFRESH_TOKEN_STORAGE_KEY, refreshToken);

      // Also store in cookies for middleware access
      document.cookie = `${TOKEN_STORAGE_KEY}=${token}; path=/; secure; samesite=strict`;
      document.cookie = `${REFRESH_TOKEN_STORAGE_KEY}=${refreshToken}; path=/; secure; samesite=strict`;
    }
  }

  getToken(): string | null {
    return this.token;
  }

  getRefreshToken(): string | null {
    return this.refreshToken;
  }

  clearTokens(): void {
    this.token = null;
    this.refreshToken = null;

    if (typeof window !== 'undefined') {
      // Clear localStorage
      localStorage.removeItem(TOKEN_STORAGE_KEY);
      localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY);

      // Clear cookies
      document.cookie = `${TOKEN_STORAGE_KEY}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
      document.cookie = `${REFRESH_TOKEN_STORAGE_KEY}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
    }
  }

  isTokenValid(): boolean {
    if (!this.token) return false;

    try {
      const decoded = jwtDecode<TokenPayload>(this.token);
      const now = Date.now() / 1000;
      return decoded.exp > now;
    } catch {
      return false;
    }
  }

  shouldRefreshToken(): boolean {
    if (!this.token) return false;

    try {
      const decoded = jwtDecode<TokenPayload>(this.token);
      const now = Date.now() / 1000;
      const timeUntilExpiry = decoded.exp - now;
      return timeUntilExpiry < TOKEN_REFRESH_THRESHOLD * 60; // Convert minutes to seconds
    } catch {
      return false;
    }
  }

  getTokenPayload(): TokenPayload | null {
    if (!this.token || !this.isTokenValid()) return null;

    try {
      return jwtDecode<TokenPayload>(this.token);
    } catch {
      return null;
    }
  }
}

export const tokenManager = AuthTokenManager.getInstance();