import { APIError } from '../api';
import { getErrorMessage, getErrorToastConfig, shouldShowErrorToast } from '../error-messages';

describe('Error Messages', () => {
  describe('getErrorMessage', () => {
    it('should handle network errors', () => {
      const error = new APIError('Network error', 0);
      const result = getErrorMessage(error);

      expect(result.title).toBe('Connection Error');
      expect(result.message).toContain('Unable to connect to the server');
      expect(result.action).toBe('retry');
      expect(result.actionLabel).toBe('Retry');
    });

    it('should handle timeout errors', () => {
      const error = new APIError('Request timeout', 408, 'timeout');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Request Timeout');
      expect(result.message).toContain('took too long to complete');
      expect(result.action).toBe('retry');
    });

    it('should handle authentication errors', () => {
      const error = new APIError('Invalid credentials', 401, 'invalid_credentials');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Invalid Credentials');
      expect(result.message).toContain('email or password you entered is incorrect');
    });

    it('should handle account locked errors', () => {
      const error = new APIError('Account locked', 401, 'account_locked');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Account Locked');
      expect(result.message).toContain('temporarily locked');
    });

    it('should handle MFA required errors', () => {
      const error = new APIError('MFA required', 401, 'mfa_required');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Two-Factor Authentication Required');
      expect(result.message).toContain('two-factor authentication code');
    });

    it('should handle invalid MFA code errors', () => {
      const error = new APIError('Invalid MFA code', 401, 'invalid_mfa_code');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Invalid Authentication Code');
      expect(result.message).toContain('incorrect');
    });

    it('should handle token expired errors', () => {
      const error = new APIError('Token expired', 401, 'token_expired');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Session Expired');
      expect(result.message).toContain('session has expired');
      expect(result.action).toBe('login');
      expect(result.actionLabel).toBe('Log In');
    });

    it('should handle insufficient permissions errors', () => {
      const error = new APIError('Insufficient permissions', 403, 'insufficient_permissions');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Access Denied');
      expect(result.message).toContain('do not have permission');
    });

    it('should handle validation errors', () => {
      const error = new APIError('Validation failed', 400, 'validation_error', {
        message: 'Email is required',
      });
      const result = getErrorMessage(error);

      expect(result.title).toBe('Invalid Input');
      expect(result.message).toBe('Email is required');
    });

    it('should handle file too large errors', () => {
      const error = new APIError('File too large', 400, 'file_too_large');
      const result = getErrorMessage(error);

      expect(result.title).toBe('File Too Large');
      expect(result.message).toContain('too large');
    });

    it('should handle unsupported file type errors', () => {
      const error = new APIError('Unsupported file type', 400, 'unsupported_file_type');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Unsupported File Type');
      expect(result.message).toContain('not supported');
    });

    it('should handle duplicate resource errors', () => {
      const error = new APIError('Duplicate resource', 400, 'duplicate_resource');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Duplicate Entry');
      expect(result.message).toContain('already exists');
    });

    it('should handle not found errors', () => {
      const error = new APIError('Not found', 404);
      const result = getErrorMessage(error);

      expect(result.title).toBe('Not Found');
      expect(result.message).toContain('could not be found');
    });

    it('should handle rate limiting errors', () => {
      const error = new APIError('Too many requests', 429);
      const result = getErrorMessage(error);

      expect(result.title).toBe('Too Many Requests');
      expect(result.message).toContain('too many requests');
      expect(result.action).toBe('retry');
    });

    it('should handle server errors', () => {
      const error = new APIError('Internal server error', 500);
      const result = getErrorMessage(error);

      expect(result.title).toBe('Server Error');
      expect(result.message).toContain('unexpected error occurred');
      expect(result.action).toBe('retry');
    });

    it('should handle maintenance mode errors', () => {
      const error = new APIError('Maintenance mode', 503, 'maintenance_mode');
      const result = getErrorMessage(error);

      expect(result.title).toBe('System Maintenance');
      expect(result.message).toContain('undergoing maintenance');
    });

    it('should handle service unavailable errors', () => {
      const error = new APIError('Service unavailable', 503, 'service_unavailable');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Service Unavailable');
      expect(result.message).toContain('temporarily unavailable');
      expect(result.action).toBe('retry');
    });

    it('should handle document processing failed errors', () => {
      const error = new APIError('Document processing failed', 422, 'document_processing_failed');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Document Processing Failed');
      expect(result.message).toContain('could not be processed');
    });

    it('should handle consultation limit reached errors', () => {
      const error = new APIError('Consultation limit reached', 429, 'consultation_limit_reached');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Consultation Limit Reached');
      expect(result.message).toContain('maximum number of active consultations');
    });

    it('should handle insufficient storage errors', () => {
      const error = new APIError('Insufficient storage', 413, 'insufficient_storage');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Storage Limit Exceeded');
      expect(result.message).toContain('exceeded your storage limit');
    });

    it('should handle export failed errors', () => {
      const error = new APIError('Export failed', 500, 'export_failed');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Export Failed');
      expect(result.message).toContain('could not be completed');
      expect(result.action).toBe('retry');
      expect(result.actionLabel).toBe('Retry Export');
    });

    it('should handle unknown errors', () => {
      const error = new APIError('Unknown error', 500, 'unknown_error');
      const result = getErrorMessage(error);

      expect(result.title).toBe('Unexpected Error');
      expect(result.message).toBe('Unknown error');
      expect(result.action).toBe('retry');
    });

    it('should handle errors without specific codes', () => {
      const error = new APIError('Generic error', 400);
      const result = getErrorMessage(error);

      expect(result.title).toBe('Invalid Request');
      expect(result.message).toBe('Generic error');
    });
  });

  describe('getErrorToastConfig', () => {
    it('should return correct toast config for retryable errors', () => {
      const error = new APIError('Server error', 500);
      const config = getErrorToastConfig(error);

      expect(config.title).toBe('Server Error');
      expect(config.description).toContain('unexpected error occurred');
      expect(config.variant).toBe('destructive');
      expect(config.duration).toBe(5000); // Retryable errors have shorter duration
    });

    it('should return correct toast config for non-retryable errors', () => {
      const error = new APIError('Invalid input', 400);
      const config = getErrorToastConfig(error);

      expect(config.title).toBe('Invalid Request');
      expect(config.description).toBe('Invalid input');
      expect(config.variant).toBe('destructive');
      expect(config.duration).toBe(8000); // Non-retryable errors have longer duration
    });
  });

  describe('shouldShowErrorToast', () => {
    it('should return true for most errors', () => {
      const error = new APIError('Server error', 500);
      expect(shouldShowErrorToast(error)).toBe(true);
    });

    it('should return false for token refreshed errors', () => {
      const error = new APIError('Token refreshed', 401, 'token_refreshed');
      expect(shouldShowErrorToast(error)).toBe(false);
    });

    it('should return false for token expired errors', () => {
      const error = new APIError('Token expired', 401, 'token_expired');
      expect(shouldShowErrorToast(error)).toBe(false);
    });

    it('should return false for insufficient permissions errors', () => {
      const error = new APIError('Insufficient permissions', 403, 'insufficient_permissions');
      expect(shouldShowErrorToast(error)).toBe(false);
    });

    it('should return true for other auth errors', () => {
      const error = new APIError('Invalid credentials', 401, 'invalid_credentials');
      expect(shouldShowErrorToast(error)).toBe(true);
    });
  });

  describe('APIError properties', () => {
    it('should correctly identify network errors', () => {
      const error = new APIError('Network error', 0);
      expect(error.isNetworkError).toBe(true);
      expect(error.isAuthError).toBe(false);
      expect(error.isServerError).toBe(false);
      expect(error.isClientError).toBe(false);
      expect(error.isRetryable).toBe(true);
    });

    it('should correctly identify auth errors', () => {
      const error = new APIError('Unauthorized', 401);
      expect(error.isNetworkError).toBe(false);
      expect(error.isAuthError).toBe(true);
      expect(error.isServerError).toBe(false);
      expect(error.isClientError).toBe(true);
      expect(error.isRetryable).toBe(false);
    });

    it('should correctly identify server errors', () => {
      const error = new APIError('Internal server error', 500);
      expect(error.isNetworkError).toBe(false);
      expect(error.isAuthError).toBe(false);
      expect(error.isServerError).toBe(true);
      expect(error.isClientError).toBe(false);
      expect(error.isRetryable).toBe(true);
    });

    it('should correctly identify client errors', () => {
      const error = new APIError('Bad request', 400);
      expect(error.isNetworkError).toBe(false);
      expect(error.isAuthError).toBe(false);
      expect(error.isServerError).toBe(false);
      expect(error.isClientError).toBe(true);
      expect(error.isRetryable).toBe(false);
    });

    it('should correctly identify retryable errors', () => {
      const retryableError = new APIError('Too many requests', 429);
      expect(retryableError.isRetryable).toBe(true);

      const nonRetryableError = new APIError('Bad request', 400);
      expect(nonRetryableError.isRetryable).toBe(false);
    });
  });
});