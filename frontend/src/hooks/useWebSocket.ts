import { useEffect, useRef, useCallback, useState } from 'react';
import {
    getWebSocketClient,
    WebSocketClient,
    WebSocketMessage,
    WebSocketStatus,
    WebSocketEventHandler,
    WebSocketStatusHandler
} from '@/lib/websocket';

export interface UseWebSocketOptions {
    autoConnect?: boolean;
    onConnect?: () => void;
    onDisconnect?: () => void;
    onError?: (error: Event) => void;
    onMessage?: (message: WebSocketMessage) => void;
}

export interface UseWebSocketReturn {
    isConnected: boolean;
    status: WebSocketStatus;
    connect: () => void;
    disconnect: () => void;
    send: (message: Omit<WebSocketMessage, 'timestamp'>) => void;
    subscribe: (eventType: string, handler: WebSocketEventHandler) => () => void;
}

export function useWebSocket(options: UseWebSocketOptions = {}): UseWebSocketReturn {
    const {
        autoConnect = true,
        onConnect,
        onDisconnect,
        onError,
        onMessage,
    } = options;

    const [status, setStatus] = useState<WebSocketStatus>(WebSocketStatus.DISCONNECTED);
    const [isConnected, setIsConnected] = useState(false);

    const clientRef = useRef<WebSocketClient | null>(null);
    const handlersRef = useRef<Map<string, WebSocketEventHandler>>(new Map());

    // Initialize WebSocket client
    useEffect(() => {
        clientRef.current = getWebSocketClient();

        const statusHandler: WebSocketStatusHandler = (newStatus) => {
            setStatus(newStatus);
            setIsConnected(newStatus === WebSocketStatus.CONNECTED);

            if (newStatus === WebSocketStatus.CONNECTED && onConnect) {
                onConnect();
            } else if (newStatus === WebSocketStatus.DISCONNECTED && onDisconnect) {
                onDisconnect();
            } else if (newStatus === WebSocketStatus.ERROR && onError) {
                onError(new Event('websocket-error'));
            }
        };

        clientRef.current.onStatusChange(statusHandler);

        // Set up message handler if provided
        if (onMessage) {
            clientRef.current.on('message', onMessage);
        }

        // Auto-connect if enabled
        if (autoConnect) {
            clientRef.current.connect();
        }

        return () => {
            if (clientRef.current) {
                clientRef.current.offStatusChange(statusHandler);
                if (onMessage) {
                    clientRef.current.off('message', onMessage);
                }
            }
        };
    }, [autoConnect, onConnect, onDisconnect, onError, onMessage]);

    const connect = useCallback(() => {
        clientRef.current?.connect();
    }, []);

    const disconnect = useCallback(() => {
        clientRef.current?.disconnect();
    }, []);

    const send = useCallback((message: Omit<WebSocketMessage, 'timestamp'>) => {
        clientRef.current?.send(message);
    }, []);

    const subscribe = useCallback((eventType: string, handler: WebSocketEventHandler) => {
        if (!clientRef.current) {
            return () => { };
        }

        clientRef.current.on(eventType, handler);
        handlersRef.current.set(`${eventType}_${Date.now()}`, handler);

        return () => {
            if (clientRef.current) {
                clientRef.current.off(eventType, handler);
            }
        };
    }, []);

    return {
        isConnected,
        status,
        connect,
        disconnect,
        send,
        subscribe,
    };
}

// Hook for chat-specific WebSocket functionality
export interface UseChatWebSocketOptions extends UseWebSocketOptions {
    consultationId?: string;
    onTypingStart?: (userId: string) => void;
    onTypingStop?: (userId: string) => void;
    onMessageReceived?: (message: any) => void;
    onMessageDelivered?: (messageId: string) => void;
    onMessageRead?: (messageId: string) => void;
}

export interface UseChatWebSocketReturn extends UseWebSocketReturn {
    sendMessage: (content: string, type?: string) => void;
    sendTypingIndicator: (isTyping: boolean) => void;
    markMessageAsRead: (messageId: string) => void;
    joinConsultation: (consultationId: string) => void;
    leaveConsultation: (consultationId: string) => void;
}

