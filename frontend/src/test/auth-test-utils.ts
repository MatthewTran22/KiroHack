import { render, RenderOptions } from '@testing-library/react';
import { ReactElement } from 'react';
import { User, AuthResponse } from '@/types';

// Mock user data
export const mockUser: User = {
  id: '1',
  email: 'test@example.com',
  name: 'Test User',
  role: 'user',
  department: 'IT',
  mfaEnabled: false,
  createdAt: new Date('2024-01-01'),
  updatedAt: new Date('2024-01-01'),
};

export const mockAdminUser: User = {
  ...mockUser,
  id: '2',
  email: 'admin@example.com',
  name: 'Admin User',
  role: 'admin',
  mfaEnabled: true,
};

export const mockAuthResponse: AuthResponse = {
  user: mockUser,
  token: 'mock-jwt-token',
  refreshToken: 'mock-refresh-token',
  expiresAt: new Date(Date.now() + 3600000), // 1 hour from now
};

// Mock JWT token (base64 encoded)
export const mockJWTToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwicm9sZSI6InVzZXIiLCJleHAiOjk5OTk5OTk5OTksImlhdCI6MTY0MDk5NTIwMH0.mock-signature';

export const mockExpiredJWTToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwicm9sZSI6InVzZXIiLCJleHAiOjE2NDA5OTUyMDAsImlhdCI6MTY0MDk5NTIwMH0.mock-signature';

// Mock API responses
export const mockAPIResponses = {
  login: {
    success: mockAuthResponse,
    mfaRequired: {
      message: 'MFA required',
      status: 401,
      code: 'MFA_REQUIRED',
    },
    invalidCredentials: {
      message: 'Invalid credentials',
      status: 401,
      code: 'INVALID_CREDENTIALS',
    },
  },
  getCurrentUser: {
    success: mockUser,
    unauthorized: {
      message: 'Unauthorized',
      status: 401,
    },
  },
  refreshToken: {
    success: mockAuthResponse,
    invalid: {
      message: 'Invalid refresh token',
      status: 401,
    },
  },
  mfaSetup: {
    success: {
      qrCode: 'data:image/png;base64,mock-qr-code',
      secret: 'MOCK-SECRET-KEY',
      backupCodes: ['123456', '789012', '345678', '901234', '567890'],
    },
  },
  mfaVerify: {
    success: { success: true },
    invalid: { success: false },
  },
};

// Custom render function with providers
export function renderWithAuth(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, {
    ...options,
  });
}

// Mock localStorage
export const mockLocalStorage = (() => {
  let store: Record<string, string> = {};

  return {
    getItem: jest.fn((key: string) => store[key] || null),
    setItem: jest.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: jest.fn((key: string) => {
      delete store[key];
    }),
    clear: jest.fn(() => {
      store = {};
    }),
    get store() {
      return store;
    },
  };
})();

// Mock fetch
export const mockFetch = jest.fn();

// Setup mocks
export function setupAuthMocks() {
  // Mock localStorage
  Object.defineProperty(window, 'localStorage', {
    value: mockLocalStorage,
  });

  // Mock fetch
  global.fetch = mockFetch;

  // Mock next/navigation
  jest.mock('next/navigation', () => ({
    useRouter: () => ({
      push: jest.fn(),
      replace: jest.fn(),
      back: jest.fn(),
    }),
    useSearchParams: () => ({
      get: jest.fn(),
    }),
  }));

  // Mock jwt-decode
  jest.mock('jwt-decode', () => ({
    jwtDecode: jest.fn((token: string) => {
      if (token === mockJWTToken) {
        return {
          sub: '1',
          email: 'test@example.com',
          role: 'user',
          exp: 9999999999,
          iat: 1640995200,
        };
      }
      if (token === mockExpiredJWTToken) {
        return {
          sub: '1',
          email: 'test@example.com',
          role: 'user',
          exp: 1640995200,
          iat: 1640995200,
        };
      }
      throw new Error('Invalid token');
    }),
  }));
}

// Cleanup mocks
export function cleanupAuthMocks() {
  jest.clearAllMocks();
  mockLocalStorage.clear();
  mockFetch.mockReset();
}