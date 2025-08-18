import { act, renderHook } from '@testing-library/react';
import { useErrorStore, useErrorBoundary, useNetworkStatus } from '../error';
import { APIError } from '@/lib/api';

// Mock the auth module
jest.mock('@/lib/auth', () => ({
  tokenManager: {
    clearTokens: jest.fn(),
  },
}));

describe('Error Store', () => {
  beforeEach(() => {
    // Reset store state before each test
    useErrorStore.setState({
      errors: [],
      globalError: null,
      isOffline: false,
      retryQueue: [],
    });
    
    // Clear any existing timers
    jest.clearAllTimers();
  });

  afterEach(() => {
    jest.clearAllTimers();
  });

  describe('Error Management', () => {
    it('should add error', () => {
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        result.current.clearErrors(); // Clear any existing errors
        result.current.addError({
          type: 'api',
          message: 'Test error',
          context: 'test',
          retryable: false,
        });
      });

      expect(result.current.errors).toHaveLength(1);
      expect(result.current.errors[0]).toMatchObject({
        type: 'api',
        message: 'Test error',
        context: 'test',
        retryable: false,
        dismissed: false,
      });
      expect(result.current.errors[0].id).toBeDefined();
      expect(result.current.errors[0].timestamp).toBeInstanceOf(Date);
    });

    it('should limit errors to 50', () => {
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        // Add 60 errors
        for (let i = 0; i < 60; i++) {
          result.current.addError({
            type: 'api',
            message: `Error ${i}`,
            context: 'test',
            retryable: false,
          });
        }
      });

      expect(result.current.errors).toHaveLength(50);
      expect(result.current.errors[0].message).toBe('Error 59'); // Latest first
    });

    it('should auto-dismiss non-critical errors', () => {
      jest.useFakeTimers();
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        result.current.addError({
          type: 'network',
          message: 'Network error',
          context: 'test',
          retryable: true,
        });
      });

      expect(result.current.errors[0].dismissed).toBe(false);

      // Fast-forward 10 seconds
      act(() => {
        jest.advanceTimersByTime(10000);
      });

      expect(result.current.errors[0].dismissed).toBe(true);
    });

    it('should not auto-dismiss auth errors', () => {
      jest.useFakeTimers();
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        result.current.addError({
          type: 'auth',
          message: 'Auth error',
          context: 'test',
          retryable: false,
        });
      });

      // Fast-forward 10 seconds
      act(() => {
        jest.advanceTimersByTime(10000);
      });

      expect(result.current.errors[0].dismissed).toBe(false);
    });

    it('should remove error', () => {
      const { result } = renderHook(() => useErrorStore());
      let errorId: string;

      act(() => {
        errorId = result.current.addError({
          type: 'api',
          message: 'Test error',
          context: 'test',
          retryable: false,
        });
      });

      expect(result.current.errors).toHaveLength(1);

      act(() => {
        result.current.removeError(errorId);
      });

      expect(result.current.errors).toHaveLength(0);
    });

    it('should dismiss error', () => {
      const { result } = renderHook(() => useErrorStore());
      let errorId: string;

      act(() => {
        errorId = result.current.addError({
          type: 'api',
          message: 'Test error',
          context: 'test',
          retryable: false,
        });
      });

      expect(result.current.errors[0].dismissed).toBe(false);

      act(() => {
        result.current.dismissError(errorId);
      });

      expect(result.current.errors[0].dismissed).toBe(true);
    });

    it('should clear all errors', () => {
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        result.current.addError({
          type: 'api',
          message: 'Error 1',
          context: 'test',
          retryable: false,
        });
        result.current.addError({
          type: 'network',
          message: 'Error 2',
          context: 'test',
          retryable: true,
        });
      });

      expect(result.current.errors).toHaveLength(2);

      act(() => {
        result.current.clearErrors();
      });

      expect(result.current.errors).toHaveLength(0);
    });

    it('should set global error', () => {
      const { result } = renderHook(() => useErrorStore());
      const globalError = {
        id: 'global-1',
        type: 'unknown' as const,
        message: 'Critical error',
        timestamp: new Date(),
        context: 'critical',
        retryable: false,
        dismissed: false,
      };

      act(() => {
        result.current.setGlobalError(globalError);
      });

      expect(result.current.globalError).toEqual(globalError);
    });
  });

  describe('Network Status', () => {
    it('should set offline status', () => {
      const { result } = renderHook(() => useErrorStore());

      act(() => {
        result.current.setOfflineStatus(true);
      });

      expect(result.current.isOffline).toBe(true);
    });

    it('should retry operations when back online', () => {
      const { result } = renderHook(() => useErrorStore());
      const retryAllOperationsSpy = jest.spyOn(result.current, 'retryAllOperations');

      act(() => {
        result.current.setOfflineStatus(true);
      });

      act(() => {
        result.current.setOfflineStatus(false);
      });

      expect(retryAllOperationsSpy).toHaveBeenCalled();
    });
  });

  describe('Retry Queue', () => {
    it('should add operation to retry queue', () => {
      const { result } = renderHook(() => useErrorStore());
      const operation = jest.fn().mockResolvedValue('success');

      act(() => {
        result.current.addToRetryQueue(operation, 3);
      });

      expect(result.current.retryQueue).toHaveLength(1);
      expect(result.current.retryQueue[0]).toMatchObject({
        operation,
        maxRetries: 3,
        currentRetries: 0,
      });
    });

    it('should retry operation successfully', async () => {
      const { result } = renderHook(() => useErrorStore());
      const operation = jest.fn().mockResolvedValue('success');
      let operationId: string;

      act(() => {
        operationId = result.current.addToRetryQueue(operation);
      });

      await act(async () => {
        await result.current.retryOperation(operationId);
      });

      expect(operation).toHaveBeenCalled();
      expect(result.current.retryQueue).toHaveLength(0); // Should be removed after success
    });

    it('should increment retry count on failure', async () => {
      const { result } = renderHook(() => useErrorStore());
      const operation = jest.fn().mockRejectedValue(new Error('Failed'));
      let operationId: string;

      act(() => {
        operationId = result.current.addToRetryQueue(operation, 3);
      });

      await act(async () => {
        await result.current.retryOperation(operationId);
      });

      expect(result.current.retryQueue[0].currentRetries).toBe(1);
    });

    it('should remove operation after max retries', async () => {
      const { result } = renderHook(() => useErrorStore());
      const operation = jest.fn().mockRejectedValue(new Error('Failed'));
      let operationId: string;

      act(() => {
        operationId = result.current.addToRetryQueue(operation, 2); // Max 2 retries
      });

      // First retry - should increment retries but keep in queue
      await act(async () => {
        await result.current.retryOperation(operationId);
      });

      expect(result.current.retryQueue).toHaveLength(1);
      expect(result.current.retryQueue[0].currentRetries).toBe(1);

      // Second retry - should increment retries but still keep in queue
      await act(async () => {
        await result.current.retryOperation(operationId);
      });

      expect(result.current.retryQueue).toHaveLength(1);
      expect(result.current.retryQueue[0].currentRetries).toBe(2);

      // Third retry - should exceed max retries and remove from queue
      await act(async () => {
        await result.current.retryOperation(operationId);
      });

      expect(result.current.retryQueue).toHaveLength(0);
    });

    it('should remove operation from retry queue', () => {
      const { result } = renderHook(() => useErrorStore());
      const operation = jest.fn();
      let operationId: string;

      act(() => {
        operationId = result.current.addToRetryQueue(operation);
      });

      expect(result.current.retryQueue).toHaveLength(1);

      act(() => {
        result.current.removeFromRetryQueue(operationId);
      });

      expect(result.current.retryQueue).toHaveLength(0);
    });

    it('should retry all operations', async () => {
      const { result } = renderHook(() => useErrorStore());
      const operation1 = jest.fn().mockResolvedValue('success1');
      const operation2 = jest.fn().mockResolvedValue('success2');

      act(() => {
        result.current.addToRetryQueue(operation1);
        result.current.addToRetryQueue(operation2);
      });

      await act(async () => {
        await result.current.retryAllOperations();
      });

      expect(operation1).toHaveBeenCalled();
      expect(operation2).toHaveBeenCalled();
      expect(result.current.retryQueue).toHaveLength(0);
    });
  });

  describe('Error Handling Utilities', () => {
    it('should handle API error', () => {
      const { result } = renderHook(() => useErrorStore());
      const apiError = new APIError('API failed', 500, 'SERVER_ERROR', { detail: 'test' });

      act(() => {
        result.current.handleAPIError(apiError, 'test-context');
      });

      expect(result.current.errors).toHaveLength(1);
      expect(result.current.errors[0]).toMatchObject({
        type: 'api',
        message: 'API failed',
        context: 'test-context',
        retryable: true, // 500 errors are retryable
        details: {
          status: 500,
          code: 'SERVER_ERROR',
          detail: 'test',
        },
      });
    });

    it('should handle 401 API error as auth error', () => {
      const { result } = renderHook(() => useErrorStore());
      const apiError = new APIError('Unauthorized', 401);

      act(() => {
        result.current.handleAPIError(apiError);
      });

      expect(result.current.errors[0].type).toBe('auth');
      expect(result.current.errors[0].retryable).toBe(false);
    });

    it('should handle network error', () => {
      const { result } = renderHook(() => useErrorStore());
      const networkError = new Error('Network failed');

      act(() => {
        result.current.handleNetworkError(networkError, 'network-context');
      });

      expect(result.current.errors).toHaveLength(1);
      expect(result.current.errors[0]).toMatchObject({
        type: 'network',
        message: 'Network connection failed. Please check your internet connection.',
        context: 'network-context',
        retryable: true,
        details: {
          originalError: 'Network failed',
        },
      });
    });

    it('should handle validation error', () => {
      const { result } = renderHook(() => useErrorStore());
      const validationErrors = {
        email: 'Invalid email',
        password: 'Password too short',
      };

      act(() => {
        result.current.handleValidationError(validationErrors, 'form-context');
      });

      expect(result.current.errors).toHaveLength(1);
      expect(result.current.errors[0]).toMatchObject({
        type: 'validation',
        message: 'Validation failed: Invalid email, Password too short',
        context: 'form-context',
        retryable: false,
        details: {
          validationErrors,
        },
      });
    });
  });
});

