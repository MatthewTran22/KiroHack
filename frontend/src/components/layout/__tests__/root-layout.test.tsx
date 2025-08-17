import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { useRouter, usePathname } from 'next/navigation';
import { RootLayout, LayoutErrorBoundary } from '../root-layout';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';

// Mock dependencies
jest.mock('next/navigation');
jest.mock('@/stores/auth');
jest.mock('@/stores/ui');
jest.mock('../header', () => ({
  Header: ({ onSearch, onNewConsultation }: { onSearch?: (query: string) => void; onNewConsultation?: () => void }) => (
    <div data-testid="header">
      <button onClick={() => onSearch('test')}>Search</button>
      <button onClick={onNewConsultation}>New Consultation</button>
    </div>
  ),
}));
jest.mock('../sidebar', () => ({
  Sidebar: () => <div data-testid="sidebar">Sidebar</div>,
}));

const mockRouter = {
  push: jest.fn(),
};

const mockUser = {
  id: '1',
  name: 'John Doe',
  email: 'john@example.com',
  role: 'admin' as const,
  department: 'IT',
  mfaEnabled: true,
  createdAt: new Date(),
  updatedAt: new Date(),
};

const mockUseAuthStore = {
  user: mockUser,
  isAuthenticated: true,
  checkAuth: jest.fn(),
};

const mockUseUIStore = {
  sidebarOpen: true,
};

describe('RootLayout', () => {
  beforeEach(() => {
    (useRouter as jest.Mock).mockReturnValue(mockRouter);
    (usePathname as jest.Mock).mockReturnValue('/dashboard');
    (useAuthStore as unknown as jest.Mock).mockReturnValue(mockUseAuthStore);
    (useUIStore as unknown as jest.Mock).mockReturnValue(mockUseUIStore);
    jest.clearAllMocks();
  });

  it('renders authenticated layout for authenticated users', () => {
    render(
      <RootLayout>
        <div>Test Content</div>
      </RootLayout>
    );
    
    expect(screen.getByTestId('sidebar')).toBeInTheDocument();
    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByText('Test Content')).toBeInTheDocument();
  });

  it('renders public routes without layout', () => {
    (usePathname as jest.Mock).mockReturnValue('/login');
    
    render(
      <RootLayout>
        <div>Login Content</div>
      </RootLayout>
    );
    
    expect(screen.queryByTestId('sidebar')).not.toBeInTheDocument();
    expect(screen.queryByTestId('header')).not.toBeInTheDocument();
    expect(screen.getByText('Login Content')).toBeInTheDocument();
  });

  it('shows loading state for unauthenticated users on protected routes', () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      ...mockUseAuthStore,
      isAuthenticated: false,
    });
    
    render(
      <RootLayout>
        <div>Protected Content</div>
      </RootLayout>
    );
    
    // Check for loading spinner by class instead of role
    expect(document.querySelector('.animate-spin')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('redirects to login for unauthenticated users', async () => {
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      ...mockUseAuthStore,
      isAuthenticated: false,
    });
    
    render(
      <RootLayout>
        <div>Protected Content</div>
      </RootLayout>
    );
    
    await waitFor(() => {
      expect(mockRouter.push).toHaveBeenCalledWith('/login');
    });
  });

  it('redirects to dashboard for authenticated users on login page', async () => {
    (usePathname as jest.Mock).mockReturnValue('/login');
    
    render(
      <RootLayout>
        <div>Login Content</div>
      </RootLayout>
    );
    
    await waitFor(() => {
      expect(mockRouter.push).toHaveBeenCalledWith('/dashboard');
    });
  });

  it('calls checkAuth on mount', () => {
    render(
      <RootLayout>
        <div>Test Content</div>
      </RootLayout>
    );
    
    expect(mockUseAuthStore.checkAuth).toHaveBeenCalled();
  });

  it('handles search functionality', () => {
    const consoleSpy = jest.spyOn(console, 'log').mockImplementation();
    
    render(
      <RootLayout>
        <div>Test Content</div>
      </RootLayout>
    );
    
    const searchButton = screen.getByText('Search');
    searchButton.click();
    
    expect(consoleSpy).toHaveBeenCalledWith('Search query:', 'test');
    
    consoleSpy.mockRestore();
  });

  it('handles new consultation navigation', () => {
    render(
      <RootLayout>
        <div>Test Content</div>
      </RootLayout>
    );
    
    const newConsultationButton = screen.getByText('New Consultation');
    newConsultationButton.click();
    
    expect(mockRouter.push).toHaveBeenCalledWith('/consultation');
  });

  it('applies correct container classes', () => {
    render(
      <RootLayout>
        <div data-testid="content">Test Content</div>
      </RootLayout>
    );
    
    const main = screen.getByRole('main');
    expect(main).toHaveClass('flex-1', 'overflow-y-auto', 'bg-background');
    
    const container = screen.getByTestId('content').parentElement;
    expect(container).toHaveClass('container', 'mx-auto');
  });
});

describe('LayoutErrorBoundary', () => {
  const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
    if (shouldThrow) {
      throw new Error('Test error');
    }
    return <div>No error</div>;
  };

  it('renders children when there is no error', () => {
    render(
      <LayoutErrorBoundary>
        <ThrowError shouldThrow={false} />
      </LayoutErrorBoundary>
    );
    
    expect(screen.getByText('No error')).toBeInTheDocument();
  });

  it('renders error UI when there is an error', () => {
    // Suppress console.error for this test
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(
      <LayoutErrorBoundary>
        <ThrowError shouldThrow={true} />
      </LayoutErrorBoundary>
    );
    
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('An error occurred while loading the application.')).toBeInTheDocument();
    expect(screen.getByText('Reload Page')).toBeInTheDocument();
    
    consoleSpy.mockRestore();
  });

  it('logs error when error occurs', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(
      <LayoutErrorBoundary>
        <ThrowError shouldThrow={true} />
      </LayoutErrorBoundary>
    );
    
    expect(consoleSpy).toHaveBeenCalledWith(
      'Layout Error:',
      expect.any(Error),
      expect.any(Object)
    );
    
    consoleSpy.mockRestore();
  });

  it('renders reload button when error occurs', () => {
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(
      <LayoutErrorBoundary>
        <ThrowError shouldThrow={true} />
      </LayoutErrorBoundary>
    );
    
    const reloadButton = screen.getByText('Reload Page');
    expect(reloadButton).toBeInTheDocument();
    expect(reloadButton.tagName).toBe('BUTTON');
    
    consoleSpy.mockRestore();
  });
});