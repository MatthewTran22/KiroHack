import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useErrorStore } from '@/stores/error';
import { useUIStore } from '@/stores/ui';

interface OfflineQueueItem {
  id: string;
  operation: () => Promise<unknown>;
  description: string;
  timestamp: Date;
  retries: number;
  maxRetries: number;
}

export function useOfflineSupport() {
  const queryClient = useQueryClient();
  const { setOfflineStatus, addToRetryQueue, retryAllOperations } = useErrorStore();
  const { addNotification } = useUIStore();
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [offlineQueue, setOfflineQueue] = useState<OfflineQueueItem[]>([]);

  // Network status detection
  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true);
      setOfflineStatus(false);
      
      // Show reconnection notification
      addNotification({
        type: 'success',
        title: 'Back Online',
        message: 'Connection restored. Syncing data...',
      });

      // Retry failed operations
      retryAllOperations();
      
      // Invalidate stale queries
      queryClient.invalidateQueries({
        predicate: (query) => {
          const now = Date.now();
          const staleTime = query.options.staleTime ?? 0;
          return now - (query.state.dataUpdatedAt ?? 0) > staleTime;
        },
      });

      // Process offline queue
      processOfflineQueue();
    };

    const handleOffline = () => {
      setIsOnline(false);
      setOfflineStatus(true);
      
      // Show offline notification
      addNotification({
        type: 'warning',
        title: 'Connection Lost',
        message: 'You are now offline. Changes will be synced when connection is restored.',
      });
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [queryClient, setOfflineStatus, addNotification, retryAllOperations]);

  // Process offline queue when back online
  const processOfflineQueue = async () => {
    if (offlineQueue.length === 0) return;

    const results = await Promise.allSettled(
      offlineQueue.map(async (item) => {
        try {
          await item.operation();
          return { success: true, id: item.id };
        } catch (error) {
          if (item.retries < item.maxRetries) {
            return { success: false, id: item.id, retry: true };
          }
          return { success: false, id: item.id, retry: false };
        }
      })
    );

    const successful = results.filter(
      (result) => result.status === 'fulfilled' && result.value.success
    ).length;

    const failed = results.filter(
      (result) => result.status === 'fulfilled' && !result.value.success && !result.value.retry
    ).length;

    const toRetry = results
      .filter(
        (result) => result.status === 'fulfilled' && !result.value.success && result.value.retry
      )
      .map((result) => result.status === 'fulfilled' ? result.value.id : null)
      .filter(Boolean) as string[];

    // Update queue - remove successful and failed, increment retries for retry items
    setOfflineQueue((prev) => 
      prev
        .filter((item) => !results.some(
          (result) => result.status === 'fulfilled' && result.value.success && result.value.id === item.id
        ))
        .filter((item) => !results.some(
          (result) => result.status === 'fulfilled' && !result.value.success && !result.value.retry && result.value.id === item.id
        ))
        .map((item) => 
          toRetry.includes(item.id) 
            ? { ...item, retries: item.retries + 1 }
            : item
        )
    );

    // Show sync results
    if (successful > 0) {
      addNotification({
        type: 'success',
        title: 'Sync Complete',
        message: `${successful} operation${successful > 1 ? 's' : ''} synced successfully.`,
      });
    }

    if (failed > 0) {
      addNotification({
        type: 'error',
        title: 'Sync Failed',
        message: `${failed} operation${failed > 1 ? 's' : ''} could not be synced.`,
      });
    }
  };

  // Add operation to offline queue
  const queueOfflineOperation = (
    operation: () => Promise<unknown>,
    description: string,
    maxRetries: number = 3
  ) => {
    const item: OfflineQueueItem = {
      id: crypto.randomUUID(),
      operation,
      description,
      timestamp: new Date(),
      retries: 0,
      maxRetries,
    };

    setOfflineQueue((prev) => [...prev, item]);
    
    addNotification({
      type: 'info',
      title: 'Operation Queued',
      message: `${description} will be processed when connection is restored.`,
    });

    return item.id;
  };

  // Remove operation from queue
  const removeFromOfflineQueue = (id: string) => {
    setOfflineQueue((prev) => prev.filter((item) => item.id !== id));
  };

  // Clear all queued operations
  const clearOfflineQueue = () => {
    setOfflineQueue([]);
  };

  // Get cached data for offline use
  const getCachedData = <T>(queryKey: unknown[]) => {
    return queryClient.getQueryData<T>(queryKey);
  };

  // Set cached data for offline use
  const setCachedData = <T>(queryKey: unknown[], data: T) => {
    queryClient.setQueryData(queryKey, data);
  };

  // Check if query has cached data
  const hasCachedData = (queryKey: unknown[]) => {
    const data = queryClient.getQueryData(queryKey);
    return data !== undefined;
  };

  // Prefetch data for offline use
  const prefetchForOffline = async (queryKey: unknown[], queryFn: () => Promise<unknown>) => {
    if (!hasCachedData(queryKey)) {
      try {
        await queryClient.prefetchQuery({
          queryKey,
          queryFn,
          staleTime: 10 * 60 * 1000, // 10 minutes
        });
      } catch (error) {
        console.warn('Failed to prefetch data for offline use:', error);
      }
    }
  };

  return {
    isOnline,
    offlineQueue,
    queueOfflineOperation,
    removeFromOfflineQueue,
    clearOfflineQueue,
    getCachedData,
    setCachedData,
    hasCachedData,
    prefetchForOffline,
    processOfflineQueue,
  };
}

// Hook for optimistic updates
export function useOptimisticUpdates() {
  const queryClient = useQueryClient();
  const { isOnline, queueOfflineOperation } = useOfflineSupport();

  const performOptimisticUpdate = async <T>({
    queryKey,
    updateFn,
    mutationFn,
    onSuccess,
    onError,
    description,
  }: {
    queryKey: unknown[];
    updateFn: (oldData: T | undefined) => T;
    mutationFn: () => Promise<T>;
    onSuccess?: (data: T) => void;
    onError?: (error: unknown, rollbackData: T | undefined) => void;
    description: string;
  }) => {
    // Cancel outgoing refetches
    await queryClient.cancelQueries({ queryKey });

    // Snapshot previous value
    const previousData = queryClient.getQueryData<T>(queryKey);

    // Optimistically update
    const optimisticData = updateFn(previousData);
    queryClient.setQueryData(queryKey, optimisticData);

    if (isOnline) {
      // Perform mutation immediately if online
      try {
        const result = await mutationFn();
        queryClient.setQueryData(queryKey, result);
        onSuccess?.(result);
        return result;
      } catch (error) {
        // Rollback on error
        queryClient.setQueryData(queryKey, previousData);
        onError?.(error, previousData);
        throw error;
      }
    } else {
      // Queue for later if offline
      queueOfflineOperation(
        async () => {
          const result = await mutationFn();
          queryClient.setQueryData(queryKey, result);
          onSuccess?.(result);
          return result;
        },
        description
      );
      
      return optimisticData;
    }
  };

  return { performOptimisticUpdate };
}