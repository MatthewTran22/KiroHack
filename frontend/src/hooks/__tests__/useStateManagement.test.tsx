import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { useStateManagement, useApiOperation, useBatchOperation } from '../useStateManagement';
import { useAppStore } from '@/stores/app';
import { useUIStore } from '@/stores/ui';
import { useErrorStore } from '@/stores/error';

// Mock the stores
jest.mock('@/stores/app');
jest.mock('@/stores/ui');
jest.mock('@/stores/error');
jest.mock('../useOfflineSupport');

const mockAppStore = {
  setLoading: jest.fn(),
  clearLoading: jest.fn(),
  isLoading: jest.fn(() => false),
  setError: jest.fn(),
  clearError: jest.fn(),
  hasError: jest.fn(() => false),
  incrementPendingOperations: jest.fn(),
  decrementPendingOperations: jest.fn(),
  updateCacheSize: jest.fn(),
  setCacheMetadata: jest.fn(),
  reset: jest.fn(),
  cache: { size: 0, maxSize: 50 * 1024 * 1024 },
  sync: { isOnline: true, pendingOperations: 0, failedOperations: 0, lastSync: null },
  clearAllErrors: jest.fn(),
  setSyncStatus: jest.fn(),
  resetFailedOperations: jest.fn(),
  getCacheMetadata: jest.fn(),
};

const mockUIStore = {
  addNotification: jest.fn(),
};

const mockErrorStore = {
  addError: jest.fn(),
  clearErrors: jest.fn(),
};

const mockOfflineSupport = {
  isOnline: true,
  queueOfflineOperation: jest.fn(),
  processOfflineQueue: jest.fn(),
  clearOfflineQueue: jest.fn(),
  offlineQueue: [],
  prefetchForOffline: jest.fn(),
};

const mockOptimisticUpdates = {
  performOptimisticUpdate: jest.fn(),
};

// Mock the hook returns
(useAppStore as unknown as jest.Mock).mockReturnValue(mockAppStore);
(useUIStore as unknown as jest.Mock).mockReturnValue(mockUIStore);
(useErrorStore as unknown as jest.Mock).mockReturnValue(mockErrorStore);

// Mock the manager hooks
jest.mock('@/stores/app', () => ({
  useAppStore: Object.assign(jest.fn(), {
    getState: () => mockAppStore,
  }),
  useLoadingManager: () => ({
    setLoading: mockAppStore.setLoading,
    clearLoading: mockAppStore.clearLoading,
    isLoading: mockAppStore.isLoading,
    withLoading: jest.fn(async (key, operation) => {
      mockAppStore.setLoading(key, true);
      try {
        const result = await operation();
        return result;
      } finally {
        mockAppStore.setLoading(key, false);
      }
    }),
  }),
  useErrorManager: () => ({
    setError: mockAppStore.setError,
    clearError: mockAppStore.clearError,
    clearAllErrors: mockAppStore.clearAllErrors,
    hasError: mockAppStore.hasError,
    withErrorHandling: jest.fn(async (key, operation, errorHandler) => {
      mockAppStore.clearError(key);
      try {
        const result = await operation();
        return result;
      } catch (error) {
        const errorMessage = errorHandler 
          ? errorHandler(error)
          : error instanceof Error 
            ? error.message 
            : 'An unknown error occurred';
        mockAppStore.setError(key, errorMessage);
        return null;
      }
    }),
  }),
  useSyncManager: () => ({
    setSyncStatus: mockAppStore.setSyncStatus,
    incrementPendingOperations: mockAppStore.incrementPendingOperations,
    decrementPendingOperations: mockAppStore.decrementPendingOperations,
    incrementFailedOperations: jest.fn(),
    resetFailedOperations: mockAppStore.resetFailedOperations,
    markSyncComplete: jest.fn(),
    setOnlineStatus: jest.fn(),
  }),
}));

// Mock the offline support hooks
jest.mock('../useOfflineSupport', () => ({
  useOfflineSupport: () => mockOfflineSupport,
  useOptimisticUpdates: () => mockOptimisticUpdates,
}));

