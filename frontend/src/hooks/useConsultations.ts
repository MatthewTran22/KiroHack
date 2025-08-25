import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-client';
import { useConsultationStore } from '@/stores/consultations';
import { PaginatedResponse } from '@/types';
import { apiClient } from '@/lib/api';
import { tokenManager } from '@/lib/auth';
import type {
  ConsultationSession,
  Message,
  ConsultationFilters,
  VoiceSettings
} from '@/stores/consultations';

// Extended API client for consultations
const consultationsAPI = {
  async getSessions(filters?: ConsultationFilters): Promise<PaginatedResponse<ConsultationSession>> {
    // For now, return the proper API client call, but we need to map the filters to the backend format
    const consultationFilters = filters ? {
      searchQuery: filters.searchQuery,
      type: filters.type,
      status: filters.status,
      tags: filters.tags,
      dateRange: filters.dateRange,
    } : undefined;

    return apiClient.consultations.getSessions(consultationFilters);
  },

  async getSession(id: string): Promise<ConsultationSession> {
    return apiClient.consultations.getSession(id);
  },

  async createSession(data: { type: string; title?: string; context?: string }): Promise<ConsultationSession> {
    // Map the data to the backend format that matches CreateConsultationRequest
    const backendRequest = {
      query: data.title || `New ${data.type} consultation`,
      type: data.type,
      context: data.context ? {
        userContext: { description: data.context },
        relatedDocuments: [],
        previousSessions: [],
        systemContext: {}
      } : undefined,
      maxSources: 10,
      confidenceThreshold: 0.7,
      tags: [],
      isMultiTurn: false
    };

    // Since apiClient.consultations.createSession expects ConsultationRequest from frontend types,
    // but the backend actually expects CreateConsultationRequest, we need to make a direct call
    const token = tokenManager.getToken();
    if (!token) {
      throw new Error('No authentication token available');
    }

    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/consultations`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
      },
      body: JSON.stringify(backendRequest),
    });

    if (!response.ok) {
      const errorText = await response.text();
      console.error('Consultation API Error:', response.status, errorText);

      if (response.status === 500) {
        throw new Error('Consultation service is temporarily unavailable. This might be due to missing AI service configuration. Please contact your administrator.');
      }

      throw new Error(`Failed to create consultation session: ${response.status} ${errorText}`);
    }

    const result = await response.json();

    // Map the backend response to frontend ConsultationSession format
    if (result.data && result.data.session) {
      const session = result.data.session;
      return {
        id: session.id || session._id,
        title: session.query || data.title || `${data.type} consultation`,
        type: session.type,
        status: session.status || 'completed',
        createdAt: new Date(session.created_at || new Date()),
        updatedAt: new Date(session.updated_at || new Date()),
        userId: session.user_id || session.userId,
        messages: [],
        context: data.context,
        priority: 'medium',
      };
    }

    throw new Error('Invalid response format from server');
  },

  async updateSession(id: string, updates: Partial<ConsultationSession>): Promise<ConsultationSession> {
    return apiClient.consultations.updateSession(id, updates);
  },

  async deleteSession(id: string): Promise<void> {
    return apiClient.consultations.deleteSession(id);
  },

  async getMessages(sessionId: string): Promise<Message[]> {
    // The backend doesn't have separate message endpoints yet
    // For now, return empty array or get from session data
    return apiClient.consultations.getMessages(sessionId);
  },

  async sendMessage(sessionId: string, content: string, inputMethod: 'text' | 'voice' = 'text'): Promise<Message> {
    // The backend uses continue consultation for additional messages
    const messageRequest = {
      content,
      inputMethod,
    };
    return apiClient.consultations.sendMessage(sessionId, messageRequest);
  },

  async sendVoiceMessage(sessionId: string, audioBlob: Blob): Promise<Message> {
    // Voice messages not implemented in backend yet
    throw new Error('Voice messages are not yet implemented');
  },

  async transcribeAudio(audioBlob: Blob, options?: { language?: string }): Promise<{ text: string; confidence: number }> {
    // Audio transcription not implemented in backend yet
    throw new Error('Audio transcription is not yet implemented');
  },

  async synthesizeSpeech(text: string, options?: { voice?: string; rate?: number }): Promise<Blob> {
    // Speech synthesis not implemented in backend yet
    throw new Error('Speech synthesis is not yet implemented');
  },

  async getAvailableVoices(): Promise<{ id: string; name: string; language: string }[]> {
    // Voice functionality not implemented in backend yet
    return [];
  },

  async exportSession(sessionId: string, format: 'pdf' | 'docx' | 'txt' = 'pdf'): Promise<Blob> {
    return apiClient.consultations.exportSession(sessionId, format);
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