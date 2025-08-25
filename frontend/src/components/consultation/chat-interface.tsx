'use client';

import { useState, useRef, useEffect, useCallback } from 'react';

import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { MessageInput } from './message-input';
import { MessageList } from './message-list';
import { VoicePanel } from './voice-panel';
import { AttachmentPanel } from './attachment-panel';
import { TypingIndicator } from '@/components/ui/typing-indicator';
import { MessageQueueStatusCompact } from '@/components/ui/message-queue-status';
import {
    Mic,
    MicOff,
    Paperclip,
    Settings,
    Volume2,
    VolumeX,
    Maximize2,
    Minimize2
} from 'lucide-react';
import { useConsultationStore } from '@/stores/consultations';
import { useConsultationMessages, useSendMessage } from '@/hooks/useConsultations';
import { useChatWebSocket } from '@/hooks/useWebSocket';
import type { ConsultationSession } from '@/stores/consultations';
import type { QueuedMessage } from '@/components/ui/message-queue-status';

interface ChatInterfaceProps {
    session: ConsultationSession;
}

export function ChatInterface({ session }: ChatInterfaceProps) {
    const [message, setMessage] = useState('');
    const [isExpanded, setIsExpanded] = useState(false);
    const [showAttachments, setShowAttachments] = useState(false);
    const [isTyping, setIsTyping] = useState(false);
    const [queuedMessages, setQueuedMessages] = useState<QueuedMessage[]>([]);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);

    const {
        currentMessages,
        voicePanelOpen,
        setVoicePanelOpen,
        isRecording,
        voiceSettings,
        setVoiceSettings,
        addMessage,
        updateMessage,
    } = useConsultationStore();

    const { data: messagesData, isLoading: messagesLoading } = useConsultationMessages(session.id);
    const sendMessage = useSendMessage();

    // WebSocket connection for real-time chat
    const {
        isConnected,
        status: wsStatus,
        sendMessage: sendWebSocketMessage,
        sendTypingIndicator,
        joinConsultation,
        leaveConsultation,
    } = useChatWebSocket({
        consultationId: session.id,
        autoConnect: true,
        onTypingStart: () => setIsTyping(true),
        onTypingStop: () => setIsTyping(false),
        onMessageReceived: useCallback((message: any) => {
            // Add received message to store
            addMessage({
                id: message.id,
                content: message.content,
                role: message.role || 'assistant',
                timestamp: new Date(message.timestamp),
                sessionId: session.id,
            });
        }, [addMessage, session.id]),
        onMessageDelivered: useCallback((messageId: string) => {
            // Update message status to delivered
            updateMessage(messageId, { status: 'delivered' });
        }, [updateMessage]),
        onMessageRead: useCallback((messageId: string) => {
            // Update message status to read
            updateMessage(messageId, { status: 'read' });
        }, [updateMessage]),
    });

    // Auto-scroll to bottom when new messages arrive
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [currentMessages, isTyping]);

    // Load messages into store when data changes
    useEffect(() => {
        if (messagesData && messagesData.length > 0) {
            // This would typically be handled by the store/hook integration
            // For now, we'll use the currentMessages from the store
        }
    }, [messagesData]);

    // Join consultation when component mounts
    useEffect(() => {
        if (isConnected && session.id) {
            joinConsultation(session.id);
        }

        return () => {
            if (session.id) {
                leaveConsultation(session.id);
            }
            if (typingTimeoutRef.current) {
                clearTimeout(typingTimeoutRef.current);
            }
        };
    }, [isConnected, session.id, joinConsultation, leaveConsultation]);

    const handleSendMessage = async (content: string, inputMethod: 'text' | 'voice' = 'text') => {
        if (!content.trim()) return;

        const messageId = `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        const messageContent = content.trim();

        // Add message to local state immediately (optimistic update)
        const userMessage = {
            id: messageId,
            content: messageContent,
            role: 'user' as const,
            timestamp: new Date(),
            sessionId: session.id,
            status: 'sending' as const,
        };
        addMessage(userMessage);

        try {
            if (isConnected) {
                // Send via WebSocket for real-time delivery
                sendWebSocketMessage(messageContent, 'user');

                // Also send via HTTP API for persistence
                await sendMessage.mutateAsync({
                    sessionId: session.id,
                    content: messageContent,
                    inputMethod,
                });

                // Update message status to sent
                updateMessage(messageId, { status: 'sent' });
            } else {
                // Queue message for later if not connected
                const queuedMessage: QueuedMessage = {
                    id: messageId,
                    content: messageContent,
                    timestamp: Date.now(),
                    status: 'pending',
                };
                setQueuedMessages(prev => [...prev, queuedMessage]);

                // Try to send via HTTP API
                await sendMessage.mutateAsync({
                    sessionId: session.id,
                    content: messageContent,
                    inputMethod,
                });

                // Remove from queue and update status
                setQueuedMessages(prev => prev.filter(m => m.id !== messageId));
                updateMessage(messageId, { status: 'sent' });
            }

            setMessage('');
        } catch (error) {
            console.error('Failed to send message:', error);

            // Update message status to failed
            updateMessage(messageId, { status: 'failed' });

            // Add to queue for retry if not already there
            if (isConnected) {
                const queuedMessage: QueuedMessage = {
                    id: messageId,
                    content: messageContent,
                    timestamp: Date.now(),
                    status: 'failed',
                };
                setQueuedMessages(prev => [...prev, queuedMessage]);
            }
        }
    };

    const handleKeyPress = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSendMessage(message);
        }
    };

    const handleMessageChange = (value: string) => {
        setMessage(value);

        // Send typing indicator
        if (isConnected) {
            sendTypingIndicator(value.length > 0);

            // Clear previous timeout
            if (typingTimeoutRef.current) {
                clearTimeout(typingTimeoutRef.current);
            }

            // Stop typing indicator after 3 seconds of inactivity
            if (value.length > 0) {
                typingTimeoutRef.current = setTimeout(() => {
                    sendTypingIndicator(false);
                }, 3000);
            }
        }
    };

    const handleRetryMessage = (messageId: string) => {
        const queuedMessage = queuedMessages.find(m => m.id === messageId);
        if (queuedMessage) {
            handleSendMessage(queuedMessage.content);
            setQueuedMessages(prev => prev.filter(m => m.id !== messageId));
        }
    };

    const handleRetryAllMessages = () => {
        const failedMessages = queuedMessages.filter(m => m.status === 'failed');
        failedMessages.forEach(msg => {
            handleSendMessage(msg.content);
        });
        setQueuedMessages(prev => prev.filter(m => m.status !== 'failed'));
    };

    const handleClearQueue = () => {
        setQueuedMessages([]);
    };

    const toggleVoiceSettings = () => {
        setVoiceSettings({
            autoPlayResponses: !voiceSettings.autoPlayResponses,
        });
    };

    return (
        <div className={`flex flex-col h-full ${isExpanded ? 'fixed inset-0 z-50 bg-background' : ''}`}>
            {/* Chat Header */}
            <div className="flex items-center justify-between p-4 border-b bg-background/95 backdrop-blur">
                <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2">
                        <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
                        <span className="text-sm text-muted-foreground">
                            {isConnected ? 'Connected' : 'Disconnected'}
                        </span>
                    </div>

                    {queuedMessages.length > 0 && (
                        <MessageQueueStatusCompact
                            queuedMessages={queuedMessages}
                            onRetryAll={handleRetryAllMessages}
                        />
                    )}
                </div>

                <div className="flex items-center gap-2">
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowAttachments(!showAttachments)}
                        className={showAttachments ? 'bg-accent' : ''}
                    >
                        <Paperclip className="h-4 w-4" />
                    </Button>

                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setVoicePanelOpen(!voicePanelOpen)}
                        className={voicePanelOpen ? 'bg-accent' : ''}
                    >
                        {isRecording ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4" />}
                    </Button>

                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={toggleVoiceSettings}
                        className={voiceSettings.autoPlayResponses ? 'bg-accent' : ''}
                    >
                        {voiceSettings.autoPlayResponses ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
                    </Button>

                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setIsExpanded(!isExpanded)}
                    >
                        {isExpanded ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
                    </Button>
                </div>
            </div>

            {/* Voice Panel */}
            {voicePanelOpen && (
                <VoicePanel
                    onVoiceMessage={(content) => handleSendMessage(content, 'voice')}
                    onClose={() => setVoicePanelOpen(false)}
                />
            )}

            {/* Attachment Panel */}
            {showAttachments && (
                <AttachmentPanel
                    sessionId={session.id}
                    onClose={() => setShowAttachments(false)}
                />
            )}

            {/* Messages Area */}
            <div className="flex-1 flex flex-col min-h-0">
                <div className="flex-1 overflow-y-auto">
                    {messagesLoading ? (
                        <div className="flex items-center justify-center h-full">
                            <div className="text-center">
                                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                                <p className="text-muted-foreground">Loading conversation...</p>
                            </div>
                        </div>
                    ) : currentMessages.length === 0 ? (
                        <div className="flex items-center justify-center h-full">
                            <div className="text-center max-w-md mx-auto p-6">
                                <div className="mb-4">
                                    <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-4">
                                        <Settings className="h-8 w-8 text-primary" />
                                    </div>
                                    <h3 className="text-lg font-semibold mb-2">Ready to Help</h3>
                                    <p className="text-muted-foreground mb-4">
                                        I&apos;m your AI Government Consultant. Ask me anything about {session.type} matters,
                                        and I&apos;ll provide expert guidance based on current policies and best practices.
                                    </p>
                                </div>

                                <div className="space-y-2 text-sm text-muted-foreground">
                                    <p>💡 <strong>Tip:</strong> Be specific about your situation for better advice</p>
                                    <p>🎤 <strong>Voice:</strong> Click the microphone to speak your question</p>
                                    <p>📎 <strong>Files:</strong> Attach relevant documents for context</p>
                                </div>
                            </div>
                        </div>
                    ) : (
                        <>
                            <MessageList
                                messages={currentMessages}
                                isTyping={false}
                                sessionId={session.id}
                            />
                            <TypingIndicator
                                isVisible={isTyping}
                                userName="AI Assistant"
                                className="px-4 py-2"
                            />
                        </>
                    )}
                    <div ref={messagesEndRef} />
                </div>

                {/* Message Input */}
                <div className="border-t bg-background/95 backdrop-blur">
                    <MessageInput
                        value={message}
                        onChange={handleMessageChange}
                        onSend={(content) => handleSendMessage(content)}
                        onKeyPress={handleKeyPress}
                        disabled={sendMessage.isPending}
                        isLoading={sendMessage.isPending}
                        placeholder={`Ask about ${session.type}...`}
                    />
                </div>
            </div>
        </div>
    );
}