describe('useErrorBoundary', () => {
  it('should capture error', () => {
    const { result: errorStore } = renderHook(() => useErrorStore());
    const { result: errorBoundary } = renderHook(() => useErrorBoundary());

    const error = new Error('Component error');
    const errorInfo = { componentStack: 'Component stack trace' };

    act(() => {
      errorStore.current.clearErrors(); // Clear any existing errors
      errorBoundary.current.captureError(error, errorInfo);
    });

    expect(errorStore.current.errors).toHaveLength(1);
    expect(errorStore.current.errors[0]).toMatchObject({
      type: 'unknown',
      message: 'Component error',
      context: 'error-boundary',
      retryable: false,
      details: {
        stack: error.stack,
        componentStack: 'Component stack trace',
      },
    });
  });

  it('should set global error for chunk load errors', () => {
    const { result: errorStore } = renderHook(() => useErrorStore());
    const { result: errorBoundary } = renderHook(() => useErrorBoundary());

    const error = new Error('ChunkLoadError: Loading chunk failed');

    act(() => {
      errorBoundary.current.captureError(error);
    });

    expect(errorStore.current.globalError).toMatchObject({
      type: 'unknown',
      message: 'Application update detected. Please refresh the page.',
      context: 'chunk-load-error',
      retryable: true,
    });
  });
});

