import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { AppState, LoadingState, ErrorState, SyncStatus } from '@/types';

interface AppStore extends AppState {
  // Loading state management
  setLoading: (key: string, loading: boolean) => void;
  clearLoading: () => void;
  isLoading: (key: string) => boolean;
  
  // Error state management
  setError: (key: string, error: string | null) => void;
  clearError: (key: string) => void;
  clearAllErrors: () => void;
  hasError: (key: string) => boolean;
  
  // Sync status management
  setSyncStatus: (status: Partial<SyncStatus>) => void;
  incrementPendingOperations: () => void;
  decrementPendingOperations: () => void;
  incrementFailedOperations: () => void;
  resetFailedOperations: () => void;
  
  // Cache management
  updateCacheSize: (size: number) => void;
  setCacheMetadata: (key: string, metadata: { timestamp: number; ttl: number; version: string }) => void;
  getCacheMetadata: (key: string) => { timestamp: number; ttl: number; version: string } | null;
  clearExpiredCache: () => void;
  
  // Global state reset
  reset: () => void;
}

const initialState: AppState = {
  loading: {},
  errors: {},
  sync: {
    isOnline: typeof navigator !== 'undefined' ? navigator.onLine : true,
    lastSync: null,
    pendingOperations: 0,
    failedOperations: 0,
  },
  cache: {
    metadata: {},
    size: 0,
    maxSize: 50 * 1024 * 1024, // 50MB default
  },
};

export const useAppStore = create<AppStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        // Loading state management
        setLoading: (key: string, loading: boolean) => {
          set((state) => ({
            loading: {
              ...state.loading,
              [key]: loading,
            },
          }));
        },

        clearLoading: () => {
          set({ loading: {} });
        },

        isLoading: (key: string) => {
          return get().loading[key] || false;
        },

        // Error state management
        setError: (key: string, error: string | null) => {
          set((state) => ({
            errors: {
              ...state.errors,
              [key]: error,
            },
          }));
        },

        clearError: (key: string) => {
          set((state) => {
            const { [key]: _, ...rest } = state.errors;
            return { errors: rest };
          });
        },

        clearAllErrors: () => {
          set({ errors: {} });
        },

        hasError: (key: string) => {
          return Boolean(get().errors[key]);
        },

        // Sync status management
        setSyncStatus: (status: Partial<SyncStatus>) => {
          set((state) => ({
            sync: {
              ...state.sync,
              ...status,
            },
          }));
        },

        incrementPendingOperations: () => {
          set((state) => ({
            sync: {
              ...state.sync,
              pendingOperations: state.sync.pendingOperations + 1,
            },
          }));
        },

        decrementPendingOperations: () => {
          set((state) => ({
            sync: {
              ...state.sync,
              pendingOperations: Math.max(0, state.sync.pendingOperations - 1),
            },
          }));
        },

        incrementFailedOperations: () => {
          set((state) => ({
            sync: {
              ...state.sync,
              failedOperations: state.sync.failedOperations + 1,
            },
          }));
        },

        resetFailedOperations: () => {
          set((state) => ({
            sync: {
              ...state.sync,
              failedOperations: 0,
            },
          }));
        },

        // Cache management
        updateCacheSize: (size: number) => {
          set((state) => ({
            cache: {
              ...state.cache,
              size,
            },
          }));
        },

        setCacheMetadata: (key: string, metadata) => {
          set((state) => ({
            cache: {
              ...state.cache,
              metadata: {
                ...state.cache.metadata,
                [key]: metadata,
              },
            },
          }));
        },

        getCacheMetadata: (key: string) => {
          return get().cache.metadata[key] || null;
        },

        clearExpiredCache: () => {
          const now = Date.now();
          set((state) => {
            const validMetadata: Record<string, { timestamp: number; ttl: number; version: string }> = {};
            
            Object.entries(state.cache.metadata).forEach(([key, metadata]) => {
              if (now - metadata.timestamp < metadata.ttl) {
                validMetadata[key] = metadata;
              }
            });

            return {
              cache: {
                ...state.cache,
                metadata: validMetadata,
              },
            };
          });
        },

        // Global state reset
        reset: () => {
          set(initialState);
        },
      }),
      {
        name: 'app-store',
        partialize: (state) => ({
          sync: {
            lastSync: state.sync.lastSync,
          },
          cache: {
            metadata: state.cache.metadata,
            maxSize: state.cache.maxSize,
          },
        }),
      }
    ),
    {
      name: 'app-store',
    }
  )
);

// Selectors for common use cases
export const useLoadingState = () => useAppStore((state) => state.loading);
export const useErrorState = () => useAppStore((state) => state.errors);
export const useSyncStatus = () => useAppStore((state) => state.sync);
export const useCacheInfo = () => useAppStore((state) => state.cache);

// Hook for managing loading states
export const useLoadingManager = () => {
  const { setLoading, clearLoading, isLoading } = useAppStore();
  
  return {
    setLoading,
    clearLoading,
    isLoading,
    withLoading: async <T>(key: string, operation: () => Promise<T>): Promise<T> => {
      setLoading(key, true);
      try {
        const result = await operation();
        return result;
      } finally {
        setLoading(key, false);
      }
    },
  };
};

// Hook for managing error states
export const useErrorManager = () => {
  const { setError, clearError, clearAllErrors, hasError } = useAppStore();
  
  return {
    setError,
    clearError,
    clearAllErrors,
    hasError,
    withErrorHandling: async <T>(
      key: string,
      operation: () => Promise<T>,
      errorHandler?: (error: unknown) => string
    ): Promise<T | null> => {
      clearError(key);
      try {
        const result = await operation();
        return result;
      } catch (error) {
        const errorMessage = errorHandler 
          ? errorHandler(error)
          : error instanceof Error 
            ? error.message 
            : 'An unknown error occurred';
        setError(key, errorMessage);
        return null;
      }
    },
  };
};

// Hook for sync status management
export const useSyncManager = () => {
  const { 
    setSyncStatus, 
    incrementPendingOperations, 
    decrementPendingOperations,
    incrementFailedOperations,
    resetFailedOperations 
  } = useAppStore();
  
  return {
    setSyncStatus,
    incrementPendingOperations,
    decrementPendingOperations,
    incrementFailedOperations,
    resetFailedOperations,
    markSyncComplete: () => {
      setSyncStatus({ lastSync: new Date() });
      resetFailedOperations();
    },
    setOnlineStatus: (isOnline: boolean) => {
      setSyncStatus({ isOnline });
    },
  };
};