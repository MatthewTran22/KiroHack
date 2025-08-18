// Common type definitions for the application

export interface User {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'analyst' | 'manager' | 'viewer' | 'consultant';
  department?: string;
  permissions?: Array<{
    resource: string;
    actions: string[];
  }>;
  security_clearance?: string;
  mfa_enabled: boolean;  // Changed from mfaEnabled to match backend
  last_login?: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
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
  tokens: {
    access_token: string;
    refresh_token: string;
    expires_at: string;
  };
  message?: string;
  session_id?: string;
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

// Document types
export interface Document {
  id: string;
  name: string;
  type: string;
  size: number;
  uploadedAt: Date;
  userId: string;
  status: 'uploading' | 'processing' | 'completed' | 'error';
  classification?: 'public' | 'internal' | 'confidential' | 'secret';
  tags: string[];
  metadata: DocumentMetadata;
  downloadUrl?: string;
  previewUrl?: string;
  thumbnail?: string;
}

export interface DocumentMetadata {
  title?: string;
  description?: string;
  author?: string;
  department?: string;
  category?: string;
  keywords?: string[];
  language?: string;
  version?: string;
}

export interface DocumentFilters {
  category?: string;
  classification?: Document['classification'];
  tags?: string[];
  dateRange?: {
    start: Date;
    end: Date;
  };
  searchQuery?: string;
  status?: Document['status'];
  userId?: string;
}

export interface DocumentUploadRequest {
  file: File;
  metadata: DocumentMetadata;
  classification?: Document['classification'];
  tags?: string[];
}

// Consultation types
export interface Consultation {
  id: string;
  title: string;
  type: ConsultationType;
  status: ConsultationStatus;
  createdAt: Date;
  updatedAt: Date;
  userId: string;
  messages: ConsultationMessage[];
  context?: string;
  attachedDocuments?: string[];
  summary?: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
}

export type ConsultationType = 'policy' | 'strategy' | 'operations' | 'technology' | 'research' | 'compliance';
export type ConsultationStatus = 'active' | 'completed' | 'draft' | 'archived' | 'paused';

export interface ConsultationMessage {
  id: string;
  sessionId: string;
  type: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  sources?: DocumentReference[];
  confidence?: number;
  metadata?: MessageMetadata;
  inputMethod: 'text' | 'voice';
  audioUrl?: string;
  transcriptionConfidence?: number;
}

export interface DocumentReference {
  id: string;
  title: string;
  excerpt: string;
  confidence: number;
  page?: number;
}

export interface MessageMetadata {
  processingTime?: number;
  modelVersion?: string;
  tokens?: {
    input: number;
    output: number;
  };
}

export interface ConsultationFilters {
  type?: ConsultationType;
  status?: ConsultationStatus;
  priority?: Consultation['priority'];
  dateRange?: {
    start: Date;
    end: Date;
  };
  searchQuery?: string;
  userId?: string;
}

export interface ConsultationRequest {
  type: ConsultationType;
  title?: string;
  context?: string;
  attachedDocuments?: string[];
  priority?: Consultation['priority'];
}

export interface MessageRequest {
  content: string;
  inputMethod?: 'text' | 'voice';
  attachments?: string[];
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

// API Error types
export interface APIErrorResponse {
  message: string;
  code?: string;
  details?: Record<string, unknown>;
  timestamp: Date;
  requestId?: string;
}

export interface RetryConfig {
  maxAttempts: number;
  baseDelay: number;
  maxDelay: number;
  backoffFactor: number;
  retryableStatuses: number[];
}

export interface RequestConfig extends RequestInit {
  timeout?: number;
  retry?: Partial<RetryConfig>;
  skipAuth?: boolean;
  skipRetry?: boolean;
}

// User management types
export interface UserFilters {
  role?: User['role'];
  department?: string;
  searchQuery?: string;
  isActive?: boolean;
}

export interface UserUpdateRequest {
  name?: string;
  email?: string;
  role?: User['role'];
  department?: string;
}

export interface UserCreateRequest {
  name: string;
  email: string;
  password: string;
  role: User['role'];
  department?: string;
}

// Audit types
export interface AuditLog {
  id: string;
  userId: string;
  action: string;
  resource: string;
  resourceId?: string;
  details: Record<string, unknown>;
  timestamp: Date;
  ipAddress?: string;
  userAgent?: string;
}

export interface AuditFilters {
  userId?: string;
  action?: string;
  resource?: string;
  dateRange?: {
    start: Date;
    end: Date;
  };
}

// Enhanced state management types
export interface LoadingState {
  [key: string]: boolean;
}

export interface ErrorState {
  [key: string]: string | null;
}

export interface CacheMetadata {
  timestamp: number;
  ttl: number;
  version: string;
}

export interface OptimisticUpdate<T> {
  id: string;
  queryKey: unknown[];
  previousData: T | undefined;
  optimisticData: T;
  timestamp: number;
}

export interface SyncStatus {
  isOnline: boolean;
  lastSync: Date | null;
  pendingOperations: number;
  failedOperations: number;
}

export interface AppState {
  loading: LoadingState;
  errors: ErrorState;
  sync: SyncStatus;
  cache: {
    metadata: Record<string, CacheMetadata>;
    size: number;
    maxSize: number;
  };
}

