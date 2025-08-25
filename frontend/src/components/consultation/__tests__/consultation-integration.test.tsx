import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import ConsultationPage from '@/app/consultation/page';
import { useConsultationStore } from '@/stores/consultations';
import {
    useCreateConsultationSession,
    useSendMessage,
    useConsultationMessages,
    useUpdateConsultationSession,
    useDeleteConsultationSession,
    useExportConsultationSession
} from '@/hooks/useConsultations';

// Mock Next.js router
jest.mock('next/navigation', () => ({
    useRouter: jest.fn(),
}));

// Mock the hooks
jest.mock('@/stores/consultations');
jest.mock('@/hooks/useConsultations');

const mockUseRouter = useRouter as jest.MockedFunction<typeof useRouter>;
const mockUseConsultationStore = useConsultationStore as jest.MockedFunction<typeof useConsultationStore>;
const mockUseCreateConsultationSession = useCreateConsultationSession as jest.MockedFunction<typeof useCreateConsultationSession>;
const mockUseSendMessage = useSendMessage as jest.MockedFunction<typeof useSendMessage>;

// Mock additional hooks
jest.mock('@/hooks/useConsultations', () => ({
    useCreateConsultationSession: jest.fn(),
    useSendMessage: jest.fn(),
    useConsultationMessages: jest.fn(),
    useUpdateConsultationSession: jest.fn(),
    useDeleteConsultationSession: jest.fn(),
    useExportConsultationSession: jest.fn(),
}));

const mockRouter = {
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    prefetch: jest.fn(),
};

const mockSession = {
    id: 'session-1',
    title: 'Policy Analysis Session',
    type: 'policy' as const,
    status: 'active' as const,
    createdAt: new Date(),
    updatedAt: new Date(),
    userId: 'user-1',
    messageCount: 0,
    hasUnread: false,
    tags: [],
};

