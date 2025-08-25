'use client';

import { useAuth } from '@/hooks/useAuth';
import { tokenManager } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

export function AuthDebug() {
    const { user, isAuthenticated, isLoading, error } = useAuth();

    const handleTestToken = () => {
        const token = tokenManager.getToken();
        const isValid = tokenManager.isTokenValid();
        const payload = tokenManager.getTokenPayload();

        console.log('Token Debug Info:', {
            hasToken: !!token,
            tokenLength: token?.length,
            isValid,
            payload,
            localStorage: {
                auth_token: localStorage.getItem('auth_token'),
                token: localStorage.getItem('token'),
                refresh_token: localStorage.getItem('refresh_token'),
            }
        });
    };

    return (
        <Card className="w-full max-w-md">
            <CardHeader>
                <CardTitle>Authentication Debug</CardTitle>
                <CardDescription>Current authentication state</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                <div>
                    <strong>Status:</strong> {isLoading ? 'Loading...' : isAuthenticated ? 'Authenticated' : 'Not Authenticated'}
                </div>

                {error && (
                    <div className="text-red-600">
                        <strong>Error:</strong> {error}
                    </div>
                )}

                {user && (
                    <div>
                        <strong>User:</strong> {user.name} ({user.email})
                        <br />
                        <strong>Role:</strong> {user.role}
                    </div>
                )}

                <Button onClick={handleTestToken} variant="outline">
                    Log Token Info to Console
                </Button>
            </CardContent>
        </Card>
    );
}