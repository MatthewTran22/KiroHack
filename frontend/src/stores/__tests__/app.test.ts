import { renderHook, act } from '@testing-library/react';
import { 
  useAppStore, 
  useLoadingManager, 
  useErrorManager, 
  useSyncManager,
  useLoadingState,
  useErrorState,
  useSyncStatus,
  useCacheInfo
} from '../app';

describe('useAppStore', () => {
  beforeEach(() => {
    // Reset store before each test
    useAppStore.getState().reset();
  });

  describe('Loading State Management', () => {
    it('should manage loading states', () => {
      const { result } = renderHook(() => useAppStore());

      expect(result.current.isLoading('test')).toBe(false);

      act(() => {
        result.current.setLoading('test', true);
      });

      expect(result.current.isLoading('test')).toBe(true);
      expect(result.current.loading.test).toBe(true);

      act(() => {
        result.current.setLoading('test', false);
      });

      expect(result.current.isLoading('test')).toBe(false);
    });

    it('should clear all loading states', () => {
      const { result } = renderHook(() => useAppStore());

      act(() => {
        result.current.setLoading('test1', true);
        result.current.setLoading('test2', true);
      });

      expect(result.current.isLoading('test1')).toBe(true);
      expect(result.current.isLoading('test2')).toBe(true);

      act(() => {
        result.current.clearLoading();
      });

      expect(result.current.isLoading('test1')).toBe(false);
      expect(result.current.isLoading('test2')).toBe(false);
      expect(Object.keys(result.current.loading)).toHaveLength(0);
    });
  });

  describe('Error State Management', () => {
    it('should manage error states', () => {
      const { result } = renderHook(() => useAppStore());

      expect(result.current.hasError('test')).toBe(false);

      act(() => {
        result.current.setError('test', 'Test error');
      });

      expect(result.current.hasError('test')).toBe(true);
      expect(result.current.errors.test).toBe('Test error');

      act(() => {
        result.current.clearError('test');
      });

      expect(result.current.hasError('test')).toBe(false);
      expect(result.current.errors.test).toBeUndefined();
    });

    it('should clear all error states', () => {
      const { result } = renderHook(() => useAppStore());

      act(() => {
        result.current.setError('test1', 'Error 1');
        result.current.setError('test2', 'Error 2');
      });

      expect(result.current.hasError('test1')).toBe(true);
      expect(result.current.hasError('test2')).toBe(true);

      act(() => {
        result.current.clearAllErrors();
      });

      expect(result.current.hasError('test1')).toBe(false);
      expect(result.current.hasError('test2')).toBe(false);
      expect(Object.keys(result.current.errors)).toHaveLength(0);
    });

    it('should handle null errors', () => {
      const { result } = renderHook(() => useAppStore());

      act(() => {
        result.current.setError('test', 'Error');
      });

      expect(result.current.hasError('test')).toBe(true);

      act(() => {
        result.current.setError('test', null);
      });

      expect(result.current.hasError('test')).toBe(false);
    });
  });

  describe('Sync Status Management', () => {
    it('should manage sync status', () => {
      const { result } = renderHook(() => useAppStore());

      const newStatus = {
        isOnline: false,
        lastSync: new Date(),
        pendingOperations: 5,
        failedOperations: 2,
      };

      act(() => {
        result.current.setSyncStatus(newStatus);
      });

      expect(result.current.sync).toMatchObject(newStatus);
    });

    it('should manage pending operations', () => {
      const { result } = renderHook(() => useAppStore());

      expect(result.current.sync.pendingOperations).toBe(0);

      act(() => {
        result.current.incrementPendingOperations();
        result.current.incrementPendingOperations();
      });

      expect(result.current.sync.pendingOperations).toBe(2);

      act(() => {
        result.current.decrementPendingOperations();
      });

      expect(result.current.sync.pendingOperations).toBe(1);

      // Should not go below 0
      act(() => {
        result.current.decrementPendingOperations();
        result.current.decrementPendingOperations();
      });

      expect(result.current.sync.pendingOperations).toBe(0);
    });

    it('should manage failed operations', () => {
      const { result } = renderHook(() => useAppStore());

      expect(result.current.sync.failedOperations).toBe(0);

      act(() => {
        result.current.incrementFailedOperations();
        result.current.incrementFailedOperations();
      });

      expect(result.current.sync.failedOperations).toBe(2);

      act(() => {
        result.current.resetFailedOperations();
      });

      expect(result.current.sync.failedOperations).toBe(0);
    });
  });

  describe('Cache Management', () => {
    it('should manage cache size', () => {
      const { result } = renderHook(() => useAppStore());

      act(() => {
        result.current.updateCacheSize(1024);
      });

      expect(result.current.cache.size).toBe(1024);
    });

    it('should manage cache metadata', () => {
      const { result } = renderHook(() => useAppStore());

      const metadata = {
        timestamp: Date.now(),
        ttl: 60000,
        version: '1.0.0',
      };

      act(() => {
        result.current.setCacheMetadata('test-key', metadata);
      });

      expect(result.current.getCacheMetadata('test-key')).toEqual(metadata);
      expect(result.current.getCacheMetadata('non-existent')).toBeNull();
    });

    it('should clear expired cache metadata', () => {
      const { result } = renderHook(() => useAppStore());

      const now = Date.now();
      const validMetadata = {
        timestamp: now,
        ttl: 60000, // 1 minute
        version: '1.0.0',
      };
      const expiredMetadata = {
        timestamp: now - 120000, // 2 minutes ago
        ttl: 60000, // 1 minute TTL
        version: '1.0.0',
      };

      act(() => {
        result.current.setCacheMetadata('valid', validMetadata);
        result.current.setCacheMetadata('expired', expiredMetadata);
      });

      expect(result.current.getCacheMetadata('valid')).toEqual(validMetadata);
      expect(result.current.getCacheMetadata('expired')).toEqual(expiredMetadata);

      act(() => {
        result.current.clearExpiredCache();
      });

      expect(result.current.getCacheMetadata('valid')).toEqual(validMetadata);
      expect(result.current.getCacheMetadata('expired')).toBeNull();
    });
  });

  describe('Global Reset', () => {
    it('should reset all state', () => {
      const { result } = renderHook(() => useAppStore());

      // Set some state
      act(() => {
        result.current.setLoading('test', true);
        result.current.setError('test', 'Error');
        result.current.setSyncStatus({ pendingOperations: 5 });
        result.current.updateCacheSize(1024);
      });

      // Verify state is set
      expect(result.current.isLoading('test')).toBe(true);
      expect(result.current.hasError('test')).toBe(true);
      expect(result.current.sync.pendingOperations).toBe(5);
      expect(result.current.cache.size).toBe(1024);

      // Reset
      act(() => {
        result.current.reset();
      });

      // Verify state is reset
      expect(result.current.isLoading('test')).toBe(false);
      expect(result.current.hasError('test')).toBe(false);
      expect(result.current.sync.pendingOperations).toBe(0);
      expect(result.current.cache.size).toBe(0);
    });
  });
});

