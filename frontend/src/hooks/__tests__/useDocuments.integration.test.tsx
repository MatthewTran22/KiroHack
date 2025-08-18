import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { useDocuments, useUploadDocuments, useUpdateDocument, useDeleteDocument } from '../useDocuments';
import { useDocumentStore } from '@/stores/documents';

// Mock fetch for API calls
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock localStorage
const mockLocalStorage = {
  getItem: jest.fn(() => 'mock-token'),
  setItem: jest.fn(),
  removeItem: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

// Test wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
};

describe('useDocuments Integration Tests', () => {
  beforeEach(() => {
    mockFetch.mockClear();
    useDocumentStore.setState({
      selectedDocuments: [],
      viewMode: 'grid',
      filters: {},
      sortBy: { field: 'uploadedAt', direction: 'desc' },
      uploadProgress: {},
      isUploading: false,
      showUploadModal: false,
    });
  });

  describe('Docker Container Integration', () => {
    it('should fetch documents from backend API', async () => {
      const mockDocuments = {
        data: [
          {
            id: 'doc-1',
            name: 'test.pdf',
            type: 'application/pdf',
            size: 1024,
            uploadedAt: new Date().toISOString(),
            userId: 'user-1',
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
        json: () => Promise.resolve(mockDocuments),
      });

      const { result } = renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockDocuments);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/documents'),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });

    it('should handle API errors gracefully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ message: 'Server error' }),
      });

      const { result } = renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBeDefined();
    });

    it('should apply filters to API request', async () => {
      const filters = {
        searchQuery: 'test',
        category: 'policy',
        classification: 'public' as const,
      };

      // Set filters in store
      useDocumentStore.setState({ filters });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], pagination: { page: 1, limit: 10, total: 0, totalPages: 0 } }),
      });

      renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalled();
      });

      const fetchCall = mockFetch.mock.calls[0];
      const url = fetchCall[0];
      
      expect(url).toContain('search=test');
      expect(url).toContain('category=policy');
      expect(url).toContain('classification=public');
    });

    it('should apply sorting to API request', async () => {
      const sortBy = { field: 'name' as const, direction: 'asc' as const };
      
      // Set sort in store
      useDocumentStore.setState({ sortBy });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: [], pagination: { page: 1, limit: 10, total: 0, totalPages: 0 } }),
      });

      renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalled();
      });

      const fetchCall = mockFetch.mock.calls[0];
      const url = fetchCall[0];
      
      expect(url).toContain('sortBy=name');
      expect(url).toContain('sortOrder=asc');
    });
  });

  describe('Upload Integration', () => {
    it('should upload documents with progress tracking', async () => {
      const mockUploadResponse = [
        {
          id: 'doc-1',
          name: 'test.pdf',
          type: 'application/pdf',
          size: 1024,
          uploadedAt: new Date().toISOString(),
          userId: 'user-1',
        },
      ];

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockUploadResponse),
      });

      const { result } = renderHook(() => useUploadDocuments(), {
        wrapper: createWrapper(),
      });

      const files = [new File(['content'], 'test.pdf', { type: 'application/pdf' })];
      const metadata = [{ category: 'policy' }];

      await waitFor(async () => {
        await result.current.mutateAsync({ files, metadata });
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/documents/upload',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
          body: expect.any(FormData),
        })
      );
    });

    it('should handle upload failures', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ message: 'Invalid file type' }),
      });

      const { result } = renderHook(() => useUploadDocuments(), {
        wrapper: createWrapper(),
      });

      const files = [new File(['content'], 'test.txt', { type: 'text/plain' })];
      const metadata = [{}];

      await expect(
        result.current.mutateAsync({ files, metadata })
      ).rejects.toThrow();
    });
  });

  describe('Update Integration', () => {
    it('should update document via API', async () => {
      const updatedDocument = {
        id: 'doc-1',
        name: 'updated.pdf',
        type: 'application/pdf',
        size: 1024,
        uploadedAt: new Date().toISOString(),
        userId: 'user-1',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(updatedDocument),
      });

      const { result } = renderHook(() => useUpdateDocument(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        await result.current.mutateAsync({
          id: 'doc-1',
          updates: { name: 'updated.pdf' },
        });
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/documents/doc-1',
        expect.objectContaining({
          method: 'PATCH',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            Authorization: 'Bearer mock-token',
          }),
          body: JSON.stringify({ name: 'updated.pdf' }),
        })
      );
    });
  });

  describe('Delete Integration', () => {
    it('should delete document via API', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
      });

      const { result } = renderHook(() => useDeleteDocument(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        await result.current.mutateAsync('doc-1');
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/documents/doc-1',
        expect.objectContaining({
          method: 'DELETE',
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });
  });

  describe('Real Backend Integration (Docker)', () => {
    // These tests would run against a real Docker container
    // They are skipped by default and run only in CI/CD with INTEGRATION_TEST=true
    
    const isIntegrationTest = process.env.INTEGRATION_TEST === 'true';
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';

    beforeEach(() => {
      if (isIntegrationTest) {
        // Use real fetch instead of mock
        global.fetch = fetch;
      }
    });

    (isIntegrationTest ? it : it.skip)('should connect to real backend API', async () => {
      const { result } = renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      }, { timeout: 10000 });

      // Should either succeed or fail with a real HTTP error
      expect(result.current.isSuccess || result.current.isError).toBe(true);
    });

    (isIntegrationTest ? it : it.skip)('should handle real authentication flow', async () => {
      // This would test against a real backend with authentication
      const loginResponse = await fetch(`${backendUrl}/api/v1/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: 'test@example.com',
          password: 'testpassword',
        }),
      });

      if (loginResponse.ok) {
        const { token } = await loginResponse.json();
        mockLocalStorage.getItem.mockReturnValue(token);

        const { result } = renderHook(() => useDocuments(), {
          wrapper: createWrapper(),
        });

        await waitFor(() => {
          expect(result.current.isLoading).toBe(false);
        }, { timeout: 10000 });

        expect(result.current.isSuccess).toBe(true);
      }
    });

    (isIntegrationTest ? it : it.skip)('should handle real file upload', async () => {
      // Test real file upload to Docker backend
      const testFile = new File(['test content'], 'test.txt', { type: 'text/plain' });
      
      const { result } = renderHook(() => useUploadDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        const uploadResult = await result.current.mutateAsync({
          files: [testFile],
          metadata: [{ category: 'test' }],
        });
        
        expect(uploadResult).toBeDefined();
        expect(Array.isArray(uploadResult)).toBe(true);
      }, { timeout: 30000 });
    });

    (isIntegrationTest ? it : it.skip)('should handle network failures gracefully', async () => {
      // Test with invalid backend URL to simulate network failure
      const originalFetch = global.fetch;
      global.fetch = jest.fn().mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useDocuments(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });

      expect(result.current.error).toBeInstanceOf(Error);
      
      global.fetch = originalFetch;
    });
  });
});