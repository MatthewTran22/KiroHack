import { QueryClient, DefaultOptions, MutationCache, QueryCache } from '@tanstack/react-query';
import { APIError } from './api';
import { tokenManager } from './auth';
import { useAppStore } from '@/stores/app';

// Default query options
const queryConfig: DefaultOptions = {
  queries: {
    retry: (failureCount, error) => {
      // Don't retry on 4xx errors except 401 (which might be token expiry)
      if (error instanceof APIError) {
        if (error.status >= 400 && error.status < 500 && error.status !== 401) {
          return false;
        }
      }
      return failureCount < 3;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
    staleTime: 5 * 60 * 1000, // 5 minutes
    gcTime: 10 * 60 * 1000, // 10 minutes (formerly cacheTime)
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
  },
  mutations: {
    retry: (failureCount, error) => {
      // Don't retry mutations on client errors
      if (error instanceof APIError && error.status >= 400 && error.status < 500) {
        return false;
      }
      return failureCount < 2;
    },
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
  },
};

// Enhanced query cache with state management integration
const queryCache = new QueryCache({
  onError: (error, query) => {
    const { setError } = useAppStore.getState();
    const queryKey = Array.isArray(query.queryKey) ? query.queryKey.join('-') : 'unknown';
    setError(`query-${queryKey}`, handleQueryError(error).message);
  },
  onSuccess: (data, query) => {
    const { clearError } = useAppStore.getState();
    const queryKey = Array.isArray(query.queryKey) ? query.queryKey.join('-') : 'unknown';
    clearError(`query-${queryKey}`);
  },
});

// Enhanced mutation cache with state management integration
const mutationCache = new MutationCache({
  onError: (error, variables, context, mutation) => {
    const { setError, incrementFailedOperations } = useAppStore.getState();
    const mutationKey = mutation.options.mutationKey 
      ? Array.isArray(mutation.options.mutationKey) 
        ? mutation.options.mutationKey.join('-') 
        : 'unknown'
      : 'unknown';
    
    setError(`mutation-${mutationKey}`, handleQueryError(error).message);
    incrementFailedOperations();
  },
  onSuccess: (data, variables, context, mutation) => {
    const { clearError, decrementPendingOperations } = useAppStore.getState();
    const mutationKey = mutation.options.mutationKey 
      ? Array.isArray(mutation.options.mutationKey) 
        ? mutation.options.mutationKey.join('-') 
        : 'unknown'
      : 'unknown';
    
    clearError(`mutation-${mutationKey}`);
    decrementPendingOperations();
  },
  onMutate: (variables, mutation) => {
    const { incrementPendingOperations } = useAppStore.getState();
    incrementPendingOperations();
  },
});

// Create query client with enhanced error handling and state management
export const queryClient = new QueryClient({
  defaultOptions: queryConfig,
  queryCache,
  mutationCache,
});

// Query keys factory for consistent key management
export const queryKeys = {
  // Authentication
  auth: {
    user: ['auth', 'user'] as const,
    session: ['auth', 'session'] as const,
  },
  
  // Documents
  documents: {
    all: ['documents'] as const,
    lists: () => [...queryKeys.documents.all, 'list'] as const,
    list: (filters?: Record<string, unknown>) => 
      [...queryKeys.documents.lists(), filters] as const,
    details: () => [...queryKeys.documents.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.documents.details(), id] as const,
    search: (query: string) => [...queryKeys.documents.all, 'search', query] as const,
  },
  
  // Consultations
  consultations: {
    all: ['consultations'] as const,
    lists: () => [...queryKeys.consultations.all, 'list'] as const,
    list: (filters?: Record<string, unknown>) => 
      [...queryKeys.consultations.lists(), filters] as const,
    details: () => [...queryKeys.consultations.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.consultations.details(), id] as const,
    sessions: () => [...queryKeys.consultations.all, 'sessions'] as const,
    session: (id: string) => [...queryKeys.consultations.sessions(), id] as const,
    messages: (sessionId: string) => 
      [...queryKeys.consultations.session(sessionId), 'messages'] as const,
  },
  
  // Knowledge base
  knowledge: {
    all: ['knowledge'] as const,
    search: (query: string) => [...queryKeys.knowledge.all, 'search', query] as const,
    suggestions: (context: string) => 
      [...queryKeys.knowledge.all, 'suggestions', context] as const,
  },
  
  // Audit
  audit: {
    all: ['audit'] as const,
    logs: (filters?: Record<string, unknown>) => 
      [...queryKeys.audit.all, 'logs', filters] as const,
    reports: () => [...queryKeys.audit.all, 'reports'] as const,
    report: (id: string) => [...queryKeys.audit.reports(), id] as const,
  },
  
  // Notifications
  notifications: {
    all: ['notifications'] as const,
    unread: () => [...queryKeys.notifications.all, 'unread'] as const,
  },
} as const;

// Offline support utilities
export const offlineConfig = {
  // Network status detection
  isOnline: () => typeof navigator !== 'undefined' ? navigator.onLine : true,
  
  // Retry failed mutations when back online
  retryFailedMutations: () => {
    queryClient.getMutationCache().getAll().forEach((mutation) => {
      if (mutation.state.status === 'error') {
        mutation.continue();
      }
    });
  },
  
  // Invalidate stale queries when back online
  invalidateOnReconnect: () => {
    queryClient.invalidateQueries({
      predicate: (query) => {
        const now = Date.now();
        const staleTime = query.options.staleTime ?? 0;
        return now - (query.state.dataUpdatedAt ?? 0) > staleTime;
      },
    });
  },
};

// Set up network status listeners
if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    offlineConfig.retryFailedMutations();
    offlineConfig.invalidateOnReconnect();
  });
  
  window.addEventListener('offline', () => {
    // Could add offline-specific logic here
    console.log('Application is now offline');
  });
}

// Error boundary for query errors
export const handleQueryError = (error: unknown) => {
  if (error instanceof APIError) {
    // Handle authentication errors
    if (error.status === 401) {
      tokenManager.clearTokens();
      // Redirect to login will be handled by middleware
      return;
    }
    
    // Handle other API errors
    console.error('Query error:', error.message, error.details);
    return error;
  }
  
  // Handle network errors
  console.error('Network error:', error);
  return new Error('Network error occurred');
};