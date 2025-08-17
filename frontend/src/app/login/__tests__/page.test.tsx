import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginPage from '../page';
import { setupAuthMocks, cleanupAuthMocks, mockAPIResponses } from '../../../test/auth-test-utils';
import { useAuth, useRedirectIfAuthenticated } from '../../../hooks/useAuth';

// Mock the hooks
jest.mock('../../../hooks/useAuth');

// Mock next/navigation
const mockPush = jest.fn();
const mockGet = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
  useSearchParams: () => ({
    get: mockGet,
  }),
}));

describe('LoginPage', () => {
  const mockAuth = {
    login: jest.fn(),
    error: null,
    isLoading: false,
    isLocked: false,
    clearError: jest.fn(),
    resetLoginAttempts: jest.fn(),
  };

  beforeEach(() => {
    setupAuthMocks();
    (useAuth as jest.Mock).mockReturnValue(mockAuth);
    (useRedirectIfAuthenticated as jest.Mock).mockReturnValue({
      isAuthenticated: false,
      isLoading: false,
    });
    mockGet.mockReturnValue(null);
    jest.clearAllMocks();
  });

  afterEach(() => {
    cleanupAuthMocks();
  });

  it('should render login form', () => {
    render(<LoginPage />);

    expect(screen.getByText('AI Government Consultant')).toBeInTheDocument();
    expect(screen.getByText('Sign in to your account')).toBeInTheDocument();
    expect(screen.getByLabelText('Email Address')).toBeInTheDocument();
    expect(screen.getByLabelText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument();
  });

  it('should handle form submission with valid credentials', async () => {
    const user = userEvent.setup();
    mockAuth.login.mockResolvedValueOnce(undefined);

    render(<LoginPage />);

    await user.type(screen.getByLabelText('Email Address'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(mockAuth.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
        rememberMe: false,
      });
    });

    expect(mockPush).toHaveBeenCalledWith('/dashboard');
  });

  it('should redirect to specified redirect URL after login', async () => {
    const user = userEvent.setup();
    mockAuth.login.mockResolvedValueOnce(undefined);
    mockGet.mockReturnValue('/documents');

    render(<LoginPage />);

    await user.type(screen.getByLabelText('Email Address'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith('/documents');
    });
  });

  it('should show MFA input when MFA is required', async () => {
    const user = userEvent.setup();
    const mfaError = {
      status: 401,
      code: 'MFA_REQUIRED',
      message: 'MFA required',
    };
    mockAuth.login.mockRejectedValueOnce(mfaError);

    render(<LoginPage />);

    await user.type(screen.getByLabelText('Email Address'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(screen.getByLabelText('Two-Factor Authentication Code')).toBeInTheDocument();
    });
  });

  it('should handle MFA code submission', async () => {
    const user = userEvent.setup();
    
    // First login attempt triggers MFA
    const mfaError = {
      status: 401,
      code: 'MFA_REQUIRED',
      message: 'MFA required',
    };
    mockAuth.login.mockRejectedValueOnce(mfaError);

    render(<LoginPage />);

    await user.type(screen.getByLabelText('Email Address'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(screen.getByLabelText('Two-Factor Authentication Code')).toBeInTheDocument();
    });

    // Second login attempt with MFA code
    mockAuth.login.mockResolvedValueOnce(undefined);
    
    await user.type(screen.getByLabelText('Two-Factor Authentication Code'), '123456');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(mockAuth.login).toHaveBeenLastCalledWith({
        email: 'test@example.com',
        password: 'password123',
        mfaCode: '123456',
        rememberMe: false,
      });
    });
  });

  it('should toggle password visibility', async () => {
    const user = userEvent.setup();
    render(<LoginPage />);

    const passwordInput = screen.getByLabelText('Password');
    const toggleButton = screen.getByRole('button', { name: '' }); // Eye icon button

    expect(passwordInput).toHaveAttribute('type', 'password');

    await user.click(toggleButton);
    expect(passwordInput).toHaveAttribute('type', 'text');

    await user.click(toggleButton);
    expect(passwordInput).toHaveAttribute('type', 'password');
  });

  it('should handle remember me checkbox', async () => {
    const user = userEvent.setup();
    mockAuth.login.mockResolvedValueOnce(undefined);

    render(<LoginPage />);

    await user.type(screen.getByLabelText('Email Address'), 'test@example.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByLabelText('Remember me'));
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(mockAuth.login).toHaveBeenCalledWith({
        email: 'test@example.com',
        password: 'password123',
        rememberMe: true,
      });
    });
  });

  it('should display error message', () => {
    const errorAuth = {
      ...mockAuth,
      error: 'Invalid credentials',
    };
    (useAuth as jest.Mock).mockReturnValue(errorAuth);

    render(<LoginPage />);

    expect(screen.getByText('Invalid credentials')).toBeInTheDocument();
  });

  it('should display locked account message', () => {
    const lockedAuth = {
      ...mockAuth,
      isLocked: true,
    };
    (useAuth as jest.Mock).mockReturnValue(lockedAuth);

    render(<LoginPage />);

    expect(screen.getByText(/Account temporarily locked/)).toBeInTheDocument();
    expect(screen.getByText('Click here to unlock')).toBeInTheDocument();
  });

  it('should handle unlock account', async () => {
    const user = userEvent.setup();
    const lockedAuth = {
      ...mockAuth,
      isLocked: true,
    };
    (useAuth as jest.Mock).mockReturnValue(lockedAuth);

    render(<LoginPage />);

    await user.click(screen.getByText('Click here to unlock'));

    expect(mockAuth.resetLoginAttempts).toHaveBeenCalled();
  });

  it('should disable form when loading', () => {
    const loadingAuth = {
      ...mockAuth,
      isLoading: true,
    };
    (useAuth as jest.Mock).mockReturnValue(loadingAuth);

    render(<LoginPage />);

    expect(screen.getByLabelText('Email Address')).toBeDisabled();
    expect(screen.getByLabelText('Password')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Signing in...' })).toBeDisabled();
  });

  it('should disable form when account is locked', () => {
    const lockedAuth = {
      ...mockAuth,
      isLocked: true,
    };
    (useAuth as jest.Mock).mockReturnValue(lockedAuth);

    render(<LoginPage />);

    expect(screen.getByLabelText('Email Address')).toBeDisabled();
    expect(screen.getByLabelText('Password')).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeDisabled();
  });

  it('should validate email format', async () => {
    const user = userEvent.setup();
    render(<LoginPage />);

    const emailInput = screen.getByLabelText('Email Address');
    const passwordInput = screen.getByLabelText('Password');
    const submitButton = screen.getByRole('button', { name: 'Sign In' });

    await user.type(emailInput, 'invalid-email');
    await user.type(passwordInput, 'password123');
    await user.click(submitButton);

    // Wait a bit for any validation to occur
    await new Promise(resolve => setTimeout(resolve, 100));

    // The main expectation is that login should not be called with invalid email
    expect(mockAuth.login).not.toHaveBeenCalled();
  });

  it('should validate required password', async () => {
    const user = userEvent.setup();
    render(<LoginPage />);

    const emailInput = screen.getByLabelText('Email Address');
    const submitButton = screen.getByRole('button', { name: 'Sign In' });

    await user.type(emailInput, 'test@example.com');
    await user.click(submitButton);

    // Wait a bit for any validation to occur
    await new Promise(resolve => setTimeout(resolve, 100));

    // The main expectation is that login should not be called without password
    expect(mockAuth.login).not.toHaveBeenCalled();
  });

  it('should clear error after timeout', async () => {
    jest.useFakeTimers();
    
    const errorAuth = {
      ...mockAuth,
      error: 'Some error',
    };
    (useAuth as jest.Mock).mockReturnValue(errorAuth);

    render(<LoginPage />);

    expect(screen.getByText('Some error')).toBeInTheDocument();

    // Fast-forward time
    jest.advanceTimersByTime(5000);

    expect(mockAuth.clearError).toHaveBeenCalled();

    jest.useRealTimers();
  });
});