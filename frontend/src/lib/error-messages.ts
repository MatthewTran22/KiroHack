import { APIError } from './api';

export interface UserFriendlyError {
  title: string;
  message: string;
  action?: string;
  actionLabel?: string;
}

export function getErrorMessage(error: APIError): UserFriendlyError {
  // Network errors
  if (error.isNetworkError) {
    return {
      title: 'Connection Error',
      message: 'Unable to connect to the server. Please check your internet connection and try again.',
      action: 'retry',
      actionLabel: 'Retry',
    };
  }

  // Timeout errors
  if (error.code === 'timeout') {
    return {
      title: 'Request Timeout',
      message: 'The request took too long to complete. Please try again.',
      action: 'retry',
      actionLabel: 'Retry',
    };
  }

  // Authentication errors
  if (error.isAuthError) {
    switch (error.code) {
      case 'invalid_credentials':
        return {
          title: 'Invalid Credentials',
          message: 'The email or password you entered is incorrect. Please try again.',
        };
      case 'account_locked':
        return {
          title: 'Account Locked',
          message: 'Your account has been temporarily locked due to multiple failed login attempts. Please try again later or contact support.',
        };
      case 'mfa_required':
        return {
          title: 'Two-Factor Authentication Required',
          message: 'Please enter your two-factor authentication code to continue.',
        };
      case 'invalid_mfa_code':
        return {
          title: 'Invalid Authentication Code',
          message: 'The two-factor authentication code you entered is incorrect. Please try again.',
        };
      case 'token_expired':
        return {
          title: 'Session Expired',
          message: 'Your session has expired. Please log in again to continue.',
          action: 'login',
          actionLabel: 'Log In',
        };
      case 'insufficient_permissions':
        return {
          title: 'Access Denied',
          message: 'You do not have permission to perform this action.',
        };
      default:
        return {
          title: 'Authentication Error',
          message: 'There was a problem with your authentication. Please log in again.',
          action: 'login',
          actionLabel: 'Log In',
        };
    }
  }

  // Validation errors
  if (error.status === 400) {
    switch (error.code) {
      case 'validation_error':
        return {
          title: 'Invalid Input',
          message: error.details?.message as string || 'Please check your input and try again.',
        };
      case 'file_too_large':
        return {
          title: 'File Too Large',
          message: 'The file you are trying to upload is too large. Please select a smaller file.',
        };
      case 'unsupported_file_type':
        return {
          title: 'Unsupported File Type',
          message: 'The file type you are trying to upload is not supported. Please select a different file.',
        };
      case 'duplicate_resource':
        return {
          title: 'Duplicate Entry',
          message: 'A resource with this information already exists.',
        };
      default:
        return {
          title: 'Invalid Request',
          message: error.message || 'There was a problem with your request. Please check your input and try again.',
        };
    }
  }

  // Not found errors
  if (error.status === 404) {
    return {
      title: 'Not Found',
      message: 'The requested resource could not be found.',
    };
  }

  // Feature-specific errors (check these first before generic status codes)
  switch (error.code) {
    case 'document_processing_failed':
      return {
        title: 'Document Processing Failed',
        message: 'The document could not be processed. Please ensure it is a valid document and try again.',
      };
    case 'consultation_limit_reached':
      return {
        title: 'Consultation Limit Reached',
        message: 'You have reached the maximum number of active consultations. Please complete or archive existing consultations before starting new ones.',
      };
    case 'insufficient_storage':
      return {
        title: 'Storage Limit Exceeded',
        message: 'You have exceeded your storage limit. Please delete some files or contact your administrator.',
      };
    case 'export_failed':
      return {
        title: 'Export Failed',
        message: 'The export could not be completed. Please try again or contact support if the problem persists.',
        action: 'retry',
        actionLabel: 'Retry Export',
      };
    case 'maintenance_mode':
      return {
        title: 'System Maintenance',
        message: 'The system is currently undergoing maintenance. Please try again later.',
      };
    case 'service_unavailable':
      return {
        title: 'Service Unavailable',
        message: 'The service is temporarily unavailable. Please try again in a few minutes.',
        action: 'retry',
        actionLabel: 'Retry',
      };
    case 'unknown_error':
      return {
        title: 'Unexpected Error',
        message: error.message || 'An unexpected error occurred. Please try again or contact support if the problem persists.',
        action: 'retry',
        actionLabel: 'Retry',
      };
  }

  // Rate limiting
  if (error.status === 429) {
    return {
      title: 'Too Many Requests',
      message: 'You have made too many requests. Please wait a moment and try again.',
      action: 'retry',
      actionLabel: 'Try Again',
    };
  }

  // Server errors
  if (error.isServerError) {
    return {
      title: 'Server Error',
      message: 'An unexpected error occurred on the server. Please try again later.',
      action: 'retry',
      actionLabel: 'Retry',
    };
  }

  // Default fallback
  return {
    title: 'Unexpected Error',
    message: error.message || 'An unexpected error occurred. Please try again or contact support if the problem persists.',
    action: 'retry',
    actionLabel: 'Retry',
  };
}

export function getErrorToastConfig(error: APIError) {
  const userError = getErrorMessage(error);
  
  return {
    title: userError.title,
    description: userError.message,
    variant: 'destructive' as const,
    duration: error.isRetryable ? 5000 : 8000,
  };
}

export function shouldShowErrorToast(error: APIError): boolean {
  // Don't show toast for certain errors that are handled elsewhere
  const silentErrors = [
    'token_refreshed',
    'token_expired',
    'insufficient_permissions',
  ];
  
  return !silentErrors.includes(error.code || '');
}