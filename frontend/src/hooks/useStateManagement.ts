import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useAppStore, useLoadingManager, useErrorManager, useSyncManager } from '@/stores/app';
import { useOfflineSupport, useOptimisticUpdates } from './useOfflineSupport';
import { useUIStore } from '@/stores/ui';
import { useErrorStore } from '@/stores/error';

/**
 * Comprehensive state management hook that integrates all state management features
 * including loading states, error handling, offline support, and optimistic updates
 */
export function useStateManagement() {
  const queryClient = useQueryClient();
  const loadingManager = useLoadingManager();
  const errorManager = useErrorManager();
  const syncManager = useSyncManager();
  const offlineSupport = useOfflineSupport();
  const optimisticUpdates = useOptimisticUpdates();
  const { addNotification } = useUIStore();
  const { addError } = useErrorStore();

  // Enhanced operation wrapper that combines loading, error handling, and offline support
  const executeOperation = useCallback(async <T>({
    key,
    operation,
    optimisticUpdate,
    onSuccess,
    onError,
    showNotifications = true,
    retryOnFailure = true,
  }: {
    key: string;
    operation: () => Promise<T>;
    optimisticUpdate?: {
      queryKey: unknown[];
      updateFn: (oldData: unknown) => unknown;
    };
    onSuccess?: (data: T) => void;
    onError?: (error: unknown) => void;
    showNotifications?: boolean;
    retryOnFailure?: boolean;
  }): Promise<T | null> => {
    // Handle optimistic updates if provided
    if (optimisticUpdate && offlineSupport.isOnline) {
      try {
        const result = await optimisticUpdates.performOptimisticUpdate({
          queryKey: optimisticUpdate.queryKey,
          updateFn: optimisticUpdate.updateFn,
          mutationFn: operation,
          onSuccess: (data) => {
            onSuccess?.(data as T);
            if (showNotifications) {
              addNotification({
                type: 'success',
                title: 'Operation Successful',
                message: `${key} completed successfully`,
                read: false,
              });
            }
          },
          onError: (error) => {
            onError?.(error);
            if (showNotifications) {
              addNotification({
                type: 'error',
                title: 'Operation Failed',
                message: error instanceof Error ? error.message : 'An error occurred',
                read: false,
              });
            }
          },
          description: key,
        });
        return result as T;
      } catch (error) {
        return null;
      }
    }

    // Handle regular operations with loading and error management
    return loadingManager.withLoading(key, async () => {
      return errorManager.withErrorHandling(
        key,
        async () => {
          // If offline, queue the operation
          if (!offlineSupport.isOnline && retryOnFailure) {
            offlineSupport.queueOfflineOperation(operation, key);
            if (showNotifications) {
              addNotification({
                type: 'info',
                title: 'Operation Queued',
                message: `${key} will be processed when connection is restored`,
                read: false,
              });
            }
            return null;
          }

          const result = await operation();
          
          onSuccess?.(result);
          if (showNotifications) {
            addNotification({
              type: 'success',
              title: 'Operation Successful',
              message: `${key} completed successfully`,
              read: false,
            });
          }
          
          return result;
        },
        (error) => {
          onError?.(error);
          
          // Add to global error store for tracking
          addError({
            type: 'api',
            message: error instanceof Error ? error.message : 'An error occurred',
            context: key,
            retryable: retryOnFailure,
          });

          if (showNotifications) {
            addNotification({
              type: 'error',
              title: 'Operation Failed',
              message: error instanceof Error ? error.message : 'An error occurred',
              read: false,
            });
          }

          return error instanceof Error ? error.message : 'An error occurred';
        }
      );
    });
  }, [
    loadingManager,
    errorManager,
    offlineSupport,
    optimisticUpdates,
    addNotification,
    addError,
  ]);

  // Batch operations with progress tracking
  const executeBatchOperations = useCallback(async <T>({
    operations,
    batchKey,
    showProgress = true,
    continueOnError = false,
  }: {
    operations: Array<{
      key: string;
      operation: () => Promise<T>;
      onSuccess?: (data: T) => void;
      onError?: (error: unknown) => void;
    }>;
    batchKey: string;
    showProgress?: boolean;
    continueOnError?: boolean;
  }): Promise<Array<T | null>> => {
    const results: Array<T | null> = [];
    let completed = 0;
    let failed = 0;

    if (showProgress) {
      addNotification({
        type: 'info',
        title: 'Batch Operation Started',
        message: `Processing ${operations.length} operations...`,
        read: false,
      });
    }

    for (const { key, operation, onSuccess, onError } of operations) {
      try {
        const result = await executeOperation({
          key: `${batchKey}-${key}`,
          operation,
          onSuccess,
          onError,
          showNotifications: false, // Handle notifications at batch level
        });
        
        results.push(result);
        if (result !== null) {
          completed++;
        } else {
          failed++;
        }
      } catch (error) {
        results.push(null);
        failed++;
        
        if (!continueOnError) {
          break;
        }
      }
    }

    if (showProgress) {
      const notificationType = failed === 0 ? 'success' : completed === 0 ? 'error' : 'warning';
      addNotification({
        type: notificationType,
        title: 'Batch Operation Complete',
        message: `${completed} succeeded, ${failed} failed`,
        read: false,
      });
    }

    return results;
  }, [executeOperation, addNotification]);

  // Cache management utilities
  const cacheManager = useCallback(() => ({
    // Prefetch data for offline use
    prefetchForOffline: async (queryKey: unknown[], queryFn: () => Promise<unknown>) => {
      return offlineSupport.prefetchForOffline(queryKey, queryFn);
    },

    // Invalidate queries with error handling
    invalidateQueries: async (queryKey: unknown[]) => {
      try {
        await queryClient.invalidateQueries({ queryKey });
      } catch (error) {
        addError({
          type: 'unknown',
          message: 'Failed to invalidate cache',
          context: 'cache-invalidation',
          retryable: true,
        });
      }
    },

    // Clear cache with size management
    clearCache: () => {
      queryClient.clear();
      useAppStore.getState().updateCacheSize(0);
      useAppStore.getState().setCacheMetadata('cleared', {
        timestamp: Date.now(),
        ttl: 0,
        version: '1.0.0',
      });
    },

    // Get cache statistics
    getCacheStats: () => {
      const queries = queryClient.getQueryCache().getAll();
      const mutations = queryClient.getMutationCache().getAll();
      
      return {
        queryCount: queries.length,
        mutationCount: mutations.length,
        totalSize: useAppStore.getState().cache.size,
        maxSize: useAppStore.getState().cache.maxSize,
      };
    },
  }), [queryClient, offlineSupport, addError]);

  // Sync management utilities
  const syncUtilities = useCallback(() => ({
    // Force sync all pending operations
    forceSyncAll: async () => {
      if (!offlineSupport.isOnline) {
        addNotification({
          type: 'warning',
          title: 'Sync Failed',
          message: 'Cannot sync while offline',
          read: false,
        });
        return;
      }

      try {
        await offlineSupport.processOfflineQueue();
        syncManager.markSyncComplete();
        
        addNotification({
          type: 'success',
          title: 'Sync Complete',
          message: 'All operations have been synchronized',
          read: false,
        });
      } catch (error) {
        addError({
          type: 'network',
          message: 'Failed to sync operations',
          context: 'force-sync',
          retryable: true,
        });
      }
    },

    // Get sync status
    getSyncStatus: () => {
      const { sync } = useAppStore.getState();
      return {
        ...sync,
        queuedOperations: offlineSupport.offlineQueue.length,
      };
    },

    // Clear failed operations
    clearFailedOperations: () => {
      syncManager.resetFailedOperations();
      offlineSupport.clearOfflineQueue();
    },
  }), [offlineSupport, syncManager, addNotification, addError]);

  return {
    // Core operation execution
    executeOperation,
    executeBatchOperations,
    
    // Individual managers
    loadingManager,
    errorManager,
    syncManager,
    offlineSupport,
    optimisticUpdates,
    
    // Utility functions
    cacheManager: cacheManager(),
    syncUtilities: syncUtilities(),
    
    // State selectors
    isLoading: loadingManager.isLoading,
    hasError: errorManager.hasError,
    isOnline: offlineSupport.isOnline,
    
    // Quick actions
    clearAllErrors: () => {
      errorManager.clearAllErrors();
      useErrorStore.getState().clearErrors();
    },
    clearAllLoading: loadingManager.clearLoading,
    reset: () => {
      useAppStore.getState().reset();
      queryClient.clear();
    },
  };
}

// Specialized hooks for common patterns
export function useApiOperation<T>(key: string) {
  const stateManager = useStateManagement();
  
  return useCallback((
    operation: () => Promise<T>,
    options?: {
      optimisticUpdate?: {
        queryKey: unknown[];
        updateFn: (oldData: unknown) => unknown;
      };
      onSuccess?: (data: T) => void;
      onError?: (error: unknown) => void;
      showNotifications?: boolean;
    }
  ) => {
    return stateManager.executeOperation({
      key,
      operation,
      ...options,
    });
  }, [key, stateManager]);
}

export function useBatchOperation(batchKey: string) {
  const stateManager = useStateManagement();
  
  return useCallback(<T>(
    operations: Array<{
      key: string;
      operation: () => Promise<T>;
      onSuccess?: (data: T) => void;
      onError?: (error: unknown) => void;
    }>,
    options?: {
      showProgress?: boolean;
      continueOnError?: boolean;
    }
  ) => {
    return stateManager.executeBatchOperations({
      operations,
      batchKey,
      ...options,
    });
  }, [batchKey, stateManager]);
}