'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { MessageSquare } from 'lucide-react';

export default function ConsultationPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">New Consultation</h1>
        <p className="text-muted-foreground">
          Start a conversation with the AI Government Consultant.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MessageSquare className="h-5 w-5" />
            Chat Interface
          </CardTitle>
          <CardDescription>
            This will be the main consultation interface (Task 7)
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground">
            The chat interface will be implemented in a future task.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}