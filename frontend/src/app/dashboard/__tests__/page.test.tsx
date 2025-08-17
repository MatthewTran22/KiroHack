import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DashboardPage from '../page';
import { setupAuthMocks, cleanupAuthMocks, mockUser, mockAdminUser } from '../../../test/auth-test-utils';
import { useAuth, useRequireAuth } from '../../../hooks/useAuth';

// Mock the hooks
jest.mock('../../../hooks/useAuth');

// Mock next/navigation
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

describe('DashboardPage', () => {
  const mockAuth = {
    user: mockUser,
    logout: jest.fn(),
  };

  const mockRequireAuth = {
    isLoading: false,
  };

  beforeEach(() => {
    setupAuthMocks();
    (useAuth as jest.Mock).mockReturnValue(mockAuth);
    (useRequireAuth as jest.Mock).mockReturnValue(mockRequireAuth);
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  it('should render dashboard with user information', () => {
    render(<DashboardPage />);

    expect(screen.getByText('AI Government Consultant Dashboard')).toBeInTheDocument();
    expect(screen.getByText(`Welcome back, ${mockUser.name}!`)).toBeInTheDocument();
    expect(screen.getByText('Profile Information')).toBeInTheDocument();
    expect(screen.getByText(mockUser.email)).toBeInTheDocument();
    expect(screen.getByText(mockUser.role)).toBeInTheDocument();
  });

  it('should show loading state when authentication is loading', () => {
    (useRequireAuth as jest.Mock).mockReturnValue({ isLoading: true });

    render(<DashboardPage />);

    // Check that the loading spinner is present
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
    expect(screen.queryByText('AI Government Consultant Dashboard')).not.toBeInTheDocument();
  });

  it('should not render when user is null', () => {
    (useAuth as jest.Mock).mockReturnValue({
      ...mockAuth,
      user: null,
    });

    const { container } = render(<DashboardPage />);

    expect(container.firstChild).toBeNull();
  });

  it('should display user department when available', () => {
    const userWithDepartment = {
      ...mockUser,
      department: 'IT Department',
    };

    (useAuth as jest.Mock).mockReturnValue({
      ...mockAuth,
      user: userWithDepartment,
    });

    render(<DashboardPage />);

    expect(screen.getByText('IT Department')).toBeInTheDocument();
  });

  it('should show MFA status correctly', () => {
    render(<DashboardPage />);

    expect(screen.getByText('Two-Factor Authentication')).toBeInTheDocument();
    expect(screen.getByText('Disabled')).toBeInTheDocument();
    expect(screen.getByText('Set Up MFA')).toBeInTheDocument();
  });

  it('should show MFA as enabled for users with MFA', () => {
    const userWithMFA = {
      ...mockUser,
      mfaEnabled: true,
    };

    (useAuth as jest.Mock).mockReturnValue({
      ...mockAuth,
      user: userWithMFA,
    });

    render(<DashboardPage />);

    expect(screen.getByText('Enabled')).toBeInTheDocument();
    expect(screen.queryByText('Set Up MFA')).not.toBeInTheDocument();
  });

  it('should handle MFA setup navigation', async () => {
    const user = userEvent.setup();
    render(<DashboardPage />);

    const mfaButton = screen.getByText('Set Up MFA');
    await user.click(mfaButton);

    expect(mockPush).toHaveBeenCalledWith('/mfa-setup');
  });

  it('should handle logout', async () => {
    const user = userEvent.setup();
    mockAuth.logout.mockResolvedValueOnce(undefined);

    render(<DashboardPage />);

    const logoutButton = screen.getByText('Sign Out');
    await user.click(logoutButton);

    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalled();
      expect(mockPush).toHaveBeenCalledWith('/login');
    });
  });

  it('should display quick action buttons', () => {
    render(<DashboardPage />);

    expect(screen.getByText('New Consultation')).toBeInTheDocument();
    expect(screen.getByText('Upload Documents')).toBeInTheDocument();
    expect(screen.getByText('View History')).toBeInTheDocument();
  });

  it('should display welcome message and getting started section', () => {
    render(<DashboardPage />);

    expect(screen.getByText('Getting Started')).toBeInTheDocument();
    expect(screen.getByText('Welcome to the AI Government Consultant platform')).toBeInTheDocument();
    expect(screen.getByText('Start New Consultation')).toBeInTheDocument();
    expect(screen.getByText('View Documentation')).toBeInTheDocument();
  });

  it('should display correct role badge styling', () => {
    render(<DashboardPage />);

    const roleText = screen.getByText(mockUser.role);
    expect(roleText).toBeInTheDocument();
  });

  it('should handle admin user correctly', () => {
    (useAuth as jest.Mock).mockReturnValue({
      ...mockAuth,
      user: mockAdminUser,
    });

    render(<DashboardPage />);

    expect(screen.getByText(mockAdminUser.name)).toBeInTheDocument();
    expect(screen.getByText(mockAdminUser.role)).toBeInTheDocument();
    expect(screen.getByText('Enabled')).toBeInTheDocument(); // MFA enabled for admin
  });

  it('should handle logout error gracefully', async () => {
    const user = userEvent.setup();
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    mockAuth.logout.mockRejectedValueOnce(new Error('Logout failed'));

    render(<DashboardPage />);

    const logoutButton = screen.getByText('Sign Out');
    await user.click(logoutButton);

    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalled();
      expect(mockPush).toHaveBeenCalledWith('/login');
    });

    expect(consoleErrorSpy).toHaveBeenCalledWith('Logout error:', expect.any(Error));
    consoleErrorSpy.mockRestore();
  });

  it('should have proper accessibility attributes', () => {
    render(<DashboardPage />);

    // Check for proper heading structure
    const mainHeading = screen.getByRole('heading', { level: 1 });
    expect(mainHeading).toHaveTextContent('AI Government Consultant Dashboard');

    // Check for buttons
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);

    // Each button should have accessible text
    buttons.forEach(button => {
      expect(button).toHaveAccessibleName();
    });
  });

  it('should display security status correctly', () => {
    render(<DashboardPage />);

    expect(screen.getByText('Security Settings')).toBeInTheDocument();
    expect(screen.getByText('Manage your account security')).toBeInTheDocument();
  });
});