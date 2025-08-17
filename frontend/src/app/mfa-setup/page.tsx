'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Shield, Copy, Check, AlertCircle, QrCode } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { useAuth, useRequireAuth } from '@/hooks/useAuth';
import { apiClient } from '@/lib/api';
import { ROUTES } from '@/lib/constants';

const mfaSchema = z.object({
  code: z.string().length(6, 'Code must be 6 digits').regex(/^\d+$/, 'Code must contain only numbers'),
});

type MFAFormData = z.infer<typeof mfaSchema>;

interface MFASetup {
  qrCode: string;
  secret: string;
  backupCodes: string[];
}

export default function MFASetupPage() {
  const [mfaSetup, setMfaSetup] = useState<MFASetup | null>(null);
  const [isVerified, setIsVerified] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copiedSecret, setCopiedSecret] = useState(false);
  const [copiedBackupCodes, setCopiedBackupCodes] = useState(false);

  const router = useRouter();
  const { user } = useAuth();
  
  // Require authentication
  useRequireAuth();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<MFAFormData>({
    resolver: zodResolver(mfaSchema),
  });

  useEffect(() => {
    const setupMFA = async () => {
      try {
        setIsLoading(true);
        const setup = await apiClient.setupMFA();
        setMfaSetup(setup);
      } catch (err: unknown) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to setup MFA';
        setError(errorMessage);
      } finally {
        setIsLoading(false);
      }
    };

    if (user && !user.mfaEnabled) {
      setupMFA();
    } else if (user?.mfaEnabled) {
      router.push(ROUTES.DASHBOARD);
    }
  }, [user, router]);

  const onSubmit = async (data: MFAFormData) => {
    try {
      setIsLoading(true);
      setError(null);
      
      const result = await apiClient.verifyMFA(data.code);
      
      if (result.success) {
        setIsVerified(true);
        reset();
      } else {
        setError('Invalid verification code. Please try again.');
      }
    } catch (err: unknown) {
      const errorMessage = err instanceof Error ? err.message : 'Verification failed';
      setError(errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = async (text: string, type: 'secret' | 'backup') => {
    try {
      await navigator.clipboard.writeText(text);
      if (type === 'secret') {
        setCopiedSecret(true);
        setTimeout(() => setCopiedSecret(false), 2000);
      } else {
        setCopiedBackupCodes(true);
        setTimeout(() => setCopiedBackupCodes(false), 2000);
      }
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
    }
  };

  const handleComplete = () => {
    router.push(ROUTES.DASHBOARD);
  };

  if (!user) {
    return null; // Will redirect via useRequireAuth
  }

  if (user.mfaEnabled) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4">
        <Card className="max-w-md w-full">
          <CardHeader className="text-center">
            <Shield className="mx-auto h-12 w-12 text-green-600" />
            <CardTitle>MFA Already Enabled</CardTitle>
            <CardDescription>
              Two-factor authentication is already set up for your account.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => router.push(ROUTES.DASHBOARD)} className="w-full">
              Go to Dashboard
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4">
      <div className="max-w-2xl w-full space-y-8">
        <div className="text-center">
          <Shield className="mx-auto h-12 w-12 text-blue-600" />
          <h2 className="mt-6 text-3xl font-extrabold text-gray-900">
            Set Up Two-Factor Authentication
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            Enhance your account security with MFA
          </p>
        </div>

        {!isVerified ? (
          <div className="grid gap-6 md:grid-cols-2">
            {/* QR Code Section */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <QrCode className="h-5 w-5" />
                  Scan QR Code
                </CardTitle>
                <CardDescription>
                  Use your authenticator app to scan this QR code
                </CardDescription>
              </CardHeader>
              <CardContent>
                {isLoading ? (
                  <div className="flex items-center justify-center h-48">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                  </div>
                ) : mfaSetup ? (
                  <div className="space-y-4">
                    <div className="flex justify-center">
                      <Image
                        src={mfaSetup.qrCode}
                        alt="MFA QR Code"
                        width={256}
                        height={256}
                        className="border rounded-lg"
                      />
                    </div>
                    <div>
                      <Label>Manual Entry Key</Label>
                      <div className="flex items-center gap-2 mt-1">
                        <Input
                          value={mfaSetup.secret}
                          readOnly
                          className="font-mono text-sm"
                        />
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => copyToClipboard(mfaSetup.secret, 'secret')}
                        >
                          {copiedSecret ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                        </Button>
                      </div>
                    </div>
                  </div>
                ) : null}
              </CardContent>
            </Card>

            {/* Verification Section */}
            <Card>
              <CardHeader>
                <CardTitle>Verify Setup</CardTitle>
                <CardDescription>
                  Enter the 6-digit code from your authenticator app
                </CardDescription>
              </CardHeader>
              <CardContent>
                {error && (
                  <Alert variant="destructive" className="mb-4">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
                  <div>
                    <Label htmlFor="code">Verification Code</Label>
                    <Input
                      id="code"
                      type="text"
                      placeholder="000000"
                      maxLength={6}
                      disabled={isLoading || !mfaSetup}
                      {...register('code')}
                      className={`text-center text-lg tracking-widest ${errors.code ? 'border-red-500' : ''}`}
                    />
                    {errors.code && (
                      <p className="mt-1 text-sm text-red-600">{errors.code.message}</p>
                    )}
                  </div>

                  <Button
                    type="submit"
                    className="w-full"
                    disabled={isLoading || !mfaSetup}
                  >
                    {isLoading ? 'Verifying...' : 'Verify Code'}
                  </Button>
                </form>
              </CardContent>
            </Card>
          </div>
        ) : (
          /* Success and Backup Codes */
          <Card>
            <CardHeader className="text-center">
              <Shield className="mx-auto h-12 w-12 text-green-600" />
              <CardTitle>MFA Setup Complete!</CardTitle>
              <CardDescription>
                Two-factor authentication has been successfully enabled for your account.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <strong>Important:</strong> Save these backup codes in a secure location. 
                  You can use them to access your account if you lose your authenticator device.
                </AlertDescription>
              </Alert>

              {mfaSetup && (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <Label>Backup Codes</Label>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(mfaSetup.backupCodes.join('\n'), 'backup')}
                    >
                      {copiedBackupCodes ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                      {copiedBackupCodes ? 'Copied!' : 'Copy All'}
                    </Button>
                  </div>
                  <div className="grid grid-cols-2 gap-2 p-4 bg-gray-50 rounded-lg font-mono text-sm">
                    {mfaSetup.backupCodes.map((code, index) => (
                      <div key={index} className="text-center py-1">
                        {code}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              <Button onClick={handleComplete} className="w-full">
                Continue to Dashboard
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}