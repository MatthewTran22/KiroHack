import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ChatInterface } from '../chat-interface';
import { useConsultationStore } from '@/stores/consultations';
import { useConsultationMessages, useSendMessage } from '@/hooks/useConsultations';

// Mock the hooks
jest.mock('@/stores/consultations');
jest.mock('@/hooks/useConsultations');

const mockUseConsultationStore = useConsultationStore as jest.MockedFunction<typeof useConsultationStore>;
const mockUseConsultationMessages = useConsultationMessages as jest.MockedFunction<typeof useConsultationMessages>;
const mockUseSendMessage = useSendMessage as jest.MockedFunction<typeof useSendMessage>;

const mockSession = {
    id: 'session-1',
    title: 'Test Policy Session',
    type: 'policy' as const,
    status: 'active' as const,
    createdAt: new Date(),
    updatedAt: new Date(),
    userId: 'user-1',
    messageCount: 0,
    hasUnread: false,
    tags: [],
};

const mockStoreState = {
    currentMessages: [],
    isTyping: false,
    isConnected: true,
    connectionError: null,
    voicePanelOpen: false,
    setVoicePanelOpen: jest.fn(),
    isRecording: false,
    voiceSettings: {
        voice: 'default',
        speechRate: 1.0,
        volume: 0.8,
        autoPlayResponses: false,
        showTranscription: true,
        language: 'en-US',
    },
    setVoiceSettings: jest.fn(),
};

const mockSendMessage = {
    mutateAsync: jest.fn(),
    isPending: false,
    isError: false,
    error: null,
};

function renderWithQueryClient(component: React.ReactElement) {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });

    return render(
        <QueryClientProvider client={queryClient}>
            {component}
        </QueryClientProvider>
    );
}

describe('ChatInterface', () => {
    beforeEach(() => {
        jest.clearAllMocks();

        mockUseConsultationStore.mockReturnValue(mockStoreState as any);
        mockUseConsultationMessages.mockReturnValue({
            data: [],
            isLoading: false,
            error: null,
        } as any);
        mockUseSendMessage.mockReturnValue(mockSendMessage as any);
    });

    it('renders chat interface with connection status', () => {
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Connected')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('Ask about policy...')).toBeInTheDocument();
    });

    it('shows disconnected state when not connected', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            isConnected: false,
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Disconnected')).toBeInTheDocument();
    });

    it('displays connection error when present', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            connectionError: 'Connection failed',
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Connection failed')).toBeInTheDocument();
    });

    it('shows welcome message when no messages', () => {
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Ready to Help')).toBeInTheDocument();
        expect(screen.getByText(/I'm your AI Government Consultant/)).toBeInTheDocument();
    });

    it('shows loading state when fetching messages', () => {
        mockUseConsultationMessages.mockReturnValue({
            data: [],
            isLoading: true,
            error: null,
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Loading conversation...')).toBeInTheDocument();
    });

    it('handles message sending', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const input = screen.getByPlaceholderText('Ask about policy...');
        const sendButton = screen.getByRole('button', { name: /send/i });

        await user.type(input, 'Test message');
        await user.click(sendButton);

        expect(mockSendMessage.mutateAsync).toHaveBeenCalledWith({
            sessionId: 'session-1',
            content: 'Test message',
            inputMethod: 'text',
        });
    });

    it('handles Enter key to send message', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const input = screen.getByPlaceholderText('Ask about policy...');

        await user.type(input, 'Test message');
        await user.keyboard('{Enter}');

        expect(mockSendMessage.mutateAsync).toHaveBeenCalledWith({
            sessionId: 'session-1',
            content: 'Test message',
            inputMethod: 'text',
        });
    });

    it('handles Shift+Enter for new line', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const input = screen.getByPlaceholderText('Ask about policy...');

        await user.type(input, 'Line 1');
        await user.keyboard('{Shift>}{Enter}{/Shift}');
        await user.type(input, 'Line 2');

        expect(mockSendMessage.mutateAsync).not.toHaveBeenCalled();
    });

    it('disables input when not connected', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            isConnected: false,
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const input = screen.getByPlaceholderText('Ask about policy...');
        const sendButton = screen.getByRole('button', { name: /send/i });

        expect(input).toBeDisabled();
        expect(sendButton).toBeDisabled();
    });

    it('shows loading state when sending message', () => {
        mockUseSendMessage.mockReturnValue({
            ...mockSendMessage,
            isPending: true,
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const sendButton = screen.getByRole('button', { name: /send/i });
        expect(sendButton).toBeDisabled();
    });

    it('toggles voice panel', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const voiceButton = screen.getByRole('button', { name: /mic/i });
        await user.click(voiceButton);

        expect(mockStoreState.setVoicePanelOpen).toHaveBeenCalledWith(true);
    });

    it('toggles voice settings', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const voiceSettingsButton = screen.getByRole('button', { name: /volume/i });
        await user.click(voiceSettingsButton);

        expect(mockStoreState.setVoiceSettings).toHaveBeenCalledWith({
            autoPlayResponses: true,
        });
    });

    it('shows voice panel when open', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            voicePanelOpen: true,
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('Voice Input')).toBeInTheDocument();
    });

    it('prevents sending empty messages', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const sendButton = screen.getByRole('button', { name: /send/i });
        await user.click(sendButton);

        expect(mockSendMessage.mutateAsync).not.toHaveBeenCalled();
    });

    it('trims whitespace from messages', async () => {
        const user = userEvent.setup();
        renderWithQueryClient(<ChatInterface session={mockSession} />);

        const input = screen.getByPlaceholderText('Ask about policy...');

        await user.type(input, '  Test message  ');
        await user.keyboard('{Enter}');

        expect(mockSendMessage.mutateAsync).toHaveBeenCalledWith({
            sessionId: 'session-1',
            content: 'Test message',
            inputMethod: 'text',
        });
    });

    it('shows typing indicator when AI is typing', () => {
        mockUseConsultationStore.mockReturnValue({
            ...mockStoreState,
            isTyping: true,
            currentMessages: [
                {
                    id: 'msg-1',
                    sessionId: 'session-1',
                    type: 'user',
                    content: 'Hello',
                    timestamp: new Date(),
                    inputMethod: 'text',
                },
            ],
        } as any);

        renderWithQueryClient(<ChatInterface session={mockSession} />);

        expect(screen.getByText('AI is thinking...')).toBeInTheDocument();
    });
});