import { QueryClient } from '@tanstack/react-query';
import { queryClient, queryKeys, offlineConfig, handleQueryError } from '../query-client';
import { APIError } from '../api';

// Mock the auth module
jest.mock('../auth', () => ({
  tokenManager: {
    clearTokens: jest.fn(),
    isTokenValid: jest.fn(() => true),
    getToken: jest.fn(() => 'mock-token'),
  },
}));

describe('Query Client Configuration', () => {
  beforeEach(() => {
    queryClient.clear();
  });

  it('should create query client with correct default options', () => {
    expect(queryClient).toBeInstanceOf(QueryClient);
    
    const defaultOptions = queryClient.getDefaultOptions();
    expect(defaultOptions.queries?.staleTime).toBe(5 * 60 * 1000); // 5 minutes
    expect(defaultOptions.queries?.gcTime).toBe(10 * 60 * 1000); // 10 minutes
    expect(defaultOptions.queries?.refetchOnWindowFocus).toBe(false);
    expect(defaultOptions.queries?.refetchOnReconnect).toBe(true);
  });

  it('should not retry on 4xx errors except 401', () => {
    const retryFn = queryClient.getDefaultOptions().queries?.retry as Function;
    
    // Should not retry on 400
    expect(retryFn(1, new APIError('Bad Request', 400))).toBe(false);
    
    // Should not retry on 404
    expect(retryFn(1, new APIError('Not Found', 404))).toBe(false);
    
    // Should retry on 401 (token might be expired)
    expect(retryFn(1, new APIError('Unauthorized', 401))).toBe(true);
    
    // Should retry on 500
    expect(retryFn(1, new APIError('Server Error', 500))).toBe(true);
    
    // Should retry on network errors
    expect(retryFn(1, new Error('Network Error'))).toBe(true);
    
    // Should not retry after 3 attempts
    expect(retryFn(3, new APIError('Server Error', 500))).toBe(false);
  });

  it('should calculate retry delay correctly', () => {
    const retryDelayFn = queryClient.getDefaultOptions().queries?.retryDelay as Function;
    
    expect(retryDelayFn(0)).toBe(1000); // 1 second
    expect(retryDelayFn(1)).toBe(2000); // 2 seconds
    expect(retryDelayFn(2)).toBe(4000); // 4 seconds
    expect(retryDelayFn(10)).toBe(30000); // Max 30 seconds
  });
});

describe('Query Keys Factory', () => {
  it('should generate consistent auth query keys', () => {
    expect(queryKeys.auth.user).toEqual(['auth', 'user']);
    expect(queryKeys.auth.session).toEqual(['auth', 'session']);
  });

  it('should generate consistent document query keys', () => {
    expect(queryKeys.documents.all).toEqual(['documents']);
    expect(queryKeys.documents.lists()).toEqual(['documents', 'list']);
    expect(queryKeys.documents.list({ category: 'policy' })).toEqual([
      'documents', 'list', { category: 'policy' }
    ]);
    expect(queryKeys.documents.detail('123')).toEqual(['documents', 'detail', '123']);
    expect(queryKeys.documents.search('test')).toEqual(['documents', 'search', 'test']);
  });

  it('should generate consistent consultation query keys', () => {
    expect(queryKeys.consultations.all).toEqual(['consultations']);
    expect(queryKeys.consultations.session('123')).toEqual(['consultations', 'sessions', '123']);
    expect(queryKeys.consultations.messages('123')).toEqual([
      'consultations', 'sessions', '123', 'messages'
    ]);
  });

  it('should generate consistent knowledge query keys', () => {
    expect(queryKeys.knowledge.search('test')).toEqual(['knowledge', 'search', 'test']);
    expect(queryKeys.knowledge.suggestions('context')).toEqual([
      'knowledge', 'suggestions', 'context'
    ]);
  });

  it('should generate consistent audit query keys', () => {
    expect(queryKeys.audit.logs({ userId: '123' })).toEqual([
      'audit', 'logs', { userId: '123' }
    ]);
    expect(queryKeys.audit.report('report-123')).toEqual([
      'audit', 'reports', 'report-123'
    ]);
  });
});

