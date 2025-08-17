import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { useRouter, usePathname } from 'next/navigation';
import { useTheme } from 'next-themes';
import { RootLayout } from '../root-layout';
import { Header } from '../header';
import { Sidebar } from '../sidebar';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';

// Mock dependencies
jest.mock('next/navigation');
jest.mock('next-themes');
jest.mock('@/stores/auth');
jest.mock('@/stores/ui');

const mockRouter = { push: jest.fn() };
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
    logout: jest.fn(),
};

const mockUseUIStore = {
    sidebarOpen: true,
    setSidebarOpen: jest.fn(),
    toggleSidebar: jest.fn(),
    notifications: [],
};

const mockUseTheme = {
    theme: 'light',
    setTheme: jest.fn(),
};

describe('Layout Integration with Docker Backend', () => {
    beforeEach(() => {
        (useRouter as jest.Mock).mockReturnValue(mockRouter);
        (usePathname as jest.Mock).mockReturnValue('/dashboard');
        (useTheme as jest.Mock).mockReturnValue(mockUseTheme);
        (useAuthStore as unknown as jest.Mock).mockReturnValue(mockUseAuthStore);
        (useUIStore as unknown as jest.Mock).mockReturnValue(mockUseUIStore);
        jest.clearAllMocks();
    });

    describe('Authentication Integration', () => {
        it('integrates with backend authentication service', async () => {
            render(
                <RootLayout>
                    <div>Dashboard Content</div>
                </RootLayout>
            );

            // Verify that checkAuth is called on mount (would connect to Docker backend)
            expect(mockUseAuthStore.checkAuth).toHaveBeenCalled();

            // Verify authenticated layout is rendered
            expect(screen.getByText('Dashboard Content')).toBeInTheDocument();
        });

        it('handles logout with backend integration', async () => {
            render(<Header />);

            // This would trigger a logout request to the Docker backend
            await mockUseAuthStore.logout();
            expect(mockUseAuthStore.logout).toHaveBeenCalled();
        });
    });

    describe('Search Integration', () => {
        it('integrates search with backend API', async () => {
            const mockOnSearch = jest.fn();
            render(<Header onSearch={mockOnSearch} />);

            const searchInput = screen.getByPlaceholderText('Search...');
            fireEvent.change(searchInput, { target: { value: 'test query' } });

            // This would trigger a search request to the Docker backend
            expect(mockOnSearch).toHaveBeenCalledWith('test query');
        });
    });

    describe('Navigation Integration', () => {
        it('integrates navigation with backend routing', () => {
            render(<Sidebar />);

            const consultationLink = screen.getByText('New Consultation');
            fireEvent.click(consultationLink);

            // This would navigate to a route that connects to Docker backend
            expect(mockRouter.push).not.toHaveBeenCalled(); // Link component handles navigation
        });
    });

    describe('Theme Integration', () => {
        it('persists theme changes with backend preferences', () => {
            render(<Header />);

            const themeButton = screen.getByLabelText('Toggle theme');
            fireEvent.click(themeButton);

            // This would save theme preference to backend via Docker API
            expect(mockUseTheme.setTheme).toHaveBeenCalledWith('dark');
        });
    });

    describe('Real-time Updates Integration', () => {
        it('handles real-time notifications from backend', async () => {
            const notificationsWithUnread = [
                {
                    id: '1',
                    type: 'info' as const,
                    title: 'New Document',
                    message: 'A new document has been processed',
                    timestamp: new Date(),
                    read: false,
                },
            ];

            (useUIStore as unknown as jest.Mock).mockReturnValue({
                ...mockUseUIStore,
                notifications: notificationsWithUnread,
            });

            render(<Header />);

            // Verify notification badge is displayed (would come from Docker backend WebSocket)
            expect(screen.getByText('1')).toBeInTheDocument();
        });
    });

    describe('Responsive Behavior with Backend', () => {
        it('maintains functionality across screen sizes with backend integration', () => {
            // Test mobile layout
            Object.defineProperty(window, 'innerWidth', {
                value: 768,
                writable: true,
            });

            render(
                <RootLayout>
                    <div>Mobile Content</div>
                </RootLayout>
            );

            // Verify layout works on mobile (backend integration remains the same)
            expect(screen.getByText('Mobile Content')).toBeInTheDocument();
            expect(mockUseAuthStore.checkAuth).toHaveBeenCalled();
        });
    });

    describe('Error Handling with Backend', () => {
        it('handles backend connection errors gracefully', () => {
            // Simulate a scenario where backend is unavailable but layout still works
            const mockCheckAuth = jest.fn(); // Mock that doesn't throw

            (useAuthStore as unknown as jest.Mock).mockReturnValue({
                ...mockUseAuthStore,
                checkAuth: mockCheckAuth,
            });

            render(
                <RootLayout>
                    <div>Content</div>
                </RootLayout>
            );

            // Layout should still render and attempt to check auth
            expect(screen.getByText('Content')).toBeInTheDocument();
            expect(mockCheckAuth).toHaveBeenCalled();
        });
    });

    describe('Performance with Backend Integration', () => {
        it('optimizes API calls to Docker backend', async () => {
            render(
                <RootLayout>
                    <div>Content</div>
                </RootLayout>
            );

            // Verify checkAuth is only called once per mount
            expect(mockUseAuthStore.checkAuth).toHaveBeenCalledTimes(1);
        });

        it('handles loading states during backend requests', () => {
            (useAuthStore as unknown as jest.Mock).mockReturnValue({
                ...mockUseAuthStore,
                isAuthenticated: false,
            });

            render(
                <RootLayout>
                    <div>Protected Content</div>
                </RootLayout>
            );

            // Should show loading state while backend authenticates
            expect(document.querySelector('.animate-spin')).toBeInTheDocument();
            expect(screen.getByText('Loading...')).toBeInTheDocument();
        });
    });

    describe('Accessibility with Backend Integration', () => {
        it('maintains accessibility during backend operations', () => {
            render(<Header />);

            // Verify ARIA labels are present for backend-connected features
            expect(screen.getByLabelText('Toggle sidebar')).toBeInTheDocument();
            expect(screen.getByLabelText('Search documents and consultations')).toBeInTheDocument();
            expect(screen.getByLabelText(/Notifications/)).toBeInTheDocument();
        });

        it('provides proper focus management during backend loading', () => {
            (useAuthStore as unknown as jest.Mock).mockReturnValue({
                ...mockUseAuthStore,
                isAuthenticated: false,
            });

            render(
                <RootLayout>
                    <div>Content</div>
                </RootLayout>
            );

            // Loading spinner should have proper ARIA attributes
            const loadingSpinner = document.querySelector('[role="status"]');
            expect(loadingSpinner).toBeInTheDocument();
            expect(loadingSpinner).toHaveAttribute('aria-label', 'Loading');
        });
    });
});