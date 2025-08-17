import { render, screen, waitFor } from '@testing-library/react';
import LogoutPage from '../page';
import { setupAuthMocks, cleanupAuthMocks } from '../../../test/auth-test-utils';
import { useAuth } from '../../../hooks/useAuth';

// Mock the hooks
jest.mock('../../../hooks/useAuth');

// Mock next/navigation
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}));

describe('LogoutPage', () => {
  const mockAuth = {
    logout: jest.fn(),
  };

  beforeEach(() => {
    setupAuthMocks();
    (useAuth as jest.Mock).mockReturnValue(mockAuth);
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  it('should render logout loading state', () => {
    render(<LogoutPage />);

    expect(screen.getByText('Signing out...')).toBeInTheDocument();
    expect(screen.getByText('Please wait while we securely log you out.')).toBeInTheDocument();
  });

  it('should call logout on mount', async () => {
    mockAuth.logout.mockResolvedValueOnce(undefined);

    render(<LogoutPage />);

    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalled();
    });
  });

  it('should redirect to login after successful logout', async () => {
    mockAuth.logout.mockResolvedValueOnce(undefined);

    render(<LogoutPage />);

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/login');
    });
  });

  it('should redirect to login even if logout fails', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    mockAuth.logout.mockRejectedValueOnce(new Error('Logout failed'));

    render(<LogoutPage />);

    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalled();
      expect(mockPush).toHaveBeenCalledWith('/login');
    });

    expect(consoleErrorSpy).toHaveBeenCalledWith('Logout error:', expect.any(Error));
    consoleErrorSpy.mockRestore();
  });

  it('should display loading spinner', () => {
    render(<LogoutPage />);

    const spinner = screen.getByText('Signing out...');
    expect(spinner).toBeInTheDocument();
    
    // Check that the SVG has the animate-spin class
    const svg = document.querySelector('.animate-spin');
    expect(svg).toBeInTheDocument();
  });

  it('should have proper accessibility attributes', () => {
    render(<LogoutPage />);

    // Check for proper heading
    const heading = screen.getByRole('heading', { level: 2 });
    expect(heading).toHaveTextContent('Signing out...');

    // Check for descriptive text
    expect(screen.getByText('Please wait while we securely log you out.')).toBeInTheDocument();
  });

  it('should handle multiple logout calls gracefully', async () => {
    mockAuth.logout.mockResolvedValueOnce(undefined);

    // Render multiple instances
    const { unmount } = render(<LogoutPage />);
    render(<LogoutPage />);

    await waitFor(() => {
      expect(mockAuth.logout).toHaveBeenCalledTimes(2);
      expect(mockPush).toHaveBeenCalledTimes(2);
    });

    unmount();
  });
});