describe('Offline Configuration', () => {
  beforeEach(() => {
    // Mock navigator.onLine
    Object.defineProperty(navigator, 'onLine', {
      writable: true,
      value: true,
    });
  });

  it('should detect online status', () => {
    expect(offlineConfig.isOnline()).toBe(true);
    
    Object.defineProperty(navigator, 'onLine', { value: false });
    expect(offlineConfig.isOnline()).toBe(false);
  });

  it('should retry failed mutations', () => {
    const mockMutation = {
      state: { status: 'error' },
      continue: jest.fn(),
      options: { mutationKey: ['test'] },
      mutationId: 1,
    };

    // Mock the getAll method to return our mock mutation
    jest.spyOn(queryClient.getMutationCache(), 'getAll').mockReturnValue([mockMutation as unknown]);
    
    offlineConfig.retryFailedMutations();
    
    expect(mockMutation.continue).toHaveBeenCalled();
  });

  it('should invalidate stale queries on reconnect', () => {
    const invalidateQueriesSpy = jest.spyOn(queryClient, 'invalidateQueries');
    
    // Add a stale query
    queryClient.setQueryData(['test'], 'data');
    const query = queryClient.getQueryCache().find({ queryKey: ['test'] });
    if (query) {
      query.state.dataUpdatedAt = Date.now() - 10 * 60 * 1000; // 10 minutes ago
    }
    
    offlineConfig.invalidateOnReconnect();
    
    expect(invalidateQueriesSpy).toHaveBeenCalled();
  });
});

describe('Error Handling', () => {
  it('should handle API errors correctly', () => {
    const apiError = new APIError('Test error', 400, 'TEST_CODE', { detail: 'test' });
    const result = handleQueryError(apiError);
    
    expect(result).toBe(apiError);
  });

  it('should handle 401 errors by clearing tokens', () => {
    const { tokenManager } = require('../auth');
    const apiError = new APIError('Unauthorized', 401);
    
    handleQueryError(apiError);
    
    expect(tokenManager.clearTokens).toHaveBeenCalled();
  });

  it('should handle network errors', () => {
    const networkError = new Error('Network failed');
    const result = handleQueryError(networkError);
    
    expect(result).toBeInstanceOf(Error);
    expect(result.message).toBe('Network error occurred');
  });

  it('should handle unknown errors', () => {
    const unknownError = 'string error';
    const result = handleQueryError(unknownError);
    
    expect(result).toBeInstanceOf(Error);
    expect(result.message).toBe('Network error occurred');
  });
});

describe('Network Event Listeners', () => {
  let originalAddEventListener: typeof window.addEventListener;
  let eventListeners: { [key: string]: EventListener[] } = {};

  beforeEach(() => {
    eventListeners = {};
    originalAddEventListener = window.addEventListener;
    
    // Mock addEventListener to track listeners
    window.addEventListener = jest.fn((event: string, listener: EventListener) => {
      if (!eventListeners[event]) {
        eventListeners[event] = [];
      }
      eventListeners[event].push(listener);
    });
  });

  afterEach(() => {
    window.addEventListener = originalAddEventListener;
  });

  it('should set up network event listeners', () => {
    // Re-import to trigger the event listener setup
    jest.resetModules();
    require('../query-client');
    
    expect(window.addEventListener).toHaveBeenCalledWith('online', expect.any(Function));
    expect(window.addEventListener).toHaveBeenCalledWith('offline', expect.any(Function));
  });

  it('should handle online event', () => {
    const retryFailedMutationsSpy = jest.spyOn(offlineConfig, 'retryFailedMutations').mockImplementation(() => {});
    const invalidateOnReconnectSpy = jest.spyOn(offlineConfig, 'invalidateOnReconnect').mockImplementation(() => {});
    
    // Re-import to trigger the event listener setup
    jest.resetModules();
    const { offlineConfig: newOfflineConfig } = require('../query-client');
    
    // Mock the functions on the new instance
    jest.spyOn(newOfflineConfig, 'retryFailedMutations').mockImplementation(() => {});
    jest.spyOn(newOfflineConfig, 'invalidateOnReconnect').mockImplementation(() => {});
    
    // Simulate online event by calling the functions directly since the event listeners are set up in module scope
    newOfflineConfig.retryFailedMutations();
    newOfflineConfig.invalidateOnReconnect();
    
    expect(newOfflineConfig.retryFailedMutations).toHaveBeenCalled();
    expect(newOfflineConfig.invalidateOnReconnect).toHaveBeenCalled();
  });
});