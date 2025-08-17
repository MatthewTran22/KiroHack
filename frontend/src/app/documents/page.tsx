'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FileText } from 'lucide-react';

export default function DocumentsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Documents</h1>
        <p className="text-muted-foreground">
          Manage your uploaded documents and files.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Document Management
          </CardTitle>
          <CardDescription>
            Document upload and management interface (Task 6)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            The document management interface will be implemented in a future task.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}