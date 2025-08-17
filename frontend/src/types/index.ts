// Common type definitions for the application

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'user';
  department?: string;
  mfaEnabled: boolean;
  createdAt: Date;
  updatedAt: Date;
}

// Authentication types
export interface LoginCredentials {
  email: string;
  password: string;
  mfaCode?: string;
  rememberMe?: boolean;
}

export interface AuthResponse {
  user: User;
  token: string;
  refreshToken: string;
  expiresAt: Date;
}

export interface MFASetupResponse {
  qrCode: string;
  secret: string;
  backupCodes: string[];
}

export interface TokenPayload {
  sub: string;
  email: string;
  role: string;
  exp: number;
  iat: number;
}

export interface Document {
  id: string;
  name: string;
  type: string;
  size: number;
  uploadedAt: Date;
  userId: string;
}

export interface Consultation {
  id: string;
  title: string;
  type: 'policy' | 'research' | 'compliance';
  status: 'active' | 'completed' | 'draft';
  createdAt: Date;
  updatedAt: Date;
  userId: string;
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
  success: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

// Layout and UI types
export interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message: string;
  timestamp: Date;
  read: boolean;
  actions?: NotificationAction[];
}

export interface NotificationAction {
  label: string;
  action: () => void;
  variant?: 'default' | 'destructive';
}

export interface NavigationItem {
  id: string;
  label: string;
  href: string;
  icon?: string;
  badge?: string | number;
  children?: NavigationItem[];
  requiredRole?: User['role'];
}

export interface LayoutState {
  sidebarOpen: boolean;
  user: User | null;
  notifications: Notification[];
  theme: 'light' | 'dark' | 'system';
}

export interface SearchResult {
  id: string;
  title: string;
  type: 'document' | 'consultation' | 'knowledge';
  excerpt: string;
  url: string;
  relevance: number;
}

