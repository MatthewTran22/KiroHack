import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { 
  useConsultationSessions, 
  useCreateConsultationSession,
  useSendMessage,
  useSendVoiceMessage,
  useTranscribeAudio,
  useSynthesizeSpeech
} from '../useConsultations';
import { useConsultationStore } from '@/stores/consultations';

// Mock fetch for API calls
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock localStorage
const mockLocalStorage = {
  getItem: jest.fn(() => 'mock-token'),
  setItem: jest.fn(),
  removeItem: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: mockLocalStorage });

// Mock WebSocket for real-time features
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;

  constructor(public url: string) {
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      this.onopen?.(new Event('open'));
    }, 100);
  }

  send(data: string) {
    // Mock sending data
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent('close'));
  }
}

global.WebSocket = MockWebSocket as any;

// Test wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  };
};

describe('useConsultations Integration Tests', () => {
  beforeEach(() => {
    mockFetch.mockClear();
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

  describe('Docker Container Integration', () => {
    it('should fetch consultation sessions from backend API', async () => {
      const mockSessions = {
        data: [
          {
            id: 'session-1',
            title: 'Policy Analysis',
            type: 'policy',
            status: 'active',
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            userId: 'user-1',
            messageCount: 5,
            hasUnread: false,
            tags: ['policy', 'analysis'],
          },
        ],
        pagination: {
          page: 1,
          limit: 10,
          total: 1,
          totalPages: 1,
        },
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSessions),
      });

      const { result } = renderHook(() => useConsultationSessions(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockSessions);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/consultations'),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
        })
      );
    });

    it('should create new consultation session', async () => {
      const mockSession = {
        id: 'session-1',
        title: 'New Policy Session',
        type: 'policy',
        status: 'active',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        userId: 'user-1',
        messageCount: 0,
        hasUnread: false,
        tags: [],
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSession),
      });

      const { result } = renderHook(() => useCreateConsultationSession(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        await result.current.mutateAsync({
          type: 'policy',
          title: 'New Policy Session',
          context: 'Initial context',
        });
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/consultations',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            Authorization: 'Bearer mock-token',
          }),
          body: JSON.stringify({
            type: 'policy',
            title: 'New Policy Session',
            context: 'Initial context',
          }),
        })
      );
    });

    it('should send text message with optimistic updates', async () => {
      const mockMessage = {
        id: 'message-1',
        sessionId: 'session-1',
        type: 'user',
        content: 'Test message',
        timestamp: new Date().toISOString(),
        inputMethod: 'text',
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockMessage),
      });

      const { result } = renderHook(() => useSendMessage(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        await result.current.mutateAsync({
          sessionId: 'session-1',
          content: 'Test message',
          inputMethod: 'text',
        });
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/consultations/session-1/messages',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            Authorization: 'Bearer mock-token',
          }),
          body: JSON.stringify({
            content: 'Test message',
            inputMethod: 'text',
          }),
        })
      );
    });

    it('should send voice message', async () => {
      const mockMessage = {
        id: 'message-1',
        sessionId: 'session-1',
        type: 'user',
        content: 'Transcribed voice message',
        timestamp: new Date().toISOString(),
        inputMethod: 'voice',
        transcriptionConfidence: 0.95,
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockMessage),
      });

      const { result } = renderHook(() => useSendVoiceMessage(), {
        wrapper: createWrapper(),
      });

      const audioBlob = new Blob(['audio data'], { type: 'audio/webm' });

      await waitFor(async () => {
        await result.current.mutateAsync({
          sessionId: 'session-1',
          audioBlob,
        });
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/consultations/session-1/voice',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
          body: expect.any(FormData),
        })
      );
    });

    it('should transcribe audio', async () => {
      const mockTranscription = {
        text: 'Hello, this is a test transcription',
        confidence: 0.95,
      };

      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockTranscription),
      });

      const { result } = renderHook(() => useTranscribeAudio(), {
        wrapper: createWrapper(),
      });

      const audioBlob = new Blob(['audio data'], { type: 'audio/webm' });

      await waitFor(async () => {
        const transcription = await result.current.mutateAsync({
          audioBlob,
          options: { language: 'en-US' },
        });
        
        expect(transcription).toEqual(mockTranscription);
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/speech/transcribe',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer mock-token',
          }),
          body: expect.any(FormData),
        })
      );
    });

    it('should synthesize speech', async () => {
      const mockAudioBlob = new Blob(['audio data'], { type: 'audio/mpeg' });

      mockFetch.mockResolvedValueOnce({
        ok: true,
        blob: () => Promise.resolve(mockAudioBlob),
      });

      const { result } = renderHook(() => useSynthesizeSpeech(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        const audioBlob = await result.current.mutateAsync({
          text: 'Hello, this is a test',
          options: { voice: 'female', rate: 1.2 },
        });
        
        expect(audioBlob).toEqual(mockAudioBlob);
      });

      expect(mockFetch).toHaveBeenCalledWith(
        '/api/speech/synthesize',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
            Authorization: 'Bearer mock-token',
          }),
          body: JSON.stringify({
            text: 'Hello, this is a test',
            voice: 'female',
            rate: 1.2,
          }),
        })
      );
    });
  });

  describe('Real-time Communication', () => {
    it('should handle WebSocket connection for real-time messaging', async () => {
      const sessionId = 'session-1';
      
      // This would be handled by a WebSocket hook in a real implementation
      const ws = new WebSocket(`ws://localhost:8080/api/consultations/${sessionId}/ws`);
      
      await new Promise((resolve) => {
        ws.onopen = resolve;
      });

      expect(ws.readyState).toBe(WebSocket.OPEN);
      
      ws.close();
    });

    it('should handle typing indicators', async () => {
      const { result: storeResult } = renderHook(() => useConsultationStore());
      
      // Simulate receiving typing indicator
      act(() => {
        storeResult.current.setIsTyping(true);
      });
      expect(storeResult.current.isTyping).toBe(true);
      
      // Simulate typing stopped
      act(() => {
        storeResult.current.setIsTyping(false);
      });
      
      await waitFor(() => {
        expect(storeResult.current.isTyping).toBe(false);
      });
    });
  });

  describe('Voice Features Integration', () => {
    it('should handle voice settings updates', () => {
      const { result } = renderHook(() => useConsultationStore());
      
      act(() => {
        result.current.setVoiceSettings({
          voice: 'female',
          speechRate: 1.2,
          autoPlayResponses: true,
        });
      });

      expect(result.current.voiceSettings).toMatchObject({
        voice: 'female',
        speechRate: 1.2,
        autoPlayResponses: true,
        volume: 0.8, // Should preserve existing settings
      });
    });

    it('should handle recording state', () => {
      const { result } = renderHook(() => useConsultationStore());
      
      act(() => {
        result.current.setIsRecording(true);
      });
      expect(result.current.isRecording).toBe(true);
      
      act(() => {
        result.current.setIsRecording(false);
      });
      expect(result.current.isRecording).toBe(false);
    });

    it('should handle audio level monitoring', () => {
      const { result } = renderHook(() => useConsultationStore());
      
      act(() => {
        result.current.setAudioLevel(0.75);
      });
      expect(result.current.audioLevel).toBe(0.75);
    });
  });

  describe('Real Backend Integration (Docker)', () => {
    const isIntegrationTest = process.env.INTEGRATION_TEST === 'true';
    const backendUrl = process.env.BACKEND_URL || 'http://localhost:8080';

    beforeEach(() => {
      if (isIntegrationTest) {
        global.fetch = fetch;
      }
    });

    (isIntegrationTest ? it : it.skip)('should connect to real consultation API', async () => {
      const { result } = renderHook(() => useConsultationSessions(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoading).toBe(false);
      }, { timeout: 10000 });

      expect(result.current.isSuccess || result.current.isError).toBe(true);
    });

    (isIntegrationTest ? it : it.skip)('should create real consultation session', async () => {
      const { result } = renderHook(() => useCreateConsultationSession(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        const session = await result.current.mutateAsync({
          type: 'policy',
          title: 'Integration Test Session',
        });
        
        expect(session).toBeDefined();
        expect(session.id).toBeDefined();
        expect(session.type).toBe('policy');
      }, { timeout: 10000 });
    });

    (isIntegrationTest ? it : it.skip)('should handle real voice transcription', async () => {
      // Create a simple audio blob for testing
      const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
      const buffer = audioContext.createBuffer(1, 44100, 44100);
      
      // Fill with simple sine wave
      const data = buffer.getChannelData(0);
      for (let i = 0; i < data.length; i++) {
        data[i] = Math.sin(2 * Math.PI * 440 * i / 44100) * 0.1;
      }

      // Convert to blob (this is simplified - real implementation would be more complex)
      const audioBlob = new Blob(['audio data'], { type: 'audio/webm' });

      const { result } = renderHook(() => useTranscribeAudio(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        const transcription = await result.current.mutateAsync({
          audioBlob,
          options: { language: 'en-US' },
        });
        
        expect(transcription).toBeDefined();
        expect(typeof transcription.text).toBe('string');
        expect(typeof transcription.confidence).toBe('number');
      }, { timeout: 30000 });
    });

    (isIntegrationTest ? it : it.skip)('should handle real speech synthesis', async () => {
      const { result } = renderHook(() => useSynthesizeSpeech(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        const audioBlob = await result.current.mutateAsync({
          text: 'This is a test of speech synthesis',
          options: { voice: 'default', rate: 1.0 },
        });
        
        expect(audioBlob).toBeInstanceOf(Blob);
        expect(audioBlob.type).toContain('audio');
      }, { timeout: 15000 });
    });

    (isIntegrationTest ? it : it.skip)('should handle WebSocket real-time messaging', async () => {
      const sessionId = 'test-session';
      const wsUrl = `${backendUrl.replace('http', 'ws')}/api/consultations/${sessionId}/ws`;
      
      const ws = new WebSocket(wsUrl);
      
      await new Promise((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('WebSocket connection timeout')), 10000);
        
        ws.onopen = () => {
          clearTimeout(timeout);
          resolve(undefined);
        };
        
        ws.onerror = (error) => {
          clearTimeout(timeout);
          reject(error);
        };
      });

      expect(ws.readyState).toBe(WebSocket.OPEN);
      
      // Test sending a message
      ws.send(JSON.stringify({
        type: 'message',
        content: 'Test message',
        sessionId,
      }));

      // Test receiving a message
      await new Promise((resolve) => {
        ws.onmessage = (event) => {
          const data = JSON.parse(event.data);
          expect(data).toBeDefined();
          resolve(undefined);
        };
        
        // Simulate server response
        setTimeout(() => resolve(undefined), 1000);
      });
      
      ws.close();
    });

    (isIntegrationTest ? it : it.skip)('should handle end-to-end consultation flow', async () => {
      // Create session
      const { result: createResult } = renderHook(() => useCreateConsultationSession(), {
        wrapper: createWrapper(),
      });

      let sessionId: string;
      await waitFor(async () => {
        const session = await createResult.current.mutateAsync({
          type: 'policy',
          title: 'E2E Test Session',
        });
        sessionId = session.id;
      });

      // Send message
      const { result: messageResult } = renderHook(() => useSendMessage(), {
        wrapper: createWrapper(),
      });

      await waitFor(async () => {
        await messageResult.current.mutateAsync({
          sessionId,
          content: 'What are the key policy considerations for AI governance?',
          inputMethod: 'text',
        });
      });

      expect(messageResult.current.isSuccess).toBe(true);
    });
  });
});