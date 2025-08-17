import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { usePathname } from 'next/navigation';
import { Sidebar } from '../sidebar';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';

// Mock dependencies
jest.mock('next/navigation');
jest.mock('@/stores/auth');
jest.mock('@/stores/ui');

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
};

const mockUseUIStore = {
  sidebarOpen: true,
  setSidebarOpen: jest.fn(),
};

describe('Sidebar', () => {
  beforeEach(() => {
    (usePathname as jest.Mock).mockReturnValue('/dashboard');
    (useAuthStore as unknown as jest.Mock).mockReturnValue(mockUseAuthStore);
    (useUIStore as unknown as jest.Mock).mockReturnValue(mockUseUIStore);
    jest.clearAllMocks();
  });

  it('renders sidebar with navigation items', () => {
    render(<Sidebar />);
    
    expect(screen.getByText('Gov Consultant')).toBeInTheDocument();
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('New Consultation')).toBeInTheDocument();
    expect(screen.getByText('Documents')).toBeInTheDocument();
    expect(screen.getByText('History')).toBeInTheDocument();
    expect(screen.getByText('Settings')).toBeInTheDocument();
  });

  it('shows admin-only navigation items for admin users', () => {
    render(<Sidebar />);
    
    expect(screen.getByText('Analytics')).toBeInTheDocument();
    expect(screen.getByText('User Management')).toBeInTheDocument();
    expect(screen.getByText('Audit Trail')).toBeInTheDocument();
  });

  it('hides admin-only navigation items for regular users', () => {
    const regularUser = { ...mockUser, role: 'user' as const };
    (useAuthStore as unknown as jest.Mock).mockReturnValue({
      user: regularUser,
    });
    
    render(<Sidebar />);
    
    expect(screen.queryByText('Analytics')).not.toBeInTheDocument();
    expect(screen.queryByText('User Management')).not.toBeInTheDocument();
    expect(screen.queryByText('Audit Trail')).not.toBeInTheDocument();
  });

  it('highlights active navigation item', () => {
    (usePathname as jest.Mock).mockReturnValue('/documents');
    
    render(<Sidebar />);
    
    const documentsLink = screen.getByText('Documents').closest('a');
    expect(documentsLink).toHaveClass('bg-accent', 'text-accent-foreground');
  });

  it('shows document badge', () => {
    render(<Sidebar />);
    
    expect(screen.getByText('12')).toBeInTheDocument();
  });

  it('displays user information in footer', () => {
    render(<Sidebar />);
    
    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText(/admin/)).toBeInTheDocument();
    expect(screen.getByText('J')).toBeInTheDocument(); // User initial
  });

  it('closes sidebar when close button is clicked', () => {
    render(<Sidebar />);
    
    const closeButton = screen.getByLabelText('Close sidebar');
    fireEvent.click(closeButton);
    
    expect(mockUseUIStore.setSidebarOpen).toHaveBeenCalledWith(false);
  });

  it('applies correct transform classes based on sidebar state', () => {
    const { rerender } = render(<Sidebar />);
    
    // Sidebar open
    let sidebar = document.querySelector('aside');
    expect(sidebar).toHaveClass('translate-x-0');
    
    // Sidebar closed
    (useUIStore as unknown as jest.Mock).mockReturnValue({
      ...mockUseUIStore,
      sidebarOpen: false,
    });
    
    rerender(<Sidebar />);
    sidebar = document.querySelector('aside');
    expect(sidebar).toHaveClass('-translate-x-full');
  });

  it('renders mobile overlay when sidebar is open', () => {
    render(<Sidebar />);
    
    const overlay = document.querySelector('.fixed.inset-0.z-40.bg-black\\/50');
    expect(overlay).toBeInTheDocument();
  });

  it('does not render mobile overlay when sidebar is closed', () => {
    (useUIStore as unknown as jest.Mock).mockReturnValue({
      ...mockUseUIStore,
      sidebarOpen: false,
    });
    
    render(<Sidebar />);
    
    const overlay = document.querySelector('.fixed.inset-0.z-40.bg-black\\/50');
    expect(overlay).not.toBeInTheDocument();
  });

  it('closes sidebar when overlay is clicked', () => {
    render(<Sidebar />);
    
    const overlay = document.querySelector('.fixed.inset-0.z-40.bg-black\\/50');
    fireEvent.click(overlay!);
    
    expect(mockUseUIStore.setSidebarOpen).toHaveBeenCalledWith(false);
  });

  it('has proper accessibility attributes', () => {
    render(<Sidebar />);
    
    const sidebar = document.querySelector('aside');
    expect(sidebar).toBeInTheDocument();
    
    const closeButton = screen.getByLabelText('Close sidebar');
    expect(closeButton).toBeInTheDocument();
  });

  it('handles dashboard route correctly', () => {
    (usePathname as jest.Mock).mockReturnValue('/');
    
    render(<Sidebar />);
    
    const dashboardLink = screen.getByText('Dashboard').closest('a');
    expect(dashboardLink).toHaveClass('bg-accent', 'text-accent-foreground');
  });

  it('handles nested routes correctly', () => {
    (usePathname as jest.Mock).mockReturnValue('/documents/upload');
    
    render(<Sidebar />);
    
    const documentsLink = screen.getByText('Documents').closest('a');
    expect(documentsLink).toHaveClass('bg-accent', 'text-accent-foreground');
  });
});