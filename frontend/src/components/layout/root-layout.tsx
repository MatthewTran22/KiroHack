"use client";

import React, { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Header } from './header';
import { Sidebar } from './sidebar';
import { useAuthStore } from '@/stores/auth';
import { tokenManager } from '@/lib/auth';
import { cn } from '@/lib/utils';

interface RootLayoutProps {
  children: React.ReactNode;
}

// Routes that don't need the full layout (auth pages)
const publicRoutes = ['/login', '/logout', '/mfa-setup'];

export function RootLayout({ children }: RootLayoutProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { isAuthenticated, checkAuth } = useAuthStore();

  const isPublicRoute = publicRoutes.includes(pathname);

  useEffect(() => {
    // Quick token check first - if no token, go directly to login
    const token = tokenManager.getToken();
    if (!token && !isPublicRoute) {
      router.push('/login');
      return;
    }

    // Only do full auth check if we have a token
    if (token) {
      checkAuth();
    }
  }, [checkAuth, isPublicRoute, router]);

  useEffect(() => {
    // Redirect to dashboard if authenticated and on login page
    if (isAuthenticated && pathname === '/login') {
      router.push('/dashboard');
      return;
    }
  }, [isAuthenticated, pathname, router]);

  const handleSearch = (query: string) => {
    // TODO: Implement global search functionality
    console.log('Search query:', query);
  };

  const handleNewConsultation = () => {
    router.push('/consultation');
  };

  // Show loading state while checking authentication
  if (!isAuthenticated && !isPublicRoute) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" role="status" aria-label="Loading"></div>
          <p className="text-sm text-muted-foreground">Loading...</p>
        </div>
      </div>
    );
  }

  // Render public routes without layout
  if (isPublicRoute) {
    return <>{children}</>;
  }

  // Render authenticated layout
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />

      <div className="flex flex-1 flex-col overflow-hidden min-w-0">
        <Header
          onSearch={handleSearch}
          onNewConsultation={handleNewConsultation}
        />

        <main
          className={cn(
            "flex-1 overflow-y-auto bg-background transition-all duration-200",
            "focus:outline-none",
            "scrollbar-thin scrollbar-thumb-muted scrollbar-track-transparent"
          )}
          tabIndex={-1}
          role="main"
          aria-label="Main content"
        >
          <div className="container mx-auto p-3 sm:p-4 md:p-6 lg:p-8 max-w-7xl">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}

// Error Boundary Component
interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class LayoutErrorBoundary extends React.Component<
  { children: React.ReactNode },
  ErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Layout Error:', error, errorInfo);
    // TODO: Send error to monitoring service
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-screen items-center justify-center">
          <div className="text-center">
            <h2 className="text-2xl font-bold text-destructive mb-4">
              Something went wrong
            </h2>
            <p className="text-muted-foreground mb-4">
              An error occurred while loading the application.
            </p>
            <button
              onClick={() => window.location.reload()}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
            >
              Reload Page
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}