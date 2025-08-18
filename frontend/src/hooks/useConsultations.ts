import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-client';
import { useConsultationStore } from '@/stores/consultations';
import { PaginatedResponse } from '@/types';
import type { 
  ConsultationSession, 
  Message, 
  ConsultationFilters,
  VoiceSettings 
} from '@/stores/consultations';

// Extended API client for consultations
const consultationsAPI = {
  async getSessions(filters?: ConsultationFilters): Promise<PaginatedResponse<ConsultationSession>> {
    const params = new URLSearchParams();
    
    if (filters?.searchQuery) params.append('search', filters.searchQuery);
    if (filters?.type) params.append('type', filters.type);
    if (filters?.status) params.append('status', filters.status);
    if (filters?.tags?.length) params.append('tags', filters.tags.join(','));
    if (filters?.dateRange) {
      params.append('startDate', filters.dateRange.start.toISOString());
      params.append('endDate', filters.dateRange.end.toISOString());
    }

    const response = await fetch(`/api/consultations?${params.toString()}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch consultation sessions');
    }

    return response.json();
  },

  async getSession(id: string): Promise<ConsultationSession> {
    const response = await fetch(`/api/consultations/${id}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch consultation session');
    }

    return response.json();
  },

  async createSession(data: { type: string; title?: string; context?: string }): Promise<ConsultationSession> {
    const response = await fetch('/api/consultations', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      throw new Error('Failed to create consultation session');
    }

    return response.json();
  },

  async updateSession(id: string, updates: Partial<ConsultationSession>): Promise<ConsultationSession> {
    const response = await fetch(`/api/consultations/${id}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(updates),
    });

    if (!response.ok) {
      throw new Error('Failed to update consultation session');
    }

    return response.json();
  },

  async deleteSession(id: string): Promise<void> {
    const response = await fetch(`/api/consultations/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to delete consultation session');
    }
  },

  async getMessages(sessionId: string): Promise<Message[]> {
    const response = await fetch(`/api/consultations/${sessionId}/messages`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch messages');
    }

    return response.json();
  },

  async sendMessage(sessionId: string, content: string, inputMethod: 'text' | 'voice' = 'text'): Promise<Message> {
    const response = await fetch(`/api/consultations/${sessionId}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({ content, inputMethod }),
    });

    if (!response.ok) {
      throw new Error('Failed to send message');
    }

    return response.json();
  },

  async sendVoiceMessage(sessionId: string, audioBlob: Blob): Promise<Message> {
    const formData = new FormData();
    formData.append('audio', audioBlob, 'voice-message.webm');

    const response = await fetch(`/api/consultations/${sessionId}/voice`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('Failed to send voice message');
    }

    return response.json();
  },

  async transcribeAudio(audioBlob: Blob, options?: { language?: string }): Promise<{ text: string; confidence: number }> {
    const formData = new FormData();
    formData.append('audio', audioBlob, 'audio.webm');
    if (options?.language) {
      formData.append('language', options.language);
    }

    const response = await fetch('/api/speech/transcribe', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('Failed to transcribe audio');
    }

    return response.json();
  },

  async synthesizeSpeech(text: string, options?: { voice?: string; rate?: number }): Promise<Blob> {
    const response = await fetch('/api/speech/synthesize', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({ text, ...options }),
    });

    if (!response.ok) {
      throw new Error('Failed to synthesize speech');
    }

    return response.blob();
  },

  async getAvailableVoices(): Promise<{ id: string; name: string; language: string }[]> {
    const response = await fetch('/api/speech/voices', {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch available voices');
    }

    return response.json();
  },

  async exportSession(sessionId: string, format: 'pdf' | 'docx' | 'txt' = 'pdf'): Promise<Blob> {
    const response = await fetch(`/api/consultations/${sessionId}/export?format=${format}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to export session');
    }

    return response.blob();
  },
};

// Hook for fetching consultation sessions
export function useConsultationSessions() {
  const { filters } = useConsultationStore();
  
  return useQuery({
    queryKey: queryKeys.consultations.list(filters),
    queryFn: () => consultationsAPI.getSessions(filters),
    staleTime: 2 * 60 * 1000, // 2 minutes
  });
}

// Hook for fetching a single consultation session
export function useConsultationSession(id: string) {
  return useQuery({
    queryKey: queryKeys.consultations.detail(id),
    queryFn: () => consultationsAPI.getSession(id),
    enabled: !!id,
  });
}

// Hook for fetching messages for a session
export function useConsultationMessages(sessionId: string) {
  return useQuery({
    queryKey: queryKeys.consultations.messages(sessionId),
    queryFn: () => consultationsAPI.getMessages(sessionId),
    enabled: !!sessionId,
    staleTime: 30 * 1000, // 30 seconds
  });
}

// Hook for creating a new consultation session
export function useCreateConsultationSession() {
  const queryClient = useQueryClient();
  const { addSession, setCurrentSession } = useConsultationStore();

  return useMutation({
    mutationFn: consultationsAPI.createSession,
    onSuccess: (newSession) => {
      // Add to store
      addSession(newSession);
      setCurrentSession(newSession);

      // Update cache
      queryClient.setQueryData(
        queryKeys.consultations.detail(newSession.id),
        newSession
      );

      // Invalidate sessions list
      queryClient.invalidateQueries({ queryKey: queryKeys.consultations.lists() });
    },
  });
}