export function useChatWebSocket(options: UseChatWebSocketOptions = {}): UseChatWebSocketReturn {
    const {
        consultationId,
        onTypingStart,
        onTypingStop,
        onMessageReceived,
        onMessageDelivered,
        onMessageRead,
        ...wsOptions
    } = options;

    const webSocket = useWebSocket(wsOptions);
    const currentConsultationRef = useRef<string | null>(null);

    // Set up chat-specific event handlers
    useEffect(() => {
        if (!webSocket.isConnected) return;

        const unsubscribers: (() => void)[] = [];

        // Handle typing indicators
        if (onTypingStart) {
            unsubscribers.push(
                webSocket.subscribe('typing_start', (message) => {
                    onTypingStart(message.data.userId);
                })
            );
        }

        if (onTypingStop) {
            unsubscribers.push(
                webSocket.subscribe('typing_stop', (message) => {
                    onTypingStop(message.data.userId);
                })
            );
        }

        // Handle chat messages
        if (onMessageReceived) {
            unsubscribers.push(
                webSocket.subscribe('chat_message', (message) => {
                    onMessageReceived(message.data);
                })
            );
        }

        // Handle message delivery confirmations
        if (onMessageDelivered) {
            unsubscribers.push(
                webSocket.subscribe('message_delivered', (message) => {
                    onMessageDelivered(message.data.messageId);
                })
            );
        }

        // Handle message read confirmations
        if (onMessageRead) {
            unsubscribers.push(
                webSocket.subscribe('message_read', (message) => {
                    onMessageRead(message.data.messageId);
                })
            );
        }

        return () => {
            unsubscribers.forEach(unsubscribe => unsubscribe());
        };
    }, [webSocket.isConnected, webSocket.subscribe, onTypingStart, onTypingStop, onMessageReceived, onMessageDelivered, onMessageRead]);

    // Auto-join consultation when connected and consultationId is provided
    useEffect(() => {
        if (webSocket.isConnected && consultationId && consultationId !== currentConsultationRef.current) {
            joinConsultation(consultationId);
        }
    }, [webSocket.isConnected, consultationId]);

    const sendMessage = useCallback((content: string, type: string = 'user') => {
        if (!currentConsultationRef.current) {
            console.warn('Cannot send message: not joined to any consultation');
            return;
        }

        webSocket.send({
            type: 'chat_message',
            data: {
                consultationId: currentConsultationRef.current,
                content,
                messageType: type,
            },
        });
    }, [webSocket.send]);

    const sendTypingIndicator = useCallback((isTyping: boolean) => {
        if (!currentConsultationRef.current) return;

        webSocket.send({
            type: isTyping ? 'typing_start' : 'typing_stop',
            data: {
                consultationId: currentConsultationRef.current,
            },
        });
    }, [webSocket.send]);

    const markMessageAsRead = useCallback((messageId: string) => {
        webSocket.send({
            type: 'mark_message_read',
            data: {
                messageId,
                consultationId: currentConsultationRef.current,
            },
        });
    }, [webSocket.send]);

    const joinConsultation = useCallback((consultationId: string) => {
        webSocket.send({
            type: 'join_consultation',
            data: {
                consultationId,
            },
        });
        currentConsultationRef.current = consultationId;
    }, [webSocket.send]);

    const leaveConsultation = useCallback((consultationId: string) => {
        webSocket.send({
            type: 'leave_consultation',
            data: {
                consultationId,
            },
        });
        if (currentConsultationRef.current === consultationId) {
            currentConsultationRef.current = null;
        }
    }, [webSocket.send]);

    return {
        ...webSocket,
        sendMessage,
        sendTypingIndicator,
        markMessageAsRead,
        joinConsultation,
        leaveConsultation,
    };
}

// Hook for real-time notifications
export interface UseNotificationWebSocketOptions extends UseWebSocketOptions {
    onNotification?: (notification: any) => void;
    onResearchUpdate?: (update: any) => void;
    onPolicyUpdate?: (update: any) => void;
}

export function useNotificationWebSocket(options: UseNotificationWebSocketOptions = {}): UseWebSocketReturn {
    const {
        onNotification,
        onResearchUpdate,
        onPolicyUpdate,
        ...wsOptions
    } = options;

    const webSocket = useWebSocket(wsOptions);

    // Set up notification-specific event handlers
    useEffect(() => {
        if (!webSocket.isConnected) return;

        const unsubscribers: (() => void)[] = [];

        if (onNotification) {
            unsubscribers.push(
                webSocket.subscribe('notification', (message) => {
                    onNotification(message.data);
                })
            );
        }

        if (onResearchUpdate) {
            unsubscribers.push(
                webSocket.subscribe('research_update', (message) => {
                    onResearchUpdate(message.data);
                })
            );
        }

        if (onPolicyUpdate) {
            unsubscribers.push(
                webSocket.subscribe('policy_update', (message) => {
                    onPolicyUpdate(message.data);
                })
            );
        }

        return () => {
            unsubscribers.forEach(unsubscribe => unsubscribe());
        };
    }, [webSocket.isConnected, webSocket.subscribe, onNotification, onResearchUpdate, onPolicyUpdate]);

    return webSocket;
}