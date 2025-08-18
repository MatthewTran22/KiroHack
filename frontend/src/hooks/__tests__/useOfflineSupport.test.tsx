import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { useOfflineSupport } from '../useOfflineSupport';
import { useErrorStore } from '@/stores/error';
import { useUIStore } from '@/stores/ui';

// Mock the stores
jest.mock('@/stores/error');
jest.mock('@/stores/ui');

const mockErrorStore = {
  setOfflineStatus: jest.fn(),
  addToRetryQueue: jest.fn(),
  retryAllOperations: jest.fn(),
};

const mockUIStore = {
  addNotification: jest.fn(),
};

(useErrorStore as unknown as jest.Mock).mockReturnValue(mockErrorStore);
(useUIStore as unknown as jest.Mock).mockReturnValue(mockUIStore);

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

describe('useOfflineSupport', () => {
  let originalOnLine: boolean;

  beforeEach(() => {
    originalOnLine = navigator.onLine;
    jest.clearAllMocks();
  });

  afterEach(() => {
    Object.defineProperty(navigator, 'onLine', {
      value: originalOnLine,
      writable: true,
    });
  });

  it('should detect initial online status', () => {
    Object.defineProperty(navigator, 'onLine', { value: true, writable: true });
    
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isOnline).toBe(true);
  });

  it('should detect initial offline status', () => {
    Object.defineProperty(navigator, 'onLine', { value: false, writable: true });
    
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isOnline).toBe(false);
  });

  it('should provide offline queue management functions', () => {
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    expect(typeof result.current.queueOfflineOperation).toBe('function');
    expect(typeof result.current.removeFromOfflineQueue).toBe('function');
    expect(typeof result.current.clearOfflineQueue).toBe('function');
    expect(typeof result.current.processOfflineQueue).toBe('function');
  });

  it('should provide cache management functions', () => {
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    expect(typeof result.current.getCachedData).toBe('function');
    expect(typeof result.current.setCachedData).toBe('function');
    expect(typeof result.current.hasCachedData).toBe('function');
    expect(typeof result.current.prefetchForOffline).toBe('function');
  });

  it('should manage offline queue state', () => {
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    expect(Array.isArray(result.current.offlineQueue)).toBe(true);
    expect(result.current.offlineQueue).toHaveLength(0);
  });

  it('should queue offline operation', () => {
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    const operation = jest.fn().mockResolvedValue('success');
    const description = 'Test operation';

    act(() => {
      const operationId = result.current.queueOfflineOperation(operation, description);
      expect(typeof operationId).toBe('string');
    });

    expect(result.current.offlineQueue).toHaveLength(1);
    expect(result.current.offlineQueue[0]).toMatchObject({
      operation,
      description,
      retries: 0,
      maxRetries: 3,
    });
  });

  it('should manage cached data', () => {
    const { result } = renderHook(() => useOfflineSupport(), {
      wrapper: createWrapper(),
    });

    const queryKey = ['test', 'data'];
    const testData = { id: 1, name: 'Test' };

    expect(result.current.hasCachedData(queryKey)).toBe(false);

    act(() => {
      result.current.setCachedData(queryKey, testData);
    });

    expect(result.current.hasCachedData(queryKey)).toBe(true);
    expect(result.current.getCachedData(queryKey)).toEqual(testData);
  });
});