describe('Selector Hooks', () => {
  beforeEach(() => {
    useAppStore.getState().reset();
  });

  it('should provide loading state selector', () => {
    const { result } = renderHook(() => useLoadingState());

    expect(result.current).toEqual({});

    act(() => {
      useAppStore.getState().setLoading('test', true);
    });

    expect(result.current.test).toBe(true);
  });

  it('should provide error state selector', () => {
    const { result } = renderHook(() => useErrorState());

    expect(result.current).toEqual({});

    act(() => {
      useAppStore.getState().setError('test', 'Error');
    });

    expect(result.current.test).toBe('Error');
  });

  it('should provide sync status selector', () => {
    const { result } = renderHook(() => useSyncStatus());

    expect(result.current.pendingOperations).toBe(0);

    act(() => {
      useAppStore.getState().incrementPendingOperations();
    });

    expect(result.current.pendingOperations).toBe(1);
  });

  it('should provide cache info selector', () => {
    const { result } = renderHook(() => useCacheInfo());

    expect(result.current.size).toBe(0);

    act(() => {
      useAppStore.getState().updateCacheSize(1024);
    });

    expect(result.current.size).toBe(1024);
  });
});

describe('Manager Hooks', () => {
  beforeEach(() => {
    useAppStore.getState().reset();
  });

  describe('useLoadingManager', () => {
    it('should provide loading management functions', () => {
      const { result } = renderHook(() => useLoadingManager());

      expect(typeof result.current.setLoading).toBe('function');
      expect(typeof result.current.clearLoading).toBe('function');
      expect(typeof result.current.isLoading).toBe('function');
      expect(typeof result.current.withLoading).toBe('function');
    });

    it('should handle withLoading wrapper', async () => {
      const { result } = renderHook(() => useLoadingManager());

      const mockOperation = jest.fn().mockResolvedValue('success');

      expect(result.current.isLoading('test')).toBe(false);

      const promise = act(async () => {
        return result.current.withLoading('test', mockOperation);
      });

      // Should be loading during operation
      expect(result.current.isLoading('test')).toBe(true);

      const resultValue = await promise;

      // Should not be loading after operation
      expect(result.current.isLoading('test')).toBe(false);
      expect(resultValue).toBe('success');
      expect(mockOperation).toHaveBeenCalled();
    });

    it('should handle withLoading wrapper errors', async () => {
      const { result } = renderHook(() => useLoadingManager());

      const mockOperation = jest.fn().mockRejectedValue(new Error('Test error'));

      await expect(
        act(async () => {
          return result.current.withLoading('test', mockOperation);
        })
      ).rejects.toThrow('Test error');

      // Should not be loading after error
      expect(result.current.isLoading('test')).toBe(false);
    });
  });

  describe('useErrorManager', () => {
    it('should provide error management functions', () => {
      const { result } = renderHook(() => useErrorManager());

      expect(typeof result.current.setError).toBe('function');
      expect(typeof result.current.clearError).toBe('function');
      expect(typeof result.current.clearAllErrors).toBe('function');
      expect(typeof result.current.hasError).toBe('function');
      expect(typeof result.current.withErrorHandling).toBe('function');
    });

    it('should handle withErrorHandling wrapper success', async () => {
      const { result } = renderHook(() => useErrorManager());

      const mockOperation = jest.fn().mockResolvedValue('success');

      const resultValue = await act(async () => {
        return result.current.withErrorHandling('test', mockOperation);
      });

      expect(resultValue).toBe('success');
      expect(result.current.hasError('test')).toBe(false);
    });

    it('should handle withErrorHandling wrapper errors', async () => {
      const { result } = renderHook(() => useErrorManager());

      const mockOperation = jest.fn().mockRejectedValue(new Error('Test error'));

      const resultValue = await act(async () => {
        return result.current.withErrorHandling('test', mockOperation);
      });

      expect(resultValue).toBeNull();
      expect(result.current.hasError('test')).toBe(true);
    });

    it('should handle custom error handler', async () => {
      const { result } = renderHook(() => useErrorManager());

      const mockOperation = jest.fn().mockRejectedValue(new Error('Test error'));
      const customErrorHandler = jest.fn().mockReturnValue('Custom error message');

      await act(async () => {
        return result.current.withErrorHandling('test', mockOperation, customErrorHandler);
      });

      expect(customErrorHandler).toHaveBeenCalledWith(expect.any(Error));
      expect(useAppStore.getState().errors.test).toBe('Custom error message');
    });
  });

  describe('useSyncManager', () => {
    it('should provide sync management functions', () => {
      const { result } = renderHook(() => useSyncManager());

      expect(typeof result.current.setSyncStatus).toBe('function');
      expect(typeof result.current.incrementPendingOperations).toBe('function');
      expect(typeof result.current.decrementPendingOperations).toBe('function');
      expect(typeof result.current.incrementFailedOperations).toBe('function');
      expect(typeof result.current.resetFailedOperations).toBe('function');
      expect(typeof result.current.markSyncComplete).toBe('function');
      expect(typeof result.current.setOnlineStatus).toBe('function');
    });

    it('should mark sync complete', () => {
      const { result } = renderHook(() => useSyncManager());

      act(() => {
        useAppStore.getState().incrementFailedOperations();
      });

      expect(useAppStore.getState().sync.failedOperations).toBe(1);
      expect(useAppStore.getState().sync.lastSync).toBeNull();

      act(() => {
        result.current.markSyncComplete();
      });

      expect(useAppStore.getState().sync.failedOperations).toBe(0);
      expect(useAppStore.getState().sync.lastSync).toBeInstanceOf(Date);
    });

    it('should set online status', () => {
      const { result } = renderHook(() => useSyncManager());

      act(() => {
        result.current.setOnlineStatus(false);
      });

      expect(useAppStore.getState().sync.isOnline).toBe(false);

      act(() => {
        result.current.setOnlineStatus(true);
      });

      expect(useAppStore.getState().sync.isOnline).toBe(true);
    });
  });
});