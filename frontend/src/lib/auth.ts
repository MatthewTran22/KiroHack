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
      this.token = localStorage.getItem(TOKEN_STORAGE_KEY);
      this.refreshToken = localStorage.getItem(REFRESH_TOKEN_STORAGE_KEY);
    }
  }

  setTokens(token: string, refreshToken: string): void {
    this.token = token;
    this.refreshToken = refreshToken;
    
    if (typeof window !== 'undefined') {
      localStorage.setItem(TOKEN_STORAGE_KEY, token);
      localStorage.setItem(REFRESH_TOKEN_STORAGE_KEY, refreshToken);
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
      localStorage.removeItem(TOKEN_STORAGE_KEY);
      localStorage.removeItem(REFRESH_TOKEN_STORAGE_KEY);
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