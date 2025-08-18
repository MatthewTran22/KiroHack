import {
  LoginCredentials,
  AuthResponse,
  User,
  MFASetupResponse,
  Document,
  DocumentFilters,
  DocumentUploadRequest,
  Consultation,
  ConsultationFilters,
  ConsultationRequest,
  MessageRequest,
  ConsultationMessage,
  UserFilters,
  UserUpdateRequest,
  UserCreateRequest,
  AuditLog,
  AuditFilters,
  PaginatedResponse,
  APIErrorResponse,
  RequestConfig,
  RetryConfig,
} from '@/types';
import { API_BASE_URL } from './constants';
import { tokenManager } from './auth';

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public details?: Record<string, unknown>,
    public requestId?: string
  ) {
    super(message);
    this.name = 'APIError';
  }

  static fromResponse(response: APIErrorResponse, status: number): APIError {
    return new APIError(
      response.message,
      status,
      response.code,
      response.details,
      response.requestId
    );
  }

  get isNetworkError(): boolean {
    return this.status === 0;
  }

  get isAuthError(): boolean {
    return this.status === 401 || this.status === 403;
  }

  get isServerError(): boolean {
    return this.status >= 500;
  }

  get isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  get isRetryable(): boolean {
    return this.isNetworkError || this.isServerError || this.status === 429;
  }
}

interface RequestInterceptor {
  (config: RequestConfig): RequestConfig | Promise<RequestConfig>;
}

interface ResponseInterceptor {
  onFulfilled?: (response: Response) => Response | Promise<Response>;
  onRejected?: (error: APIError) => APIError | Promise<APIError>;
}

