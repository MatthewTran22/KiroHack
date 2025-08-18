'use client';

import React, { Component, ErrorInfo, ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AlertTriangle, RefreshCw } from 'lucide-react';
import { useErrorStore } from '@/stores/error';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

class ErrorBoundaryClass extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error, errorInfo: null };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo });
    
    // Log error to error store
    const { addError, setGlobalError } = useErrorStore.getState();
    
    const errorId = addError({
      type: 'unknown',
      message: error.message,
      details: {
        stack: error.stack,
        componentStack: errorInfo.componentStack,
      },
      context: 'error-boundary',
      retryable: true,
    });

    // Handle specific error types
    if (error.message.includes('ChunkLoadError') || error.message.includes('Loading chunk')) {
      setGlobalError({
        id: errorId,
        type: 'unknown',
        message: 'Application update detected. Please refresh the page to continue.',
        timestamp: new Date(),
        context: 'chunk-load-error',
        retryable: true,
        dismissed: false,
      });
    }

    // Call custom error handler
    this.props.onError?.(error, errorInfo);

    // Log to console in development
    if (process.env.NODE_ENV === 'development') {
      console.error('Error Boundary caught an error:', error, errorInfo);
    }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleRefresh = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const isChunkError = this.state.error?.message.includes('ChunkLoadError') || 
                          this.state.error?.message.includes('Loading chunk');

      return (
        <div className="min-h-screen flex items-center justify-center p-4 bg-background">
          <Card className="w-full max-w-md">
            <CardHeader className="text-center">
              <div className="mx-auto mb-4 w-12 h-12 rounded-full bg-destructive/10 flex items-center justify-center">
                <AlertTriangle className="w-6 h-6 text-destructive" />
              </div>
              <CardTitle className="text-xl">
                {isChunkError ? 'Update Available' : 'Something went wrong'}
              </CardTitle>
              <CardDescription>
                {isChunkError 
                  ? 'A new version of the application is available. Please refresh to continue.'
                  : 'An unexpected error occurred. You can try refreshing the page or contact support if the problem persists.'
                }
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {process.env.NODE_ENV === 'development' && this.state.error && (
                <details className="text-sm">
                  <summary className="cursor-pointer font-medium mb-2">
                    Error Details (Development)
                  </summary>
                  <pre className="whitespace-pre-wrap text-xs bg-muted p-2 rounded overflow-auto max-h-32">
                    {this.state.error.message}
                    {this.state.error.stack && `\n\n${this.state.error.stack}`}
                  </pre>
                </details>
              )}
              
              <div className="flex gap-2">
                <Button 
                  onClick={this.handleRefresh}
                  className="flex-1"
                  variant={isChunkError ? 'default' : 'outline'}
                >
                  <RefreshCw className="w-4 h-4 mr-2" />
                  Refresh Page
                </Button>
                
                {!isChunkError && (
                  <Button 
                    onClick={this.handleRetry}
                    variant="default"
                    className="flex-1"
                  >
                    Try Again
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      );
    }

    return this.props.children;
  }
}

// Hook-based error boundary for functional components
export function useErrorHandler() {
  const { addError } = useErrorStore();

  return React.useCallback((error: Error, context?: string) => {
    addError({
      type: 'unknown',
      message: error.message,
      details: { stack: error.stack },
      context,
      retryable: false,
    });
  }, [addError]);
}

// Main error boundary component
export function ErrorBoundary({ children, fallback, onError }: Props) {
  return (
    <ErrorBoundaryClass fallback={fallback} onError={onError}>
      {children}
    </ErrorBoundaryClass>
  );
}

// Feature-specific error boundary
interface FeatureErrorBoundaryProps {
  children: ReactNode;
  feature: string;
  fallback?: ReactNode;
}

export function FeatureErrorBoundary({ children, feature, fallback }: FeatureErrorBoundaryProps) {
  const defaultFallback = (
    <Card className="w-full">
      <CardContent className="flex flex-col items-center justify-center py-8">
        <AlertTriangle className="w-8 h-8 text-muted-foreground mb-4" />
        <p className="text-sm text-muted-foreground text-center">
          Unable to load {feature}. Please try refreshing the page.
        </p>
        <Button 
          variant="outline" 
          size="sm" 
          className="mt-4"
          onClick={() => window.location.reload()}
        >
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </CardContent>
    </Card>
  );

  return (
    <ErrorBoundary 
      fallback={fallback || defaultFallback}
      onError={(error, errorInfo) => {
        console.error(`Error in ${feature}:`, error, errorInfo);
      }}
    >
      {children}
    </ErrorBoundary>
  );
}