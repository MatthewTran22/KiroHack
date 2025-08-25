import { create } from 'zustand';
import { devtools } from 'zustand/middleware';

// Consultation-specific types
export interface ConsultationType {
  id: string;
  name: string;
  description: string;
  icon?: string;
}

export interface ConsultationSession {
  id: string;
  title: string;
  type: string;
  status: 'active' | 'completed' | 'draft' | 'archived';
  createdAt: Date;
  updatedAt: Date;
  userId: string;
  messageCount: number;
  hasUnread: boolean;
  summary?: string;
  tags: string[];
  metadata?: Record<string, unknown>;
}

export interface Message {
  id: string;
  sessionId: string;
  type: 'user' | 'assistant' | 'system';
  role?: 'user' | 'assistant' | 'system'; // Alternative field name for compatibility
  content: string;
  timestamp: Date;
  sources?: DocumentReference[];
  confidence?: number;
  metadata?: MessageMetadata;
  audioUrl?: string;
  transcriptionConfidence?: number;
  inputMethod: 'text' | 'voice';
  isStreaming?: boolean;
  reactions?: MessageReaction[];
  status?: 'sending' | 'sent' | 'delivered' | 'read' | 'failed';
}

export interface DocumentReference {
  id: string;
  title: string;
  type: 'document' | 'knowledge' | 'research';
  excerpt: string;
  confidence: number;
  url?: string;
}

export interface MessageMetadata {
  processingTime?: number;
  modelUsed?: string;
  tokensUsed?: number;
  [key: string]: unknown;
}

export interface MessageReaction {
  type: 'helpful' | 'not_helpful' | 'save' | 'share';
  timestamp: Date;
}

export interface ConsultationFilters {
  type?: string;
  status?: ConsultationSession['status'];
  dateRange?: {
    start: Date;
    end: Date;
  };
  tags?: string[];
  searchQuery?: string;
}

export interface VoiceSettings {
  voice: string;
  speechRate: number;
  volume: number;
  autoPlayResponses: boolean;
  showTranscription: boolean;
  language: string;
}

interface ConsultationState {
  // Current session
  currentSession: ConsultationSession | null;
  currentMessages: Message[];

  // Session management
  sessions: ConsultationSession[];
  filters: ConsultationFilters;

  // Chat state
  isTyping: boolean;
  isConnected: boolean;
  connectionError: string | null;

  // Voice state
  voiceSettings: VoiceSettings;
  isRecording: boolean;
  isPlaying: boolean;
  audioLevel: number;
  transcriptionActive: boolean;
  voicePanelOpen: boolean;
  listeningMode: 'push-to-talk' | 'continuous';

  // UI state
  showSessionList: boolean;
  selectedMessageId: string | null;
}

interface ConsultationActions {
  // Session management
  setCurrentSession: (session: ConsultationSession | null) => void;
  addSession: (session: ConsultationSession) => void;
  updateSession: (id: string, updates: Partial<ConsultationSession>) => void;
  removeSession: (id: string) => void;
  setFilters: (filters: Partial<ConsultationFilters>) => void;
  clearFilters: () => void;

  // Message management
  addMessage: (message: Message) => void;
  updateMessage: (id: string, updates: Partial<Message>) => void;
  removeMessage: (id: string) => void;
  clearMessages: () => void;
  addMessageReaction: (messageId: string, reaction: MessageReaction) => void;

  // Chat state
  setIsTyping: (typing: boolean) => void;
  setConnectionStatus: (connected: boolean, error?: string) => void;

  // Voice actions
  setVoiceSettings: (settings: Partial<VoiceSettings>) => void;
  setIsRecording: (recording: boolean) => void;
  setIsPlaying: (playing: boolean) => void;
  setAudioLevel: (level: number) => void;
  setTranscriptionActive: (active: boolean) => void;
  setVoicePanelOpen: (open: boolean) => void;
  setListeningMode: (mode: 'push-to-talk' | 'continuous') => void;

  // UI actions
  setShowSessionList: (show: boolean) => void;
  setSelectedMessageId: (id: string | null) => void;
}

type ConsultationStore = ConsultationState & ConsultationActions;

