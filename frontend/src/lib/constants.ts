// Application constants

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const ROUTES = {
  HOME: '/',
  LOGIN: '/login',
  LOGOUT: '/logout',
  DASHBOARD: '/dashboard',
  DOCUMENTS: '/documents',
  CONSULTATIONS: '/consultations',
  HISTORY: '/history',
  AUDIT: '/audit',
  MFA_SETUP: '/mfa-setup',
} as const;

// Auth constants
export const TOKEN_STORAGE_KEY = 'auth_token';
export const REFRESH_TOKEN_STORAGE_KEY = 'refresh_token';
export const TOKEN_REFRESH_THRESHOLD = 5; // minutes before expiry
export const MAX_LOGIN_ATTEMPTS = 3;
export const SESSION_TIMEOUT = 60; // minutes

export const CONSULTATION_TYPES = {
  POLICY: 'policy',
  RESEARCH: 'research',
  COMPLIANCE: 'compliance',
} as const;

export const USER_ROLES = {
  ADMIN: 'admin',
  USER: 'user',
} as const;