class APIClient {
  private baseURL: string;
  private defaultRetryConfig: RetryConfig = {
    maxAttempts: 3,
    baseDelay: 1000,
    maxDelay: 10000,
    backoffFactor: 2,
    retryableStatuses: [408, 429, 500, 502, 503, 504],
  };
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL;
    this.setupDefaultInterceptors();
  }

  private setupDefaultInterceptors(): void {
    // Request interceptor for authentication
    this.addRequestInterceptor((config) => {
      if (!config.skipAuth) {
        const token = tokenManager.getToken();
        if (token) {
          config.headers = {
            ...config.headers,
            Authorization: `Bearer ${token}`,
          };
        }
      }
      return config;
    });

    // Response interceptor for token refresh
    this.addResponseInterceptor({
      onRejected: async (error) => {
        if (error.status === 401 && !error.code?.includes('invalid_credentials')) {
          try {
            await this.refreshToken();
            // Retry the original request
            throw new APIError('Token refreshed, retry needed', 401, 'token_refreshed');
          } catch (refreshError) {
            // Refresh failed, redirect to login
            tokenManager.clearTokens();
            if (typeof window !== 'undefined' && process.env.NODE_ENV !== 'test') {
              window.location.href = '/login';
            }
            throw error;
          }
        }
        return error;
      },
    });
  }

  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  private async sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  private calculateDelay(attempt: number, config: RetryConfig): number {
    const delay = config.baseDelay * Math.pow(config.backoffFactor, attempt - 1);
    return Math.min(delay, config.maxDelay);
  }

  private async executeWithRetry<T>(
    operation: () => Promise<T>,
    retryConfig: RetryConfig
  ): Promise<T> {
    let lastError: APIError;

    for (let attempt = 1; attempt <= retryConfig.maxAttempts; attempt++) {
      try {
        return await operation();
      } catch (error) {
        lastError = error instanceof APIError ? error : new APIError('Unknown error', 0);

        const shouldRetry =
          attempt < retryConfig.maxAttempts &&
          (lastError.isRetryable || retryConfig.retryableStatuses.includes(lastError.status));

        if (!shouldRetry) {
          throw lastError;
        }

        const delay = this.calculateDelay(attempt, retryConfig);
        await this.sleep(delay);
      }
    }

    throw lastError!;
  }

  private async request<T>(
    endpoint: string,
    options: RequestConfig = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    
    // Apply request interceptors
    let config = { ...options };
    for (const interceptor of this.requestInterceptors) {
      config = await interceptor(config);
    }

    // Set default headers
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...config.headers,
    };

    // Remove custom properties from config
    const { timeout = 30000, retry, skipAuth, skipRetry, ...fetchConfig } = config;
    
    const requestConfig: RequestInit = {
      ...fetchConfig,
      headers,
    };

    const retryConfig = { ...this.defaultRetryConfig, ...retry };

    const operation = async (): Promise<T> => {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);

      try {
        const response = await fetch(url, {
          ...requestConfig,
          signal: controller.signal,
        });

        clearTimeout(timeoutId);

        // Apply response interceptors
        let processedResponse = response;
        for (const interceptor of this.responseInterceptors) {
          if (interceptor.onFulfilled) {
            processedResponse = await interceptor.onFulfilled(processedResponse);
          }
        }

        if (!processedResponse.ok) {
          let errorData: APIErrorResponse;
          try {
            errorData = await processedResponse.json();
          } catch {
            errorData = {
              message: `HTTP ${processedResponse.status}: ${processedResponse.statusText}`,
              timestamp: new Date(),
            };
          }

          let error = APIError.fromResponse(errorData, processedResponse.status);

          // Apply error interceptors
          for (const interceptor of this.responseInterceptors) {
            if (interceptor.onRejected) {
              error = await interceptor.onRejected(error);
            }
          }

          throw error;
        }

        // Handle empty responses
        const contentType = processedResponse.headers.get('content-type');
        if (!contentType?.includes('application/json')) {
          return {} as T;
        }

        return await processedResponse.json();
      } catch (error) {
        clearTimeout(timeoutId);
        
        if (error instanceof APIError) {
          throw error;
        }

        if (error instanceof Error) {
          if (error.name === 'AbortError') {
            throw new APIError('Request timeout', 408, 'timeout');
          }
          throw new APIError(error.message, 0, 'network_error');
        }

        throw new APIError('Unknown error', 0, 'unknown_error');
      }
    };

    if (skipRetry) {
      return operation();
    }

    return this.executeWithRetry(operation, retryConfig);
  }

  // Authentication API
  auth = {
    login: async (credentials: LoginCredentials): Promise<AuthResponse> => {
      // Filter out rememberMe as backend doesn't expect it
      const { rememberMe, ...loginData } = credentials;
      return this.request<AuthResponse>('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify(loginData),
        skipAuth: true,
      });
    },

    logout: async (): Promise<void> => {
      const refreshToken = tokenManager.getRefreshToken();
      if (refreshToken) {
        try {
          await this.request('/api/v1/auth/logout', {
            method: 'POST',
            body: JSON.stringify({ refreshToken }),
          });
        } catch (error) {
          console.error('Logout request failed:', error);
        }
      }
      tokenManager.clearTokens();
    },

    refreshToken: async (): Promise<AuthResponse> => {
      const refreshToken = tokenManager.getRefreshToken();
      if (!refreshToken) {
        throw new APIError('No refresh token available', 401, 'no_refresh_token');
      }

      return this.request<AuthResponse>('/api/v1/auth/refresh', {
        method: 'POST',
        body: JSON.stringify({ refreshToken }),
        skipAuth: true,
      });
    },

    getCurrentUser: async (): Promise<User> => {
      const response = await this.request<{ user: User }>('/api/v1/auth/profile');
      return response.user;
    },

    setupMFA: async (): Promise<MFASetupResponse> => {
      return this.request<MFASetupResponse>('/api/v1/auth/mfa/setup', {
        method: 'POST',
      });
    },

    verifyMFA: async (code: string): Promise<{ success: boolean }> => {
      return this.request<{ success: boolean }>('/api/v1/auth/mfa/verify', {
        method: 'POST',
        body: JSON.stringify({ code }),
      });
    },

    disableMFA: async (currentPassword: string): Promise<{ success: boolean }> => {
      return this.request<{ success: boolean }>('/api/v1/auth/mfa/disable', {
        method: 'POST',
        body: JSON.stringify({ current_password: currentPassword }),
      });
    },
  };

  // Documents API
  documents = {
    upload: async (requests: DocumentUploadRequest[]): Promise<Document[]> => {
      const formData = new FormData();
      
      requests.forEach((request, index) => {
        formData.append(`files`, request.file);
        formData.append(`metadata_${index}`, JSON.stringify(request.metadata));
        if (request.classification) {
          formData.append(`classification_${index}`, request.classification);
        }
        if (request.tags) {
          formData.append(`tags_${index}`, JSON.stringify(request.tags));
        }
      });

      return this.request<Document[]>('/api/v1/documents/upload', {
        method: 'POST',
        body: formData,
        headers: {}, // Let browser set Content-Type for FormData
      });
    },

    getDocuments: async (filters?: DocumentFilters): Promise<PaginatedResponse<Document>> => {
      const params = new URLSearchParams();
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else if (Array.isArray(value)) {
              value.forEach(item => params.append(key, item));
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      const queryString = params.toString();
      const endpoint = queryString ? `/api/v1/documents?${queryString}` : '/api/v1/documents';
      
      return this.request<PaginatedResponse<Document>>(endpoint);
    },

    getDocument: async (id: string): Promise<Document> => {
      return this.request<Document>(`/api/v1/documents/${id}`);
    },

    updateDocument: async (id: string, updates: Partial<Document>): Promise<Document> => {
      return this.request<Document>(`/api/v1/documents/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(updates),
      });
    },

    deleteDocument: async (id: string): Promise<void> => {
      return this.request<void>(`/api/v1/documents/${id}`, {
        method: 'DELETE',
      });
    },

    deleteDocuments: async (ids: string[]): Promise<void> => {
      return this.request<void>('/api/v1/documents/batch', {
        method: 'DELETE',
        body: JSON.stringify({ ids }),
      });
    },

    searchDocuments: async (query: string, filters?: DocumentFilters): Promise<PaginatedResponse<Document>> => {
      const params = new URLSearchParams({ q: query });
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else if (Array.isArray(value)) {
              value.forEach(item => params.append(key, item));
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      return this.request<PaginatedResponse<Document>>(`/api/v1/documents/search?${params.toString()}`);
    },

    downloadDocument: async (id: string): Promise<Blob> => {
      const response = await fetch(`${this.baseURL}/api/v1/documents/${id}/download`, {
        headers: {
          Authorization: `Bearer ${tokenManager.getToken()}`,
        },
      });

      if (!response.ok) {
        throw new APIError('Download failed', response.status);
      }

      return response.blob();
    },
  };

  // Consultations API
  consultations = {
    createSession: async (request: ConsultationRequest): Promise<Consultation> => {
      return this.request<Consultation>('/api/v1/consultations', {
        method: 'POST',
        body: JSON.stringify(request),
      });
    },

    getSessions: async (filters?: ConsultationFilters): Promise<PaginatedResponse<Consultation>> => {
      const params = new URLSearchParams();
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else if (Array.isArray(value)) {
              value.forEach(item => params.append(key, item));
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      const queryString = params.toString();
      const endpoint = queryString ? `/api/v1/consultations?${queryString}` : '/api/v1/consultations';
      
      return this.request<PaginatedResponse<Consultation>>(endpoint);
    },

    getSession: async (sessionId: string): Promise<Consultation> => {
      return this.request<Consultation>(`/api/v1/consultations/${sessionId}`);
    },

    updateSession: async (sessionId: string, updates: Partial<Consultation>): Promise<Consultation> => {
      return this.request<Consultation>(`/api/v1/consultations/${sessionId}`, {
        method: 'PATCH',
        body: JSON.stringify(updates),
      });
    },

    deleteSession: async (sessionId: string): Promise<void> => {
      return this.request<void>(`/api/v1/consultations/${sessionId}`, {
        method: 'DELETE',
      });
    },

    sendMessage: async (sessionId: string, request: MessageRequest): Promise<ConsultationMessage> => {
      return this.request<ConsultationMessage>(`/api/v1/consultations/${sessionId}/messages`, {
        method: 'POST',
        body: JSON.stringify(request),
      });
    },

    getMessages: async (sessionId: string): Promise<ConsultationMessage[]> => {
      return this.request<ConsultationMessage[]>(`/api/v1/consultations/${sessionId}/messages`);
    },

    exportSession: async (sessionId: string, format: 'pdf' | 'docx' | 'json' = 'pdf'): Promise<Blob> => {
      const response = await fetch(`${this.baseURL}/api/v1/consultations/${sessionId}/export?format=${format}`, {
        headers: {
          Authorization: `Bearer ${tokenManager.getToken()}`,
        },
      });

      if (!response.ok) {
        throw new APIError('Export failed', response.status);
      }

      return response.blob();
    },

    searchSessions: async (query: string, filters?: ConsultationFilters): Promise<PaginatedResponse<Consultation>> => {
      const params = new URLSearchParams({ q: query });
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else if (Array.isArray(value)) {
              value.forEach(item => params.append(key, item));
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      return this.request<PaginatedResponse<Consultation>>(`/api/v1/consultations/search?${params.toString()}`);
    },
  };

  // Users API (Admin only)
  users = {
    getUsers: async (filters?: UserFilters): Promise<PaginatedResponse<User>> => {
      const params = new URLSearchParams();
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            params.append(key, String(value));
          }
        });
      }

      const queryString = params.toString();
      const endpoint = queryString ? `/api/v1/users?${queryString}` : '/api/v1/users';
      
      return this.request<PaginatedResponse<User>>(endpoint);
    },

    getUser: async (userId: string): Promise<User> => {
      return this.request<User>(`/api/v1/users/${userId}`);
    },

    createUser: async (request: UserCreateRequest): Promise<User> => {
      return this.request<User>('/api/v1/users', {
        method: 'POST',
        body: JSON.stringify(request),
      });
    },

    updateUser: async (userId: string, updates: UserUpdateRequest): Promise<User> => {
      return this.request<User>(`/api/v1/users/${userId}`, {
        method: 'PATCH',
        body: JSON.stringify(updates),
      });
    },

    deleteUser: async (userId: string): Promise<void> => {
      return this.request<void>(`/api/v1/users/${userId}`, {
        method: 'DELETE',
      });
    },

    searchUsers: async (query: string, filters?: UserFilters): Promise<PaginatedResponse<User>> => {
      const params = new URLSearchParams({ q: query });
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            params.append(key, String(value));
          }
        });
      }

      return this.request<PaginatedResponse<User>>(`/api/v1/users/search?${params.toString()}`);
    },
  };

  // Audit API
  audit = {
    getLogs: async (filters?: AuditFilters): Promise<PaginatedResponse<AuditLog>> => {
      const params = new URLSearchParams();
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      const queryString = params.toString();
      const endpoint = queryString ? `/api/v1/audit?${queryString}` : '/api/v1/audit';
      
      return this.request<PaginatedResponse<AuditLog>>(endpoint);
    },

    exportLogs: async (filters?: AuditFilters, format: 'csv' | 'json' = 'csv'): Promise<Blob> => {
      const params = new URLSearchParams({ format });
      
      if (filters) {
        Object.entries(filters).forEach(([key, value]) => {
          if (value !== undefined && value !== null) {
            if (typeof value === 'object' && 'start' in value && 'end' in value) {
              params.append(`${key}_start`, value.start.toISOString());
              params.append(`${key}_end`, value.end.toISOString());
            } else {
              params.append(key, String(value));
            }
          }
        });
      }

      const response = await fetch(`${this.baseURL}/api/v1/audit/export?${params.toString()}`, {
        headers: {
          Authorization: `Bearer ${tokenManager.getToken()}`,
        },
      });

      if (!response.ok) {
        throw new APIError('Export failed', response.status);
      }

      return response.blob();
    },
  };

  // Legacy methods for backward compatibility
  async login(credentials: LoginCredentials): Promise<AuthResponse> {
    return this.auth.login(credentials);
  }

  async logout(): Promise<void> {
    return this.auth.logout();
  }

  async refreshToken(): Promise<AuthResponse> {
    return this.auth.refreshToken();
  }

  async getCurrentUser(): Promise<User> {
    return this.auth.getCurrentUser();
  }

  async setupMFA(): Promise<MFASetupResponse> {
    return this.auth.setupMFA();
  }

  async verifyMFA(code: string): Promise<{ success: boolean }> {
    return this.auth.verifyMFA(code);
  }

  async disableMFA(currentPassword: string): Promise<{ success: boolean }> {
    return this.auth.disableMFA(currentPassword);
  }
}

export const apiClient = new APIClient();