// Hook for updating a consultation session
export function useUpdateConsultationSession() {
  const queryClient = useQueryClient();
  const { updateSession } = useConsultationStore();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<ConsultationSession> }) =>
      consultationsAPI.updateSession(id, updates),
    onSuccess: (updatedSession) => {
      // Update store
      updateSession(updatedSession.id, updatedSession);

      // Update cache
      queryClient.setQueryData(
        queryKeys.consultations.detail(updatedSession.id),
        updatedSession
      );

      // Update in lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.consultations.lists() },
        (oldData: PaginatedResponse<ConsultationSession> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: oldData.data.map((session) =>
              session.id === updatedSession.id ? updatedSession : session
            ),
          };
        }
      );
    },
  });
}

// Hook for deleting a consultation session
export function useDeleteConsultationSession() {
  const queryClient = useQueryClient();
  const { removeSession } = useConsultationStore();

  return useMutation({
    mutationFn: consultationsAPI.deleteSession,
    onSuccess: (_, deletedId) => {
      // Remove from store
      removeSession(deletedId);

      // Remove from cache
      queryClient.removeQueries({ queryKey: queryKeys.consultations.detail(deletedId) });
      queryClient.removeQueries({ queryKey: queryKeys.consultations.messages(deletedId) });

      // Remove from lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.consultations.lists() },
        (oldData: PaginatedResponse<ConsultationSession> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: oldData.data.filter((session) => session.id !== deletedId),
            pagination: {
              ...oldData.pagination,
              total: oldData.pagination.total - 1,
            },
          };
        }
      );
    },
  });
}

// Hook for sending a text message
export function useSendMessage() {
  const queryClient = useQueryClient();
  const { addMessage, currentSession } = useConsultationStore();

  return useMutation({
    mutationFn: ({ sessionId, content, inputMethod }: { 
      sessionId: string; 
      content: string; 
      inputMethod?: 'text' | 'voice' 
    }) => consultationsAPI.sendMessage(sessionId, content, inputMethod),
    onMutate: async ({ sessionId, content, inputMethod = 'text' }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.consultations.messages(sessionId) });

      // Snapshot previous value
      const previousMessages = queryClient.getQueryData(queryKeys.consultations.messages(sessionId));

      // Optimistically update
      const optimisticMessage: Message = {
        id: `temp_${Date.now()}`,
        sessionId,
        type: 'user',
        content,
        timestamp: new Date(),
        inputMethod,
      };

      queryClient.setQueryData(
        queryKeys.consultations.messages(sessionId),
        (old: Message[] | undefined) => [...(old || []), optimisticMessage]
      );

      addMessage(optimisticMessage);

      return { previousMessages, optimisticMessage };
    },
    onSuccess: (newMessage, variables, context) => {
      // Replace optimistic message with real one
      queryClient.setQueryData(
        queryKeys.consultations.messages(variables.sessionId),
        (old: Message[] | undefined) => 
          (old || []).map((msg) => 
            msg.id === context?.optimisticMessage.id ? newMessage : msg
          )
      );
    },
    onError: (err, variables, context) => {
      // Rollback on error
      if (context?.previousMessages) {
        queryClient.setQueryData(
          queryKeys.consultations.messages(variables.sessionId),
          context.previousMessages
        );
      }
    },
  });
}

// Hook for sending voice messages
export function useSendVoiceMessage() {
  const queryClient = useQueryClient();
  const { addMessage } = useConsultationStore();

  return useMutation({
    mutationFn: ({ sessionId, audioBlob }: { sessionId: string; audioBlob: Blob }) =>
      consultationsAPI.sendVoiceMessage(sessionId, audioBlob),
    onSuccess: (newMessage) => {
      addMessage(newMessage);
      queryClient.setQueryData(
        queryKeys.consultations.messages(newMessage.sessionId),
        (old: Message[] | undefined) => [...(old || []), newMessage]
      );
    },
  });
}

// Hook for audio transcription
export function useTranscribeAudio() {
  return useMutation({
    mutationFn: ({ audioBlob, options }: { 
      audioBlob: Blob; 
      options?: { language?: string } 
    }) => consultationsAPI.transcribeAudio(audioBlob, options),
  });
}

// Hook for text-to-speech synthesis
export function useSynthesizeSpeech() {
  return useMutation({
    mutationFn: ({ text, options }: { 
      text: string; 
      options?: { voice?: string; rate?: number } 
    }) => consultationsAPI.synthesizeSpeech(text, options),
  });
}

// Hook for getting available voices
export function useAvailableVoices() {
  return useQuery({
    queryKey: ['speech', 'voices'],
    queryFn: consultationsAPI.getAvailableVoices,
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
}

// Hook for exporting consultation sessions
export function useExportConsultationSession() {
  return useMutation({
    mutationFn: ({ sessionId, format }: { sessionId: string; format?: 'pdf' | 'docx' | 'txt' }) =>
      consultationsAPI.exportSession(sessionId, format),
    onSuccess: (blob, { sessionId, format = 'pdf' }) => {
      // Create download link
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `consultation-${sessionId}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    },
  });
}