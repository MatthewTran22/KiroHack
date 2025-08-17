import { NextRequest } from 'next/server';
import { middleware } from '../middleware';
import { setupAuthMocks, cleanupAuthMocks, mockJWTToken, mockExpiredJWTToken } from '../test/auth-test-utils';

// Mock next/server
jest.mock('next/server', () => ({
  NextResponse: {
    next: jest.fn(() => ({
      headers: {
        set: jest.fn(),
      },
    })),
    redirect: jest.fn((url) => ({
      headers: {
        set: jest.fn(),
      },
      url: url.toString(),
    })),
  },
}));

import { NextResponse } from 'next/server';

describe('Middleware', () => {
  beforeEach(() => {
    setupAuthMocks();
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  const createRequest = (pathname: string, token?: string) => {
    const url = `http://localhost:3000${pathname}`;
    const request = {
      nextUrl: { pathname },
      url,
      cookies: {
        get: jest.fn((name) => {
          if (name === 'auth_token' && token) {
            return { value: token };
          }
          return undefined;
        }),
      },
      headers: {
        get: jest.fn((name) => {
          if (name === 'authorization' && token) {
            return `Bearer ${token}`;
          }
          return null;
        }),
      },
    } as unknown as NextRequest;

    return request;
  };

  describe('Protected Routes', () => {
    it('should redirect to login for unauthenticated user on protected route', () => {
      const request = createRequest('/dashboard');
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/login',
          search: '?redirect=%2Fdashboard',
        })
      );
    });

    it('should allow access to protected route with valid token', () => {
      const request = createRequest('/dashboard', mockJWTToken);
      
      middleware(request);

      expect(NextResponse.redirect).not.toHaveBeenCalled();
      expect(NextResponse.next).toHaveBeenCalled();
    });

    it('should redirect to login with expired token', () => {
      const request = createRequest('/dashboard', mockExpiredJWTToken);
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/login',
          search: '?redirect=%2Fdashboard',
        })
      );
    });

    it('should handle all protected routes', () => {
      const protectedRoutes = ['/dashboard', '/documents', '/consultations', '/history', '/audit', '/mfa-setup'];
      
      protectedRoutes.forEach(route => {
        jest.clearAllMocks();
        const request = createRequest(route);
        
        middleware(request);

        expect(NextResponse.redirect).toHaveBeenCalledWith(
          expect.objectContaining({
            pathname: '/login',
          })
        );
      });
    });
  });

  describe('Admin Routes', () => {
    it('should redirect non-admin user from admin route', () => {
      // Mock JWT token with user role
      const userToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwicm9sZSI6InVzZXIiLCJleHAiOjk5OTk5OTk5OTksImlhdCI6MTY0MDk5NTIwMH0.mock-signature';
      const request = createRequest('/audit', userToken);
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/dashboard',
        })
      );
    });

    it('should allow admin user to access admin route', () => {
      // Mock JWT token with admin role
      const adminToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIyIiwiZW1haWwiOiJhZG1pbkBleGFtcGxlLmNvbSIsInJvbGUiOiJhZG1pbiIsImV4cCI6OTk5OTk5OTk5OSwiaWF0IjoxNjQwOTk1MjAwfQ.mock-signature';
      
      // Update the mock to handle admin token
      jest.doMock('jwt-decode', () => ({
        jwtDecode: jest.fn((token: string) => {
          if (token === adminToken) {
            return {
              sub: '2',
              email: 'admin@example.com',
              role: 'admin',
              exp: 9999999999,
              iat: 1640995200,
            };
          }
          return {
            sub: '1',
            email: 'test@example.com',
            role: 'user',
            exp: 9999999999,
            iat: 1640995200,
          };
        }),
      }));

      const request = createRequest('/audit', adminToken);
      
      middleware(request);

      expect(NextResponse.redirect).not.toHaveBeenCalled();
      expect(NextResponse.next).toHaveBeenCalled();
    });
  });

  describe('Auth Routes', () => {
    it('should redirect authenticated user from login page to dashboard', () => {
      const request = createRequest('/login', mockJWTToken);
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/dashboard',
        })
      );
    });

    it('should redirect authenticated user from home page to dashboard', () => {
      const request = createRequest('/', mockJWTToken);
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/dashboard',
        })
      );
    });

    it('should allow unauthenticated user to access login page', () => {
      const request = createRequest('/login');
      
      middleware(request);

      expect(NextResponse.redirect).not.toHaveBeenCalled();
      expect(NextResponse.next).toHaveBeenCalled();
    });
  });

  describe('Security Headers', () => {
    it('should add security headers to all responses', () => {
      const request = createRequest('/dashboard', mockJWTToken);
      const mockResponse = {
        headers: {
          set: jest.fn(),
        },
      };
      
      (NextResponse.next as jest.Mock).mockReturnValue(mockResponse);
      
      middleware(request);

      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-Frame-Options', 'DENY');
      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-Content-Type-Options', 'nosniff');
      expect(mockResponse.headers.set).toHaveBeenCalledWith('Referrer-Policy', 'strict-origin-when-cross-origin');
      expect(mockResponse.headers.set).toHaveBeenCalledWith('X-XSS-Protection', '1; mode=block');
      expect(mockResponse.headers.set).toHaveBeenCalledWith('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
      expect(mockResponse.headers.set).toHaveBeenCalledWith(
        'Strict-Transport-Security',
        'max-age=31536000; includeSubDomains; preload'
      );
      expect(mockResponse.headers.set).toHaveBeenCalledWith(
        'Content-Security-Policy',
        expect.stringContaining("default-src 'self'")
      );
    });
  });

  describe('Token Sources', () => {
    it('should check authorization header when cookie is not present', () => {
      const request = {
        nextUrl: { pathname: '/dashboard' },
        url: 'http://localhost:3000/dashboard',
        cookies: {
          get: jest.fn(() => undefined),
        },
        headers: {
          get: jest.fn((name) => {
            if (name === 'authorization') {
              return `Bearer ${mockJWTToken}`;
            }
            return null;
          }),
        },
      } as unknown as NextRequest;

      middleware(request);

      expect(NextResponse.redirect).not.toHaveBeenCalled();
      expect(NextResponse.next).toHaveBeenCalled();
    });

    it('should handle malformed authorization header', () => {
      const request = {
        nextUrl: { pathname: '/dashboard' },
        url: 'http://localhost:3000/dashboard',
        cookies: {
          get: jest.fn(() => undefined),
        },
        headers: {
          get: jest.fn((name) => {
            if (name === 'authorization') {
              return 'InvalidHeader';
            }
            return null;
          }),
        },
      } as unknown as NextRequest;

      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/login',
        })
      );
    });
  });

  describe('Edge Cases', () => {
    it('should handle invalid JWT tokens gracefully', () => {
      const request = createRequest('/dashboard', 'invalid-token');
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/login',
        })
      );
    });

    it('should handle nested protected routes', () => {
      const request = createRequest('/dashboard/settings');
      
      middleware(request);

      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.objectContaining({
          pathname: '/login',
          search: '?redirect=%2Fdashboard%2Fsettings',
        })
      );
    });

    it('should handle query parameters in redirect', () => {
      const request = {
        nextUrl: { pathname: '/dashboard' },
        url: 'http://localhost:3000/dashboard?tab=settings',
        cookies: {
          get: jest.fn(() => undefined),
        },
        headers: {
          get: jest.fn(() => null),
        },
      } as unknown as NextRequest;

      middleware(request);

      // The middleware creates a URL object, so we check the actual call
      expect(NextResponse.redirect).toHaveBeenCalledWith(
        expect.any(URL)
      );
      
      // Verify the URL contains the redirect parameter
      const redirectCall = (NextResponse.redirect as jest.Mock).mock.calls[0][0];
      expect(redirectCall.toString()).toContain('redirect=');
    });
  });
});