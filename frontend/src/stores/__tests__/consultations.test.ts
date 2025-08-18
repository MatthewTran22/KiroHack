import { act, renderHook } from '@testing-library/react';
import { useConsultationStore } from '../consultations';
import type { ConsultationSession, Message, VoiceSettings } from '../consultations';

const mockSession: ConsultationSession = {
  id: 'session-1',
  title: 'Test Session',
  type: 'policy',
  status: 'active',
  createdAt: new Date(),
  updatedAt: new Date(),
  userId: 'user-1',
  messageCount: 0,
  hasUnread: false,
  tags: [],
};

const mockMessage: Message = {
  id: 'message-1',
  sessionId: 'session-1',
  type: 'user',
  content: 'Test message',
  timestamp: new Date(),
  inputMethod: 'text',
};

describe('Consultation Store', () => {
  beforeEach(() => {
    // Reset store state before each test
    useConsultationStore.setState({
      currentSession: null,
      currentMessages: [],
      sessions: [],
      filters: {},
      isTyping: false,
      isConnected: false,
      connectionError: null,
      voiceSettings: {
        voice: 'default',
        speechRate: 1.0,
        volume: 0.8,
        autoPlayResponses: false,
        showTranscription: true,
        language: 'en-US',
      },
      isRecording: false,
      isPlaying: false,
      audioLevel: 0,
      transcriptionActive: false,
      voicePanelOpen: false,
      listeningMode: 'push-to-talk',
      showSessionList: false,
      selectedMessageId: null,
    });
  });

  describe('Session Management', () => {
    it('should set current session', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setCurrentSession(mockSession);
      });

      expect(result.current.currentSession).toEqual(mockSession);
      expect(result.current.currentMessages).toEqual([]); // Should clear messages
    });

    it('should clear current session', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setCurrentSession(mockSession);
        result.current.addMessage(mockMessage);
        result.current.setCurrentSession(null);
      });

      expect(result.current.currentSession).toBeNull();
      expect(result.current.currentMessages).toEqual([]);
    });

    it('should add session', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.addSession(mockSession);
      });

      expect(result.current.sessions).toEqual([mockSession]);
    });

    it('should add multiple sessions in correct order', () => {
      const { result } = renderHook(() => useConsultationStore());
      const session2 = { ...mockSession, id: 'session-2', title: 'Session 2' };

      act(() => {
        result.current.addSession(mockSession);
        result.current.addSession(session2);
      });

      expect(result.current.sessions).toEqual([session2, mockSession]); // Newest first
    });

    it('should update session', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.addSession(mockSession);
        result.current.updateSession('session-1', { title: 'Updated Title' });
      });

      expect(result.current.sessions[0].title).toBe('Updated Title');
    });

    it('should update current session when it matches', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setCurrentSession(mockSession);
        result.current.updateSession('session-1', { title: 'Updated Title' });
      });

      expect(result.current.currentSession?.title).toBe('Updated Title');
    });

    it('should remove session', () => {
      const { result } = renderHook(() => useConsultationStore());
      const session2 = { ...mockSession, id: 'session-2' };

      act(() => {
        result.current.addSession(mockSession);
        result.current.addSession(session2);
        result.current.removeSession('session-1');
      });

      expect(result.current.sessions).toEqual([session2]);
    });

    it('should clear current session when removing it', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setCurrentSession(mockSession);
        result.current.addMessage(mockMessage);
        result.current.removeSession('session-1');
      });

      expect(result.current.currentSession).toBeNull();
      expect(result.current.currentMessages).toEqual([]);
    });
  });

  describe('Filter Management', () => {
    it('should set filters', () => {
      const { result } = renderHook(() => useConsultationStore());
      const filters = { type: 'policy', status: 'active' as const };

      act(() => {
        result.current.setFilters(filters);
      });

      expect(result.current.filters).toEqual(filters);
    });

    it('should merge filters', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setFilters({ type: 'policy' });
        result.current.setFilters({ status: 'active' });
      });

      expect(result.current.filters).toEqual({
        type: 'policy',
        status: 'active',
      });
    });

    it('should clear filters', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setFilters({ type: 'policy', status: 'active' });
        result.current.clearFilters();
      });

      expect(result.current.filters).toEqual({});
    });
  });

  describe('Message Management', () => {
    it('should add message', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.addMessage(mockMessage);
      });

      expect(result.current.currentMessages).toEqual([mockMessage]);
    });

    it('should add multiple messages in order', () => {
      const { result } = renderHook(() => useConsultationStore());
      const message2 = { ...mockMessage, id: 'message-2', content: 'Second message' };

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.addMessage(message2);
      });

      expect(result.current.currentMessages).toEqual([mockMessage, message2]);
    });

    it('should update message', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.updateMessage('message-1', { content: 'Updated content' });
      });

      expect(result.current.currentMessages[0].content).toBe('Updated content');
    });

    it('should remove message', () => {
      const { result } = renderHook(() => useConsultationStore());
      const message2 = { ...mockMessage, id: 'message-2' };

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.addMessage(message2);
        result.current.removeMessage('message-1');
      });

      expect(result.current.currentMessages).toEqual([message2]);
    });

    it('should clear messages', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.clearMessages();
      });

      expect(result.current.currentMessages).toEqual([]);
    });

    it('should add message reaction', () => {
      const { result } = renderHook(() => useConsultationStore());
      const reaction = { type: 'helpful' as const, timestamp: new Date() };

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.addMessageReaction('message-1', reaction);
      });

      expect(result.current.currentMessages[0].reactions).toEqual([reaction]);
    });

    it('should add multiple reactions to message', () => {
      const { result } = renderHook(() => useConsultationStore());
      const reaction1 = { type: 'helpful' as const, timestamp: new Date() };
      const reaction2 = { type: 'save' as const, timestamp: new Date() };

      act(() => {
        result.current.addMessage(mockMessage);
        result.current.addMessageReaction('message-1', reaction1);
        result.current.addMessageReaction('message-1', reaction2);
      });

      expect(result.current.currentMessages[0].reactions).toEqual([reaction1, reaction2]);
    });
  });

  describe('Chat State Management', () => {
    it('should set typing state', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setIsTyping(true);
      });

      expect(result.current.isTyping).toBe(true);
    });

    it('should set connection status', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setConnectionStatus(true);
      });

      expect(result.current.isConnected).toBe(true);
      expect(result.current.connectionError).toBeNull();
    });

    it('should set connection error', () => {
      const { result } = renderHook(() => useConsultationStore());
      const error = 'Connection failed';

      act(() => {
        result.current.setConnectionStatus(false, error);
      });

      expect(result.current.isConnected).toBe(false);
      expect(result.current.connectionError).toBe(error);
    });
  });

  describe('Voice Settings Management', () => {
    it('should update voice settings', () => {
      const { result } = renderHook(() => useConsultationStore());
      const newSettings: Partial<VoiceSettings> = {
        voice: 'female',
        speechRate: 1.2,
        autoPlayResponses: true,
      };

      act(() => {
        result.current.setVoiceSettings(newSettings);
      });

      expect(result.current.voiceSettings).toMatchObject(newSettings);
      expect(result.current.voiceSettings.volume).toBe(0.8); // Should preserve other settings
    });

    it('should set recording state', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setIsRecording(true);
      });

      expect(result.current.isRecording).toBe(true);
    });

    it('should set playing state', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setIsPlaying(true);
      });

      expect(result.current.isPlaying).toBe(true);
    });

    it('should set audio level', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setAudioLevel(0.75);
      });

      expect(result.current.audioLevel).toBe(0.75);
    });

    it('should set transcription active state', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setTranscriptionActive(true);
      });

      expect(result.current.transcriptionActive).toBe(true);
    });

    it('should toggle voice panel', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setVoicePanelOpen(true);
      });

      expect(result.current.voicePanelOpen).toBe(true);
    });

    it('should set listening mode', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setListeningMode('continuous');
      });

      expect(result.current.listeningMode).toBe('continuous');
    });
  });

  describe('UI State Management', () => {
    it('should toggle session list', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setShowSessionList(true);
      });

      expect(result.current.showSessionList).toBe(true);
    });

    it('should set selected message', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setSelectedMessageId('message-1');
      });

      expect(result.current.selectedMessageId).toBe('message-1');
    });

    it('should clear selected message', () => {
      const { result } = renderHook(() => useConsultationStore());

      act(() => {
        result.current.setSelectedMessageId('message-1');
        result.current.setSelectedMessageId(null);
      });

      expect(result.current.selectedMessageId).toBeNull();
    });
  });

  describe('Store Persistence', () => {
    it('should maintain state across hook instances', () => {
      const { result: result1 } = renderHook(() => useConsultationStore());
      
      act(() => {
        result1.current.setCurrentSession(mockSession);
        result1.current.addMessage(mockMessage);
        result1.current.setVoiceSettings({ voice: 'custom' });
      });

      const { result: result2 } = renderHook(() => useConsultationStore());
      
      expect(result2.current.currentSession).toEqual(mockSession);
      expect(result2.current.currentMessages).toEqual([mockMessage]);
      expect(result2.current.voiceSettings.voice).toBe('custom');
    });
  });
});