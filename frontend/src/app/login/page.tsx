'use client';

import { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Eye, EyeOff, Shield, AlertCircle } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { useAuth } from '@/hooks/useAuth';
import { ROUTES } from '@/lib/constants';
import { apiClient } from '@/lib/api';
import { tokenManager } from '@/lib/auth';

const loginSchema = z.object({
  email: z.string().email('Please enter a valid email address'),
  password: z.string().min(1, 'Password is required'),
  mfaCode: z.string().optional(),
  rememberMe: z.boolean().default(false),
});

type LoginFormData = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const [showPassword, setShowPassword] = useState(false);
  const [showMFA, setShowMFA] = useState(false);
  const searchParams = useSearchParams();
  const redirectTo = searchParams.get('redirect') || ROUTES.DASHBOARD;

  const { error, isLoading, isLocked, clearError, resetLoginAttempts } = useAuth();
  
  // Note: Removed automatic redirect check to prevent infinite loops
  // Users can manually navigate to dashboard if already authenticated

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
    clearErrors,
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  useEffect(() => {
    if (error) {
      // Clear error after 5 seconds
      const timer = setTimeout(clearError, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, clearError]);

  const onSubmit = async (data: LoginFormData) => {
    try {
      clearError();
      clearErrors();
      
      // Call API directly to get response
      const loginCredentials: any = {
        email: data.email,
        password: data.password
      };
      
      // Only include mfaCode if it's provided
      if (data.mfaCode) {
        loginCredentials.mfaCode = data.mfaCode;
      }
      
      const response = await apiClient.auth.login(loginCredentials);
      
      // Store tokens directly
      tokenManager.setTokens(response.tokens.access_token, response.tokens.refresh_token);
      
      // Instead of trying to update state manually, use window.location for redirect
      // This will cause a full page navigation which will re-evaluate auth state
      window.location.href = redirectTo;
      
    } catch (err: unknown) {
      // Handle login errors
      console.error('Login error:', err);
      if (err && typeof err === 'object' && 'status' in err) {
        const error = err as { status: number; code?: string; message?: string; details?: { field?: string } };
        if (error.status === 401 && error.code === 'MFA_REQUIRED') {
          setShowMFA(true);
        } else if (error.status === 400) {
          // Handle validation errors
          if (error.details?.field) {
            setError(error.details.field as keyof LoginFormData, {
              message: error.message || 'Validation error',
            });
          }
        }
      }
    }
  };

  const handleUnlock = () => {
    resetLoginAttempts();
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div className="text-center">
          <Shield className="mx-auto h-12 w-12 text-blue-600" />
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            AI Government Consultant
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            Sign in to your account
          </p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Sign In</CardTitle>
            <CardDescription>
              Enter your credentials to access the platform
            </CardDescription>
          </CardHeader>
          <CardContent>
            {error && (
              <Alert variant="destructive" className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            {isLocked && (
              <Alert variant="destructive" className="mb-4">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  Account temporarily locked due to multiple failed attempts.
                  <Button
                    variant="link"
                    className="p-0 h-auto ml-1"
                    onClick={handleUnlock}
                  >
                    Click here to unlock
                  </Button>
                </AlertDescription>
              </Alert>
            )}

            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div>
                <Label htmlFor="email">Email Address</Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  disabled={isLoading || isLocked}
                  {...register('email')}
                  className={errors.email ? 'border-red-500' : ''}
                />
                {errors.email && (
                  <p className="mt-1 text-sm text-red-600">{errors.email.message}</p>
                )}
              </div>

              <div>
                <Label htmlFor="password">Password</Label>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    disabled={isLoading || isLocked}
                    {...register('password')}
                    className={errors.password ? 'border-red-500 pr-10' : 'pr-10'}
                  />
                  <button
                    type="button"
                    className="absolute inset-y-0 right-0 pr-3 flex items-center"
                    onClick={() => setShowPassword(!showPassword)}
                    disabled={isLoading || isLocked}
                  >
                    {showPassword ? (
                      <EyeOff className="h-4 w-4 text-gray-400" />
                    ) : (
                      <Eye className="h-4 w-4 text-gray-400" />
                    )}
                  </button>
                </div>
                {errors.password && (
                  <p className="mt-1 text-sm text-red-600">{errors.password.message}</p>
                )}
              </div>

              {showMFA && (
                <div>
                  <Label htmlFor="mfaCode">Two-Factor Authentication Code</Label>
                  <Input
                    id="mfaCode"
                    type="text"
                    placeholder="Enter 6-digit code"
                    maxLength={6}
                    disabled={isLoading || isLocked}
                    {...register('mfaCode')}
                    className={errors.mfaCode ? 'border-red-500' : ''}
                  />
                  {errors.mfaCode && (
                    <p className="mt-1 text-sm text-red-600">{errors.mfaCode.message}</p>
                  )}
                </div>
              )}

              <div className="flex items-center">
                <input
                  id="rememberMe"
                  type="checkbox"
                  disabled={isLoading || isLocked}
                  {...register('rememberMe')}
                  className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                />
                <Label htmlFor="rememberMe" className="ml-2 block text-sm text-gray-900">
                  Remember me
                </Label>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={isLoading || isLocked}
              >
                {isLoading ? 'Signing in...' : 'Sign In'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}