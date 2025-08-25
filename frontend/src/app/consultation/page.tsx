'use client';

import { useState, useEffect } from 'react';
import { useConsultationStore } from '@/stores/consultations';
import { useCreateConsultationSession } from '@/hooks/useConsultations';
import { ConsultationTypeSelector } from '@/components/consultation/consultation-type-selector';
import { ChatInterface } from '@/components/consultation/chat-interface';
import { ConsultationHeader } from '@/components/consultation/consultation-header';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { MessageSquare, Plus } from 'lucide-react';
import type { ConsultationType } from '@/types';

export default function ConsultationPage() {
  const { currentSession, setCurrentSession } = useConsultationStore();
  const createSession = useCreateConsultationSession();
  const [showTypeSelector, setShowTypeSelector] = useState(!currentSession);

  // Reset to type selector when no current session
  useEffect(() => {
    if (!currentSession) {
      setShowTypeSelector(true);
    }
  }, [currentSession]);

  const handleStartConsultation = async (type: ConsultationType, title?: string, context?: string) => {
    try {
      await createSession.mutateAsync({
        type,
        title: title || `${type.charAt(0).toUpperCase() + type.slice(1)} Consultation`,
        ...(context && { context }),
      });
      setShowTypeSelector(false);
    } catch (error) {
      console.error('Failed to create consultation session:', error);
    }
  };

  const handleNewConsultation = () => {
    setCurrentSession(null);
    setShowTypeSelector(true);
  };

  if (showTypeSelector) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">New Consultation</h1>
          <p className="text-muted-foreground">
            Start a conversation with the AI Government Consultant.
          </p>
        </div>

        <ConsultationTypeSelector
          onStartConsultation={handleStartConsultation}
          isLoading={createSession.isPending}
        />
      </div>
    );
  }

  if (!currentSession) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Consultation</h1>
            <p className="text-muted-foreground">
              AI Government Consultant Interface
            </p>
          </div>
          <Button onClick={handleNewConsultation} className="gap-2">
            <Plus className="h-4 w-4" />
            New Consultation
          </Button>
        </div>

        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <MessageSquare className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No Active Consultation</h3>
            <p className="text-muted-foreground text-center mb-4">
              Start a new consultation to begin chatting with the AI assistant.
            </p>
            <Button onClick={handleNewConsultation} className="gap-2">
              <Plus className="h-4 w-4" />
              Start New Consultation
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full">
      <ConsultationHeader
        session={currentSession}
        onNewConsultation={handleNewConsultation}
      />
      <div className="flex-1 min-h-0">
        <ChatInterface session={currentSession} />
      </div>
    </div>
  );
}