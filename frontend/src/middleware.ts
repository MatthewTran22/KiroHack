import { NextRequest, NextResponse } from 'next/server';
import { jwtDecode } from 'jwt-decode';
import { TokenPayload } from '@/types';
import { ROUTES, TOKEN_STORAGE_KEY } from '@/lib/constants';

// Routes that require authentication
const protectedRoutes = [
  '/dashboard',
  '/documents',
  '/consultations',
  '/history',
  '/audit',
  '/mfa-setup',
];

// Routes that should redirect to dashboard if user is authenticated
const authRoutes = ['/login', '/'];

// Admin-only routes
const adminRoutes = ['/audit'];

function isTokenValid(token: string): boolean {
  try {
    const decoded = jwtDecode<TokenPayload>(token);
    const now = Date.now() / 1000;
    return decoded.exp > now;
  } catch {
    return false;
  }
}

function getTokenPayload(token: string): TokenPayload | null {
  try {
    return jwtDecode<TokenPayload>(token);
  } catch {
    return null;
  }
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Get API URL for CSP
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  // Get token from cookie or header
  let token = request.cookies.get(TOKEN_STORAGE_KEY)?.value;

  // If not in cookie, check Authorization header
  if (!token) {
    const authHeader = request.headers.get('authorization');
    if (authHeader?.startsWith('Bearer ')) {
      token = authHeader.substring(7);
    }
  }

  const isAuthenticated = token ? isTokenValid(token) : false;
  const tokenPayload = token ? getTokenPayload(token) : null;

  // Temporarily disable middleware redirects to stop the loop
  // We'll handle authentication purely client-side for now
  // TODO: Re-enable this once we fix the token sync between client and server

  /*
  // Handle protected routes
  if (protectedRoutes.some(route => pathname.startsWith(route))) {
    if (!isAuthenticated) {
      const loginUrl = new URL(ROUTES.LOGIN, request.url);
      loginUrl.searchParams.set('redirect', pathname);
      return NextResponse.redirect(loginUrl);
    }

    // Check admin routes
    if (adminRoutes.some(route => pathname.startsWith(route))) {
      if (tokenPayload?.role !== 'admin') {
        return NextResponse.redirect(new URL(ROUTES.DASHBOARD, request.url));
      }
    }
  }

  // Handle auth routes (redirect to dashboard if already authenticated)
  if (authRoutes.includes(pathname) && isAuthenticated) {
    return NextResponse.redirect(new URL(ROUTES.DASHBOARD, request.url));
  }
  */

  // Add security headers
  const response = NextResponse.next();

  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
  response.headers.set('X-XSS-Protection', '1; mode=block');
  response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=()');
  response.headers.set(
    'Strict-Transport-Security',
    'max-age=31536000; includeSubDomains; preload'
  );
  response.headers.set(
    'Content-Security-Policy',
    `default-src 'self'; script-src 'self' 'unsafe-eval' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' ${apiUrl} ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self';`
  );

  return response;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     */
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};