const initialVoiceSettings: VoiceSettings = {
  voice: 'default',
  speechRate: 1.0,
  volume: 0.8,
  autoPlayResponses: false,
  showTranscription: true,
  language: 'en-US',
};

const initialState: ConsultationState = {
  currentSession: null,
  currentMessages: [],
  sessions: [],
  filters: {},
  isTyping: false,
  isConnected: false,
  connectionError: null,
  voiceSettings: initialVoiceSettings,
  isRecording: false,
  isPlaying: false,
  audioLevel: 0,
  transcriptionActive: false,
  voicePanelOpen: false,
  listeningMode: 'push-to-talk',
  showSessionList: false,
  selectedMessageId: null,
};

export const useConsultationStore = create<ConsultationStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // Session management
      setCurrentSession: (session: ConsultationSession | null) => {
        set({
          currentSession: session,
          currentMessages: [], // Clear messages when switching sessions
        });
      },

      addSession: (session: ConsultationSession) => {
        set((state) => ({
          sessions: [session, ...state.sessions],
        }));
      },

      updateSession: (id: string, updates: Partial<ConsultationSession>) => {
        set((state) => ({
          sessions: state.sessions.map((session) =>
            session.id === id ? { ...session, ...updates } : session
          ),
          currentSession: state.currentSession?.id === id
            ? { ...state.currentSession, ...updates }
            : state.currentSession,
        }));
      },

      removeSession: (id: string) => {
        set((state) => ({
          sessions: state.sessions.filter((session) => session.id !== id),
          currentSession: state.currentSession?.id === id ? null : state.currentSession,
          currentMessages: state.currentSession?.id === id ? [] : state.currentMessages,
        }));
      },

      setFilters: (newFilters: Partial<ConsultationFilters>) => {
        set((state) => ({
          filters: { ...state.filters, ...newFilters },
        }));
      },

      clearFilters: () => {
        set({ filters: {} });
      },

      // Message management
      addMessage: (message: Message) => {
        set((state) => ({
          currentMessages: [...state.currentMessages, message],
        }));
      },

      updateMessage: (id: string, updates: Partial<Message>) => {
        set((state) => ({
          currentMessages: state.currentMessages.map((message) =>
            message.id === id ? { ...message, ...updates } : message
          ),
        }));
      },

      removeMessage: (id: string) => {
        set((state) => ({
          currentMessages: state.currentMessages.filter((message) => message.id !== id),
        }));
      },

      clearMessages: () => {
        set({ currentMessages: [] });
      },

      addMessageReaction: (messageId: string, reaction: MessageReaction) => {
        set((state) => ({
          currentMessages: state.currentMessages.map((message) =>
            message.id === messageId
              ? {
                ...message,
                reactions: [...(message.reactions || []), reaction],
              }
              : message
          ),
        }));
      },

      // Chat state
      setIsTyping: (typing: boolean) => {
        set({ isTyping: typing });
      },

      setConnectionStatus: (connected: boolean, error?: string) => {
        set({
          isConnected: connected,
          connectionError: error || null,
        });
      },

      // Voice actions
      setVoiceSettings: (settings: Partial<VoiceSettings>) => {
        set((state) => ({
          voiceSettings: { ...state.voiceSettings, ...settings },
        }));
      },

      setIsRecording: (recording: boolean) => {
        set({ isRecording: recording });
      },

      setIsPlaying: (playing: boolean) => {
        set({ isPlaying: playing });
      },

      setAudioLevel: (level: number) => {
        set({ audioLevel: level });
      },

      setTranscriptionActive: (active: boolean) => {
        set({ transcriptionActive: active });
      },

      setVoicePanelOpen: (open: boolean) => {
        set({ voicePanelOpen: open });
      },

      setListeningMode: (mode: 'push-to-talk' | 'continuous') => {
        set({ listeningMode: mode });
      },

      // UI actions
      setShowSessionList: (show: boolean) => {
        set({ showSessionList: show });
      },

      setSelectedMessageId: (id: string | null) => {
        set({ selectedMessageId: id });
      },
    }),
    {
      name: 'consultation-store',
    }
  )
);