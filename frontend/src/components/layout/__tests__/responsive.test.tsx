import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { useRouter, usePathname } from 'next/navigation';
import { Header } from '../header';
import { Sidebar } from '../sidebar';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import { useTheme } from 'next-themes';

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
  logout: jest.fn(),
};

const mockUseUIStore = {
  toggleSidebar: jest.fn(),
  setSidebarOpen: jest.fn(),
  sidebarOpen: true,
  notifications: [],
};

const mockUseTheme = {
  theme: 'light',
  setTheme: jest.fn(),
};

// Mock window.innerWidth
Object.defineProperty(window, 'innerWidth', {
  writable: true,
  configurable: true,
  value: 1024,
});

describe('Responsive Layout', () => {
  beforeEach(() => {
    (useRouter as jest.Mock).mockReturnValue(mockRouter);
    (usePathname as jest.Mock).mockReturnValue('/dashboard');
    (useTheme as jest.Mock).mockReturnValue(mockUseTheme);
    (useAuthStore as unknown as jest.Mock).mockReturnValue(mockUseAuthStore);
    (useUIStore as unknown as jest.Mock).mockReturnValue(mockUseUIStore);
    jest.clearAllMocks();
  });

  describe('Header Responsive Behavior', () => {
    it('shows mobile menu button on small screens', () => {
      render(<Header />);
      
      const menuButton = screen.getByLabelText('Toggle sidebar');
      expect(menuButton).toHaveClass('lg:hidden');
    });

    it('shows mobile logo and hides desktop logo appropriately', () => {
      render(<Header />);
      
      const desktopLogoContainer = screen.getByText('Gov Consultant').parentElement;
      expect(desktopLogoContainer).toHaveClass('hidden', 'sm:flex');
      
      // Mobile logo should be visible on small screens
      const mobileLogoContainer = document.querySelector('.flex.sm\\:hidden');
      expect(mobileLogoContainer).toBeInTheDocument();
    });

    it('hides action buttons on small screens', () => {
      render(<Header />);
      
      // Check for desktop new chat button by aria-label since text is conditionally hidden
      const desktopNewChatButton = screen.getAllByLabelText('New consultation')[0];
      expect(desktopNewChatButton).toHaveClass('hidden', 'md:flex');
      
      const uploadButton = screen.getByLabelText('Upload document');
      expect(uploadButton).toHaveClass('hidden', 'md:flex');
      
      // Mobile new chat button should be visible
      const mobileNewChatButton = screen.getAllByLabelText('New consultation')[1];
      expect(mobileNewChatButton).toHaveClass('md:hidden');
    });

    it('maintains search bar with responsive sizing', () => {
      render(<Header />);
      
      const searchContainer = screen.getByPlaceholderText('Search...').parentElement?.parentElement;
      expect(searchContainer).toHaveClass('flex-1', 'max-w-xs', 'sm:max-w-md', 'mx-2', 'sm:mx-4');
    });

    it('applies responsive header height', () => {
      render(<Header />);
      
      const headerContainer = screen.getByRole('banner').firstChild;
      expect(headerContainer).toHaveClass('h-14', 'sm:h-16');
    });
  });

  describe('Sidebar Responsive Behavior', () => {
    it('applies mobile-specific classes', () => {
      render(<Sidebar />);
      
      const sidebar = screen.getByRole('complementary');
      expect(sidebar).toHaveClass(
        'fixed',
        'left-0',
        'top-0',
        'z-50',
        'h-full',
        'w-64',
        'sm:w-72',
        'transform',
        'transition-transform',
        'duration-200',
        'ease-in-out',
        'lg:relative',
        'lg:translate-x-0'
      );
    });

    it('shows mobile overlay when sidebar is open', () => {
      render(<Sidebar />);
      
      const overlay = document.querySelector('.fixed.inset-0.z-40.bg-black\\/50.lg\\:hidden');
      expect(overlay).toBeInTheDocument();
    });

    it('hides close button on desktop', () => {
      render(<Sidebar />);
      
      const closeButton = screen.getByLabelText('Close sidebar');
      expect(closeButton).toHaveClass('lg:hidden');
    });

    it('applies responsive header height', () => {
      render(<Sidebar />);
      
      const sidebarHeader = screen.getByText('Gov Consultant').closest('div')?.parentElement;
      expect(sidebarHeader).toHaveClass('h-14', 'sm:h-16');
    });

    it('closes sidebar on mobile after navigation', () => {
      // Mock mobile screen size
      Object.defineProperty(window, 'innerWidth', {
        value: 1023,
        writable: true,
      });
      
      render(<Sidebar />);
      
      const dashboardLink = screen.getByText('Dashboard');
      fireEvent.click(dashboardLink);
      
      expect(mockUseUIStore.setSidebarOpen).toHaveBeenCalledWith(false);
    });

    it('does not close sidebar on desktop after navigation', () => {
      // Mock desktop screen size
      Object.defineProperty(window, 'innerWidth', {
        value: 1024,
        writable: true,
      });
      
      render(<Sidebar />);
      
      const dashboardLink = screen.getByText('Dashboard');
      fireEvent.click(dashboardLink);
      
      expect(mockUseUIStore.setSidebarOpen).not.toHaveBeenCalled();
    });
  });

  describe('Breakpoint Behavior', () => {
    it('applies correct responsive classes for different breakpoints', () => {
      render(<Header />);
      
      // Check that responsive classes are applied
      const container = screen.getByRole('banner').firstChild;
      expect(container).toHaveClass('container', 'flex', 'h-14', 'sm:h-16', 'items-center', 'justify-between', 'px-3', 'sm:px-4');
    });

    it('handles tablet breakpoint correctly', () => {
      render(<Sidebar />);
      
      const sidebar = screen.getByRole('complementary');
      expect(sidebar).toHaveClass('lg:relative', 'lg:translate-x-0');
    });

    it('applies mobile-first approach', () => {
      render(<Header />);
      
      // Mobile-first: hidden by default, shown on larger screens
      const logoContainer = screen.getByText('Gov Consultant').parentElement;
      expect(logoContainer).toHaveClass('hidden', 'sm:flex');
      
      // Mobile-first: visible by default, hidden on larger screens
      const menuButton = screen.getByLabelText('Toggle sidebar');
      expect(menuButton).toHaveClass('lg:hidden');
    });
  });

  describe('Touch-Friendly Interfaces', () => {
    it('applies appropriate sizing for touch targets', () => {
      render(<Header />);
      
      // Buttons should have adequate touch target size
      const themeButton = screen.getByLabelText('Toggle theme');
      expect(themeButton).toBeInTheDocument(); // Button exists
      
      const notificationButton = screen.getByLabelText(/Notifications/);
      expect(notificationButton).toHaveClass('relative'); // Button container
    });

    it('provides adequate spacing for mobile interactions', () => {
      render(<Sidebar />);
      
      // Navigation items should have adequate padding
      const dashboardLink = screen.getByText('Dashboard').closest('a');
      expect(dashboardLink).toHaveClass('px-3', 'py-2.5');
    });
  });

  describe('Accessibility on Mobile', () => {
    it('maintains proper focus management on mobile', () => {
      render(<Sidebar />);
      
      const closeButton = screen.getByLabelText('Close sidebar');
      expect(closeButton).toBeInTheDocument();
      expect(closeButton.getAttribute('aria-label')).toBe('Close sidebar');
    });

    it('provides proper ARIA labels for mobile interactions', () => {
      render(<Header />);
      
      const menuButton = screen.getByLabelText('Toggle sidebar');
      expect(menuButton.getAttribute('aria-label')).toBe('Toggle sidebar');
    });

    it('maintains semantic structure on all screen sizes', () => {
      render(<Header />);
      
      const header = screen.getByRole('banner');
      expect(header).toBeInTheDocument();
      
      render(<Sidebar />);
      
      const sidebar = screen.getByRole('complementary');
      expect(sidebar).toBeInTheDocument();
    });
  });

  describe('Performance Considerations', () => {
    it('uses CSS transforms for smooth animations', () => {
      render(<Sidebar />);
      
      const sidebar = screen.getByRole('complementary');
      expect(sidebar).toHaveClass('transform', 'transition-transform', 'duration-200', 'ease-in-out');
    });

    it('applies backdrop-blur for performance', () => {
      render(<Header />);
      
      const header = screen.getByRole('banner');
      expect(header).toHaveClass('backdrop-blur', 'supports-[backdrop-filter]:bg-background/60');
    });
  });
});