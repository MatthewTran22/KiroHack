// Application constants

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const ROUTES = {
  HOME: '/',
  LOGIN: '/login',
  DASHBOARD: '/dashboard',
  DOCUMENTS: '/documents',
  CONSULTATIONS: '/consultations',
  HISTORY: '/history',
  AUDIT: '/audit',
} as const;

export const CONSULTATION_TYPES = {
  POLICY: 'policy',
  RESEARCH: 'research',
  COMPLIANCE: 'compliance',
} as const;

export const USER_ROLES = {
  ADMIN: 'admin',
  USER: 'user',
} as const;
