'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Shield } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';
import { ROUTES } from '@/lib/constants';

export default function LogoutPage() {
  const { logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    const performLogout = async () => {
      try {
        await logout();
      } catch (error) {
        console.error('Logout error:', error);
      } finally {
        // Always redirect to login after logout attempt
        router.push(ROUTES.LOGIN);
      }
    };

    performLogout();
  }, [logout, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <Shield className="mx-auto h-12 w-12 text-blue-600 animate-spin" />
        <h2 className="mt-4 text-xl font-semibold text-gray-900">
          Signing out...
        </h2>
        <p className="mt-2 text-sm text-gray-600">
          Please wait while we securely log you out.
        </p>
      </div>
    </div>
  );
}