const mockCreateSession = {
    mutateAsync: jest.fn(),
    isPending: false,
    isError: false,
    error: null,
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

describe('Consultation Integration', () => {
    beforeEach(() => {
        jest.clearAllMocks();

        mockUseRouter.mockReturnValue(mockRouter as any);
        mockUseCreateConsultationSession.mockReturnValue(mockCreateSession as any);
        mockUseSendMessage.mockReturnValue(mockSendMessage as any);

        // Mock additional hooks
        (useConsultationMessages as jest.Mock).mockReturnValue({
            data: [],
            isLoading: false,
            error: null,
        });

        (useUpdateConsultationSession as jest.Mock).mockReturnValue({
            mutateAsync: jest.fn(),
            isPending: false,
        });

        (useDeleteConsultationSession as jest.Mock).mockReturnValue({
            mutateAsync: jest.fn(),
            isPending: false,
        });

        (useExportConsultationSession as jest.Mock).mockReturnValue({
            mutateAsync: jest.fn(),
            isPending: false,
        });
    });

    describe('No Active Session', () => {
        beforeEach(() => {
            mockUseConsultationStore.mockReturnValue({
                currentSession: null,
                setCurrentSession: jest.fn(),
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
            } as any);
        });

        it('shows consultation type selector initially', () => {
            renderWithQueryClient(<ConsultationPage />);

            expect(screen.getByText('New Consultation')).toBeInTheDocument();
            expect(screen.getByText('Choose Consultation Type')).toBeInTheDocument();
            expect(screen.getByText('Policy Analysis')).toBeInTheDocument();
        });

        it('creates session with quick start', async () => {
            const user = userEvent.setup();
            mockCreateSession.mutateAsync.mockResolvedValue(mockSession);

            renderWithQueryClient(<ConsultationPage />);

            const quickStartButtons = screen.getAllByText('Quick Start');
            await user.click(quickStartButtons[0]);

            expect(mockCreateSession.mutateAsync).toHaveBeenCalledWith({
                type: 'policy',
                title: 'Policy Consultation',
                context: undefined,
                priority: 'medium',
            });
        });

        it('creates session with detailed form', async () => {
            const user = userEvent.setup();
            mockCreateSession.mutateAsync.mockResolvedValue(mockSession);

            renderWithQueryClient(<ConsultationPage />);

            // Select policy type
            const policyCard = screen.getByText('Policy Analysis').closest('div');
            if (policyCard) {
                await user.click(policyCard);
            }

            await waitFor(() => {
                expect(screen.getByText('Consultation Details')).toBeInTheDocument();
            });

            // Fill form
            const titleInput = screen.getByLabelText('Session Title');
            const contextTextarea = screen.getByLabelText(/Context & Background/);

            await user.clear(titleInput);
            await user.type(titleInput, 'Custom Policy Session');
            await user.type(contextTextarea, 'Need help with healthcare policy');

            // Submit
            const startButton = screen.getByText('Start Consultation');
            await user.click(startButton);

            expect(mockCreateSession.mutateAsync).toHaveBeenCalledWith({
                type: 'policy',
                title: 'Custom Policy Session',
                context: 'Need help with healthcare policy',
                priority: 'medium',
            });
        });
    });

    describe('Active Session', () => {
        beforeEach(() => {
            mockUseConsultationStore.mockReturnValue({
                currentSession: mockSession,
                setCurrentSession: jest.fn(),
                currentMessages: [
                    {
                        id: 'msg-1',
                        sessionId: 'session-1',
                        type: 'user',
                        content: 'Hello, I need help with policy analysis',
                        timestamp: new Date(),
                        inputMethod: 'text',
                    },
                    {
                        id: 'msg-2',
                        sessionId: 'session-1',
                        type: 'assistant',
                        content: 'I\'d be happy to help you with policy analysis. What specific area would you like to focus on?',
                        timestamp: new Date(),
                        inputMethod: 'text',
                        confidence: 0.95,
                        sources: [
                            {
                                id: 'doc-1',
                                title: 'Policy Analysis Guidelines',
                                type: 'document',
                                excerpt: 'Best practices for conducting policy analysis...',
                                confidence: 0.9,
                            },
                        ],
                    },
                ],
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
            } as any);
        });

        it('shows chat interface with messages', () => {
            renderWithQueryClient(<ConsultationPage />);

            expect(screen.getByText('Policy Analysis Session')).toBeInTheDocument();
            expect(screen.getByText('Hello, I need help with policy analysis')).toBeInTheDocument();
            expect(screen.getByText(/I'd be happy to help you with policy analysis/)).toBeInTheDocument();
        });

        it('sends new messages', async () => {
            const user = userEvent.setup();
            mockSendMessage.mutateAsync.mockResolvedValue({
                id: 'msg-3',
                sessionId: 'session-1',
                type: 'user',
                content: 'I need help with healthcare policy',
                timestamp: new Date(),
                inputMethod: 'text',
            });

            renderWithQueryClient(<ConsultationPage />);

            const input = screen.getByPlaceholderText('Ask about policy...');
            const sendButton = screen.getByRole('button', { name: /send/i });

            await user.type(input, 'I need help with healthcare policy');
            await user.click(sendButton);

            expect(mockSendMessage.mutateAsync).toHaveBeenCalledWith({
                sessionId: 'session-1',
                content: 'I need help with healthcare policy',
                inputMethod: 'text',
            });
        });

        it('shows message sources and confidence', () => {
            renderWithQueryClient(<ConsultationPage />);

            expect(screen.getByText('95%')).toBeInTheDocument(); // Confidence badge
            expect(screen.getByText('1 source')).toBeInTheDocument(); // Sources button
        });

        it('expands message sources when clicked', async () => {
            const user = userEvent.setup();
            renderWithQueryClient(<ConsultationPage />);

            const sourcesButton = screen.getByText('1 source');
            await user.click(sourcesButton);

            expect(screen.getByText('Policy Analysis Guidelines')).toBeInTheDocument();
            expect(screen.getByText(/Best practices for conducting policy analysis/)).toBeInTheDocument();
        });

        it('handles voice panel toggle', async () => {
            const user = userEvent.setup();
            const setVoicePanelOpen = jest.fn();

            mockUseConsultationStore.mockReturnValue({
                currentSession: mockSession,
                setCurrentSession: jest.fn(),
                currentMessages: [],
                isTyping: false,
                isConnected: true,
                connectionError: null,
                voicePanelOpen: false,
                setVoicePanelOpen,
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
            } as any);

            renderWithQueryClient(<ConsultationPage />);

            const voiceButton = screen.getByRole('button', { name: /mic/i });
            await user.click(voiceButton);

            expect(setVoicePanelOpen).toHaveBeenCalledWith(true);
        });

        it('shows typing indicator when AI is responding', () => {
            mockUseConsultationStore.mockReturnValue({
                currentSession: mockSession,
                setCurrentSession: jest.fn(),
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
                isTyping: true,
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
            } as any);

            renderWithQueryClient(<ConsultationPage />);

            expect(screen.getByText('AI is thinking...')).toBeInTheDocument();
        });
    });

    describe('Error Handling', () => {
        it('handles session creation errors', async () => {
            const user = userEvent.setup();
            const consoleError = jest.spyOn(console, 'error').mockImplementation(() => { });

            mockUseConsultationStore.mockReturnValue({
                currentSession: null,
                setCurrentSession: jest.fn(),
            } as any);

            mockCreateSession.mutateAsync.mockRejectedValue(new Error('Failed to create session'));

            renderWithQueryClient(<ConsultationPage />);

            const quickStartButtons = screen.getAllByText('Quick Start');
            await user.click(quickStartButtons[0]);

            await waitFor(() => {
                expect(consoleError).toHaveBeenCalledWith('Failed to create consultation session:', expect.any(Error));
            });

            consoleError.mockRestore();
        });

        it('handles message sending errors', async () => {
            const user = userEvent.setup();
            const consoleError = jest.spyOn(console, 'error').mockImplementation(() => { });

            mockUseConsultationStore.mockReturnValue({
                currentSession: mockSession,
                setCurrentSession: jest.fn(),
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
            } as any);

            mockSendMessage.mutateAsync.mockRejectedValue(new Error('Failed to send message'));

            renderWithQueryClient(<ConsultationPage />);

            const input = screen.getByPlaceholderText('Ask about policy...');
            await user.type(input, 'Test message');
            await user.keyboard('{Enter}');

            await waitFor(() => {
                expect(consoleError).toHaveBeenCalledWith('Failed to send message:', expect.any(Error));
            });

            consoleError.mockRestore();
        });
    });
});