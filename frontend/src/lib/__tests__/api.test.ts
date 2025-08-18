import { apiClient, APIError } from '../api';
import { tokenManager } from '../auth';

// Mock the auth module
jest.mock('../auth', () => ({
  tokenManager: {
    getToken: jest.fn(),
    getRefreshToken: jest.fn(),
    setTokens: jest.fn(),
    clearTokens: jest.fn(),
  },
}));

// Mock fetch
global.fetch = jest.fn();

const mockFetch = fetch as jest.MockedFunction<typeof fetch>;
const mockTokenManager = tokenManager as jest.Mocked<typeof tokenManager>;

describe('API Client', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockTokenManager.getToken.mockReturnValue('mock-token');
    mockTokenManager.getRefreshToken.mockReturnValue('mock-refresh-token');
  });

  describe('Authentication API', () => {
    it('should login successfully', async () => {
      const mockResponse = {
        user: { id: '1', email: 'test@example.com', name: 'Test User', role: 'user' as const, mfaEnabled: false, createdAt: new Date(), updatedAt: new Date() },
        token: 'new-token',
        refreshToken: 'new-refresh-token',
        expiresAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.auth.login({
        email: 'test@example.com',
        password: 'password',
      });

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/login',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
          body: JSON.stringify({
            email: 'test@example.com',
            password: 'password',
          }),
        })
      );
    });

    it('should handle login failure with proper error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: async () => ({
          message: 'Invalid credentials',
          code: 'invalid_credentials',
          timestamp: new Date(),
        }),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await expect(
        apiClient.auth.login({
          email: 'test@example.com',
          password: 'wrong-password',
        })
      ).rejects.toThrow(APIError);
    });

    it('should refresh token successfully', async () => {
      const mockResponse = {
        user: { id: '1', email: 'test@example.com', name: 'Test User', role: 'user' as const, mfaEnabled: false, createdAt: new Date(), updatedAt: new Date() },
        token: 'new-token',
        refreshToken: 'new-refresh-token',
        expiresAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.auth.refreshToken();

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/refresh',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ refreshToken: 'mock-refresh-token' }),
        })
      );
    });

    it('should logout and clear tokens', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await apiClient.auth.logout();

      expect(mockTokenManager.clearTokens).toHaveBeenCalled();
    });

    it('should get current user', async () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user' as const,
        mfaEnabled: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockUser,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.auth.getCurrentUser();

      expect(result).toEqual(mockUser);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/me',
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });
  });

  describe('Documents API', () => {
    it('should upload documents successfully', async () => {
      const mockDocuments = [
        {
          id: '1',
          name: 'test.pdf',
          type: 'application/pdf',
          size: 1024,
          uploadedAt: new Date(),
          userId: '1',
          status: 'completed' as const,
          tags: ['test'],
          metadata: { title: 'Test Document' },
        },
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockDocuments,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const file = new File(['test content'], 'test.pdf', { type: 'application/pdf' });
      const result = await apiClient.documents.upload([
        {
          file,
          metadata: { title: 'Test Document' },
          tags: ['test'],
        },
      ]);

      expect(result).toEqual(mockDocuments);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/documents/upload',
        expect.objectContaining({
          method: 'POST',
          body: expect.any(FormData),
        })
      );
    });

    it('should get documents with filters', async () => {
      const mockResponse = {
        data: [
          {
            id: '1',
            name: 'test.pdf',
            type: 'application/pdf',
            size: 1024,
            uploadedAt: new Date(),
            userId: '1',
            status: 'completed' as const,
            tags: ['test'],
            metadata: { title: 'Test Document' },
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.documents.getDocuments({
        category: 'policy',
        tags: ['test'],
      });

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/documents?'),
        expect.any(Object)
      );
    });

    it('should search documents', async () => {
      const mockResponse = {
        data: [
          {
            id: '1',
            name: 'test.pdf',
            type: 'application/pdf',
            size: 1024,
            uploadedAt: new Date(),
            userId: '1',
            status: 'completed' as const,
            tags: ['test'],
            metadata: { title: 'Test Document' },
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.documents.searchDocuments('test query');

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/documents/search?q=test'),
        expect.any(Object)
      );
    });

    it('should delete document', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await apiClient.documents.deleteDocument('1');

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/documents/1',
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });

  describe('Consultations API', () => {
    it('should create consultation session', async () => {
      const mockConsultation = {
        id: '1',
        title: 'Test Consultation',
        type: 'policy' as const,
        status: 'active' as const,
        createdAt: new Date(),
        updatedAt: new Date(),
        userId: '1',
        messages: [],
        priority: 'medium' as const,
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockConsultation,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.consultations.createSession({
        type: 'policy',
        title: 'Test Consultation',
        priority: 'medium',
      });

      expect(result).toEqual(mockConsultation);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/consultations',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            type: 'policy',
            title: 'Test Consultation',
            priority: 'medium',
          }),
        })
      );
    });

    it('should send message to consultation', async () => {
      const mockMessage = {
        id: '1',
        sessionId: '1',
        type: 'user' as const,
        content: 'Test message',
        timestamp: new Date(),
        inputMethod: 'text' as const,
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockMessage,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.consultations.sendMessage('1', {
        content: 'Test message',
        inputMethod: 'text',
      });

      expect(result).toEqual(mockMessage);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/consultations/1/messages',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            content: 'Test message',
            inputMethod: 'text',
          }),
        })
      );
    });

    it('should get consultation sessions with filters', async () => {
      const mockResponse = {
        data: [
          {
            id: '1',
            title: 'Test Consultation',
            type: 'policy' as const,
            status: 'active' as const,
            createdAt: new Date(),
            updatedAt: new Date(),
            userId: '1',
            messages: [],
            priority: 'medium' as const,
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.consultations.getSessions({
        type: 'policy',
        status: 'active',
      });

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/consultations?'),
        expect.any(Object)
      );
    });
  });

  describe('Error Handling and Retry Logic', () => {
    it('should handle network errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      await expect(apiClient.auth.getCurrentUser()).rejects.toThrow(APIError);
    });

    it('should handle timeout errors', async () => {
      mockFetch.mockImplementationOnce(() => 
        new Promise((_, reject) => {
          setTimeout(() => reject(new Error('AbortError')), 100);
        })
      );

      await expect(
        apiClient.auth.getCurrentUser()
      ).rejects.toThrow(APIError);
    });

    it('should retry on server errors', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 500,
          json: async () => ({
            message: 'Internal server error',
            code: 'server_error',
            timestamp: new Date(),
          }),
          headers: new Headers({ 'content-type': 'application/json' }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ id: '1', email: 'test@example.com' }),
          headers: new Headers({ 'content-type': 'application/json' }),
        } as Response);

      const result = await apiClient.auth.getCurrentUser();

      expect(result).toEqual({ id: '1', email: 'test@example.com' });
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('should not retry on client errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({
          message: 'Bad request',
          code: 'validation_error',
          timestamp: new Date(),
        }),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await expect(apiClient.auth.getCurrentUser()).rejects.toThrow(APIError);
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('should handle token refresh on 401 errors', async () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user' as const,
        mfaEnabled: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      // First call returns 401
      mockFetch
        .mockResolvedValueOnce({
          ok: false,
          status: 401,
          json: async () => ({
            message: 'Token expired',
            code: 'token_expired',
            timestamp: new Date(),
          }),
          headers: new Headers({ 'content-type': 'application/json' }),
        } as Response)
        // Refresh token call
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user: mockUser,
            token: 'new-token',
            refreshToken: 'new-refresh-token',
            expiresAt: new Date(),
          }),
          headers: new Headers({ 'content-type': 'application/json' }),
        } as Response);

      // The interceptor should trigger token refresh
      await expect(apiClient.auth.getCurrentUser()).rejects.toThrow(APIError);
      expect(mockFetch).toHaveBeenCalledTimes(2);
    });

    it('should include authorization header when token is available', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: '1', email: 'test@example.com' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await apiClient.auth.getCurrentUser();

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/me',
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });

    it('should skip auth header when skipAuth is true', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ token: 'new-token' }),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await apiClient.auth.login({
        email: 'test@example.com',
        password: 'password',
      });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/auth/login',
        expect.objectContaining({
          headers: expect.not.objectContaining({
            Authorization: expect.any(String),
          }),
        })
      );
    });
  });

  describe('Users API', () => {
    it('should get users with filters', async () => {
      const mockResponse = {
        data: [
          {
            id: '1',
            email: 'test@example.com',
            name: 'Test User',
            role: 'user' as const,
            mfaEnabled: false,
            createdAt: new Date(),
            updatedAt: new Date(),
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.users.getUsers({
        role: 'user',
        department: 'IT',
      });

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/users?'),
        expect.any(Object)
      );
    });

    it('should create user', async () => {
      const mockUser = {
        id: '1',
        email: 'new@example.com',
        name: 'New User',
        role: 'user' as const,
        mfaEnabled: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockUser,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.users.createUser({
        name: 'New User',
        email: 'new@example.com',
        password: 'password123',
        role: 'user',
      });

      expect(result).toEqual(mockUser);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/users',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            name: 'New User',
            email: 'new@example.com',
            password: 'password123',
            role: 'user',
          }),
        })
      );
    });
  });

  describe('Audit API', () => {
    it('should get audit logs with filters', async () => {
      const mockResponse = {
        data: [
          {
            id: '1',
            userId: '1',
            action: 'login',
            resource: 'auth',
            details: {},
            timestamp: new Date(),
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.audit.getLogs({
        userId: '1',
        action: 'login',
      });

      expect(result).toEqual(mockResponse);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/audit?'),
        expect.any(Object)
      );
    });

    it('should export audit logs', async () => {
      const mockBlob = new Blob(['csv data'], { type: 'text/csv' });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: async () => mockBlob,
      } as Response);

      const result = await apiClient.audit.exportLogs({}, 'csv');

      expect(result).toEqual(mockBlob);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/audit/export?format=csv'),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });
  });

  describe('Legacy Methods', () => {
    it('should maintain backward compatibility for login', async () => {
      const mockResponse = {
        user: { id: '1', email: 'test@example.com', name: 'Test User', role: 'user' as const, mfaEnabled: false, createdAt: new Date(), updatedAt: new Date() },
        token: 'new-token',
        refreshToken: 'new-refresh-token',
        expiresAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.login({
        email: 'test@example.com',
        password: 'password',
      });

      expect(result).toEqual(mockResponse);
    });

    it('should maintain backward compatibility for logout', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => ({}),
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      await apiClient.logout();

      expect(mockTokenManager.clearTokens).toHaveBeenCalled();
    });

    it('should maintain backward compatibility for getCurrentUser', async () => {
      const mockUser = {
        id: '1',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user' as const,
        mfaEnabled: false,
        createdAt: new Date(),
        updatedAt: new Date(),
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockUser,
        headers: new Headers({ 'content-type': 'application/json' }),
      } as Response);

      const result = await apiClient.getCurrentUser();

      expect(result).toEqual(mockUser);
    });
  });
});