// Test wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('useStateManagement', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should provide all state management utilities', () => {
    const { result } = renderHook(() => useStateManagement(), {
      wrapper: createWrapper(),
    });

    expect(typeof result.current.executeOperation).toBe('function');
    expect(typeof result.current.executeBatchOperations).toBe('function');
    expect(typeof result.current.loadingManager).toBe('object');
    expect(typeof result.current.errorManager).toBe('object');
    expect(typeof result.current.syncManager).toBe('object');
    expect(typeof result.current.cacheManager).toBe('object');
    expect(typeof result.current.syncUtilities).toBe('object');
  });

  describe('executeOperation', () => {
    it('should execute operation successfully', async () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const mockOperation = jest.fn().mockResolvedValue('success');
      const mockOnSuccess = jest.fn();

      await act(async () => {
        const operationResult = await result.current.executeOperation({
          key: 'test-operation',
          operation: mockOperation,
          onSuccess: mockOnSuccess,
        });

        expect(operationResult).toBe('success');
      });

      expect(mockOperation).toHaveBeenCalled();
      expect(mockOnSuccess).toHaveBeenCalledWith('success');
      expect(mockUIStore.addNotification).toHaveBeenCalledWith({
        type: 'success',
        title: 'Operation Successful',
        message: 'test-operation completed successfully',
        read: false,
      });
    });

    it('should handle operation errors', async () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const mockError = new Error('Test error');
      const mockOperation = jest.fn().mockRejectedValue(mockError);
      const mockOnError = jest.fn();

      await act(async () => {
        await result.current.executeOperation({
          key: 'test-operation',
          operation: mockOperation,
          onError: mockOnError,
        });
      });

      expect(mockOnError).toHaveBeenCalledWith(mockError);
      expect(mockErrorStore.addError).toHaveBeenCalledWith({
        type: 'api',
        message: 'Test error',
        context: 'test-operation',
        retryable: true,
      });
    });

    it('should queue operation when offline', async () => {
      mockOfflineSupport.isOnline = false;
      
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const mockOperation = jest.fn().mockResolvedValue('success');

      await act(async () => {
        await result.current.executeOperation({
          key: 'test-operation',
          operation: mockOperation,
        });
      });

      expect(mockOfflineSupport.queueOfflineOperation).toHaveBeenCalledWith(
        mockOperation,
        'test-operation'
      );
      expect(mockUIStore.addNotification).toHaveBeenCalledWith({
        type: 'info',
        title: 'Operation Queued',
        message: 'test-operation will be processed when connection is restored',
        read: false,
      });
    });

    it('should handle optimistic updates', async () => {
      mockOfflineSupport.isOnline = true;
      mockOptimisticUpdates.performOptimisticUpdate.mockResolvedValue('optimistic-result');
      
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const mockOperation = jest.fn().mockResolvedValue('success');
      const mockUpdateFn = jest.fn();

      await act(async () => {
        await result.current.executeOperation({
          key: 'test-operation',
          operation: mockOperation,
          optimisticUpdate: {
            queryKey: ['test'],
            updateFn: mockUpdateFn,
          },
        });
      });

      expect(mockOptimisticUpdates.performOptimisticUpdate).toHaveBeenCalledWith({
        queryKey: ['test'],
        updateFn: mockUpdateFn,
        mutationFn: mockOperation,
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
        description: 'test-operation',
      });
    });
  });

  describe('executeBatchOperations', () => {
    it('should execute batch operations successfully', async () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const operations = [
        {
          key: 'op1',
          operation: jest.fn().mockResolvedValue('result1'),
          onSuccess: jest.fn(),
        },
        {
          key: 'op2',
          operation: jest.fn().mockResolvedValue('result2'),
          onSuccess: jest.fn(),
        },
      ];

      await act(async () => {
        const results = await result.current.executeBatchOperations({
          operations,
          batchKey: 'test-batch',
        });

        expect(results).toHaveLength(2);
      });

      expect(operations[0].operation).toHaveBeenCalled();
      expect(operations[1].operation).toHaveBeenCalled();
      expect(mockUIStore.addNotification).toHaveBeenCalledWith({
        type: 'info',
        title: 'Batch Operation Started',
        message: 'Processing 2 operations...',
        read: false,
      });
    });
  });

  describe('cacheManager', () => {
    it('should provide cache management utilities', () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.cacheManager.prefetchForOffline).toBe('function');
      expect(typeof result.current.cacheManager.invalidateQueries).toBe('function');
      expect(typeof result.current.cacheManager.clearCache).toBe('function');
      expect(typeof result.current.cacheManager.getCacheStats).toBe('function');
    });

    it('should get cache statistics', () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const stats = result.current.cacheManager.getCacheStats();

      expect(stats).toHaveProperty('queryCount');
      expect(stats).toHaveProperty('mutationCount');
      expect(stats).toHaveProperty('totalSize');
      expect(stats).toHaveProperty('maxSize');
    });
  });

  describe('syncUtilities', () => {
    it('should provide sync management utilities', () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.syncUtilities.forceSyncAll).toBe('function');
      expect(typeof result.current.syncUtilities.getSyncStatus).toBe('function');
      expect(typeof result.current.syncUtilities.clearFailedOperations).toBe('function');
    });

    it('should get sync status', () => {
      const { result } = renderHook(() => useStateManagement(), {
        wrapper: createWrapper(),
      });

      const status = result.current.syncUtilities.getSyncStatus();

      expect(status).toHaveProperty('isOnline');
      expect(status).toHaveProperty('pendingOperations');
      expect(status).toHaveProperty('failedOperations');
      expect(status).toHaveProperty('queuedOperations');
    });
  });
});

describe('useApiOperation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should create API operation function', () => {
    const { result } = renderHook(() => useApiOperation('test-api'), {
      wrapper: createWrapper(),
    });

    expect(typeof result.current).toBe('function');
  });

  it('should execute API operation', async () => {
    const { result } = renderHook(() => useApiOperation('test-api'), {
      wrapper: createWrapper(),
    });

    const mockOperation = jest.fn().mockResolvedValue('api-result');

    await act(async () => {
      await result.current(mockOperation);
    });

    expect(mockOperation).toHaveBeenCalled();
  });
});

describe('useBatchOperation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should create batch operation function', () => {
    const { result } = renderHook(() => useBatchOperation('test-batch'), {
      wrapper: createWrapper(),
    });

    expect(typeof result.current).toBe('function');
  });

  it('should execute batch operations', async () => {
    const { result } = renderHook(() => useBatchOperation('test-batch'), {
      wrapper: createWrapper(),
    });

    const operations = [
      {
        key: 'op1',
        operation: jest.fn().mockResolvedValue('result1'),
      },
    ];

    await act(async () => {
      await result.current(operations);
    });

    expect(operations[0].operation).toHaveBeenCalled();
  });
});