describe('useNetworkStatus', () => {
  let originalOnLine: boolean;
  let eventListeners: { [key: string]: EventListener[] } = {};

  beforeEach(() => {
    originalOnLine = navigator.onLine;
    eventListeners = {};

    // Mock addEventListener
    window.addEventListener = jest.fn((event: string, listener: EventListener) => {
      if (!eventListeners[event]) {
        eventListeners[event] = [];
      }
      eventListeners[event].push(listener);
    });

    window.removeEventListener = jest.fn();
  });

  afterEach(() => {
    Object.defineProperty(navigator, 'onLine', {
      value: originalOnLine,
      writable: true,
    });
  });

  it('should set up network event listeners', () => {
    renderHook(() => useNetworkStatus());

    expect(window.addEventListener).toHaveBeenCalledWith('online', expect.any(Function));
    expect(window.addEventListener).toHaveBeenCalledWith('offline', expect.any(Function));
  });

  it('should set initial offline status', () => {
    Object.defineProperty(navigator, 'onLine', { value: false, writable: true });
    
    const { result: errorStore } = renderHook(() => useErrorStore());
    renderHook(() => useNetworkStatus());

    expect(errorStore.current.isOffline).toBe(true);
  });

  it('should handle online event', () => {
    const { result: errorStore } = renderHook(() => useErrorStore());
    renderHook(() => useNetworkStatus());

    // Simulate offline first
    act(() => {
      errorStore.current.setOfflineStatus(true);
    });

    // Simulate online event
    act(() => {
      const onlineListeners = eventListeners['online'] || [];
      onlineListeners.forEach(listener => listener(new Event('online')));
    });

    expect(errorStore.current.isOffline).toBe(false);
  });

  it('should handle offline event', () => {
    const { result: errorStore } = renderHook(() => useErrorStore());
    renderHook(() => useNetworkStatus());

    // Simulate offline event
    act(() => {
      const offlineListeners = eventListeners['offline'] || [];
      offlineListeners.forEach(listener => listener(new Event('offline')));
    });

    expect(errorStore.current.isOffline).toBe(true);
  });
});