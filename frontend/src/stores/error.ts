import React from 'react';
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { APIError } from '@/lib/api';

export interface AppError {
  id: string;
  type: 'network' | 'api' | 'validation' | 'auth' | 'unknown';
  message: string;
  details?: Record<string, unknown>;
  timestamp: Date;
  context?: string; // Which feature/component the error occurred in
  retryable: boolean;
  dismissed: boolean;
}

export interface ErrorRecoveryAction {
  label: string;
  action: () => void | Promise<void>;
  variant?: 'default' | 'destructive';
}

interface ErrorState {
  errors: AppError[];
  globalError: AppError | null; // Critical errors that should block the UI
  isOffline: boolean;
  retryQueue: Array<{
    id: string;
    operation: () => Promise<unknown>;
    maxRetries: number;
    currentRetries: number;
  }>;
}

interface ErrorActions {
  // Error management
  addError: (error: Omit<AppError, 'id' | 'timestamp' | 'dismissed'>) => string;
  removeError: (id: string) => void;
  dismissError: (id: string) => void;
  clearErrors: () => void;
  setGlobalError: (error: AppError | null) => void;
  
  // Network status
  setOfflineStatus: (offline: boolean) => void;
  
  // Retry mechanism
  addToRetryQueue: (operation: () => Promise<unknown>, maxRetries?: number) => string;
  retryOperation: (id: string) => Promise<void>;
  removeFromRetryQueue: (id: string) => void;
  retryAllOperations: () => Promise<void>;
  
  // Error handling utilities
  handleAPIError: (error: unknown, context?: string) => string;
  handleNetworkError: (error: unknown, context?: string) => string;
  handleValidationError: (errors: Record<string, string>, context?: string) => string;
}

type ErrorStore = ErrorState & ErrorActions;

const initialState: ErrorState = {
  errors: [],
  globalError: null,
  isOffline: false,
  retryQueue: [],
};

export const useErrorStore = create<ErrorStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // Error management
      addError: (errorData) => {
        const error: AppError = {
          ...errorData,
          id: crypto.randomUUID(),
          timestamp: new Date(),
          dismissed: false,
        };

        set((state) => ({
          errors: [error, ...state.errors].slice(0, 50), // Keep only latest 50 errors
        }));

        // Auto-dismiss non-critical errors after 10 seconds
        if (error.type !== 'auth' && !error.context?.includes('critical')) {
          setTimeout(() => {
            get().dismissError(error.id);
          }, 10000);
        }

        return error.id;
      },

      removeError: (id: string) => {
        set((state) => ({
          errors: state.errors.filter((error) => error.id !== id),
        }));
      },

      dismissError: (id: string) => {
        set((state) => ({
          errors: state.errors.map((error) =>
            error.id === id ? { ...error, dismissed: true } : error
          ),
        }));
      },

      clearErrors: () => {
        set({ errors: [] });
      },

      setGlobalError: (error: AppError | null) => {
        set({ globalError: error });
      },

      // Network status
      setOfflineStatus: (offline: boolean) => {
        set({ isOffline: offline });
        
        if (!offline) {
          // When back online, retry queued operations
          get().retryAllOperations();
        }
      },

      // Retry mechanism
      addToRetryQueue: (operation, maxRetries = 3) => {
        const id = crypto.randomUUID();
        
        set((state) => ({
          retryQueue: [
            ...state.retryQueue,
            {
              id,
              operation,
              maxRetries,
              currentRetries: 0,
            },
          ],
        }));

        return id;
      },

      retryOperation: async (id: string) => {
        const { retryQueue } = get();
        const item = retryQueue.find((item) => item.id === id);
        
        if (!item) return;

        try {
          await item.operation();
          get().removeFromRetryQueue(id);
        } catch (error) {
          const updatedItem = {
            ...item,
            currentRetries: item.currentRetries + 1,
          };

          if (updatedItem.currentRetries > updatedItem.maxRetries) {
            get().removeFromRetryQueue(id);
            get().handleAPIError(error, 'retry-failed');
          } else {
            set((state) => ({
              retryQueue: state.retryQueue.map((queueItem) =>
                queueItem.id === id ? updatedItem : queueItem
              ),
            }));
          }
        }
      },

      removeFromRetryQueue: (id: string) => {
        set((state) => ({
          retryQueue: state.retryQueue.filter((item) => item.id !== id),
        }));
      },

      retryAllOperations: async () => {
        const { retryQueue } = get();
        
        await Promise.allSettled(
          retryQueue.map((item) => get().retryOperation(item.id))
        );
      },

      // Error handling utilities
      handleAPIError: (error: unknown, context?: string) => {
        if (error instanceof APIError) {
          const errorType = error.status === 401 ? 'auth' : 'api';
          
          return get().addError({
            type: errorType,
            message: error.message,
            details: {
              status: error.status,
              code: error.code,
              ...error.details,
            },
            context,
            retryable: error.status >= 500 || error.status === 0, // Server errors or network errors
          });
        }

        return get().addError({
          type: 'unknown',
          message: error instanceof Error ? error.message : 'An unknown error occurred',
          context,
          retryable: false,
        });
      },

      handleNetworkError: (error: unknown, context?: string) => {
        return get().addError({
          type: 'network',
          message: 'Network connection failed. Please check your internet connection.',
          details: {
            originalError: error instanceof Error ? error.message : String(error),
          },
          context,
          retryable: true,
        });
      },

      handleValidationError: (errors: Record<string, string>, context?: string) => {
        const message = Object.values(errors).join(', ');
        
        return get().addError({
          type: 'validation',
          message: `Validation failed: ${message}`,
          details: { validationErrors: errors },
          context,
          retryable: false,
        });
      },
    }),
    {
      name: 'error-store',
    }
  )
);

// Error boundary hook
export function useErrorBoundary() {
  const { addError, setGlobalError } = useErrorStore();

  const captureError = (error: Error, errorInfo?: { componentStack: string }) => {
    const errorId = addError({
      type: 'unknown',
      message: error.message,
      details: {
        stack: error.stack,
        componentStack: errorInfo?.componentStack,
      },
      context: 'error-boundary',
      retryable: false,
    });

    // Set as global error if it's critical
    if (error.message.includes('ChunkLoadError') || error.message.includes('Loading chunk')) {
      setGlobalError({
        id: errorId,
        type: 'unknown',
        message: 'Application update detected. Please refresh the page.',
        timestamp: new Date(),
        context: 'chunk-load-error',
        retryable: true,
        dismissed: false,
      });
    }
  };

  return { captureError };
}

// Network status hook
export function useNetworkStatus() {
  const { isOffline, setOfflineStatus } = useErrorStore();

  // Set up network status listeners
  React.useEffect(() => {
    const handleOnline = () => setOfflineStatus(false);
    const handleOffline = () => setOfflineStatus(true);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    // Set initial status
    setOfflineStatus(!navigator.onLine);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [setOfflineStatus]);

  return { isOffline };
}