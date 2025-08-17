'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { History } from 'lucide-react';

export default function HistoryPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Consultation History</h1>
        <p className="text-muted-foreground">
          View and search your past consultations.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            History Interface
          </CardTitle>
          <CardDescription>
            Consultation history and search functionality (Task 11)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            The history interface will be implemented in a future task.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}