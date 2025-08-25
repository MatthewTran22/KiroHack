import { tokenManager } from './auth';

export interface WebSocketMessage {
    type: string;
    data: any;
    timestamp: number;
    id?: string;
}

export interface WebSocketConfig {
    url: string;
    reconnectInterval: number;
    maxReconnectAttempts: number;
    heartbeatInterval: number;
}

export type WebSocketEventHandler = (message: WebSocketMessage) => void;
export type WebSocketStatusHandler = (status: WebSocketStatus) => void;

export enum WebSocketStatus {
    CONNECTING = 'connecting',
    CONNECTED = 'connected',
    DISCONNECTED = 'disconnected',
    RECONNECTING = 'reconnecting',
    ERROR = 'error',
}

export class WebSocketClient {
    private ws: WebSocket | null = null;
    private config: WebSocketConfig;
    private eventHandlers: Map<string, WebSocketEventHandler[]> = new Map();
    private statusHandlers: WebSocketStatusHandler[] = [];
    private reconnectAttempts = 0;
    private reconnectTimer: NodeJS.Timeout | null = null;
    private heartbeatTimer: NodeJS.Timeout | null = null;
    private status: WebSocketStatus = WebSocketStatus.DISCONNECTED;
    private messageQueue: WebSocketMessage[] = [];
    private isManualClose = false;

    constructor(config: Partial<WebSocketConfig> = {}) {
        this.config = {
            url: config.url || `${process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'}/ws`,
            reconnectInterval: config.reconnectInterval || 3000,
            maxReconnectAttempts: config.maxReconnectAttempts || 10,
            heartbeatInterval: config.heartbeatInterval || 30000,
        };
    }

    connect(): void {
        if (this.ws?.readyState === WebSocket.OPEN) {
            return;
        }

        this.isManualClose = false;
        this.setStatus(WebSocketStatus.CONNECTING);

        try {
            const token = tokenManager.getToken();
            const wsUrl = token
                ? `${this.config.url}?token=${encodeURIComponent(token)}`
                : this.config.url;

            console.log('Attempting WebSocket connection to:', wsUrl);
            this.ws = new WebSocket(wsUrl);
            this.setupEventListeners();
        } catch (error) {
            console.error('WebSocket connection error:', error);
            this.setStatus(WebSocketStatus.ERROR);
            this.scheduleReconnect();
        }
    }

    disconnect(): void {
        this.isManualClose = true;
        this.clearTimers();

        if (this.ws) {
            this.ws.close(1000, 'Manual disconnect');
            this.ws = null;
        }

        this.setStatus(WebSocketStatus.DISCONNECTED);
        this.reconnectAttempts = 0;
    }

    send(message: Omit<WebSocketMessage, 'timestamp'>): void {
        const fullMessage: WebSocketMessage = {
            ...message,
            timestamp: Date.now(),
            id: message.id || this.generateMessageId(),
        };

        if (this.ws?.readyState === WebSocket.OPEN) {
            try {
                this.ws.send(JSON.stringify(fullMessage));
            } catch (error) {
                console.error('Failed to send WebSocket message:', error);
                this.queueMessage(fullMessage);
            }
        } else {
            this.queueMessage(fullMessage);
        }
    }

    on(eventType: string, handler: WebSocketEventHandler): void {
        if (!this.eventHandlers.has(eventType)) {
            this.eventHandlers.set(eventType, []);
        }
        this.eventHandlers.get(eventType)!.push(handler);
    }

    off(eventType: string, handler: WebSocketEventHandler): void {
        const handlers = this.eventHandlers.get(eventType);
        if (handlers) {
            const index = handlers.indexOf(handler);
            if (index > -1) {
                handlers.splice(index, 1);
            }
        }
    }

    onStatusChange(handler: WebSocketStatusHandler): void {
        this.statusHandlers.push(handler);
    }

    offStatusChange(handler: WebSocketStatusHandler): void {
        const index = this.statusHandlers.indexOf(handler);
        if (index > -1) {
            this.statusHandlers.splice(index, 1);
        }
    }

    getStatus(): WebSocketStatus {
        return this.status;
    }

    isConnected(): boolean {
        return this.status === WebSocketStatus.CONNECTED;
    }

    private setupEventListeners(): void {
        if (!this.ws) return;

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.setStatus(WebSocketStatus.CONNECTED);
            this.reconnectAttempts = 0;
            this.startHeartbeat();
            this.flushMessageQueue();
        };

        this.ws.onmessage = (event) => {
            try {
                const message: WebSocketMessage = JSON.parse(event.data);
                this.handleMessage(message);
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error);
            }
        };

        this.ws.onclose = (event) => {
            console.log('WebSocket closed:', event.code, event.reason);
            console.log('Close event details:', {
                code: event.code,
                reason: event.reason,
                wasClean: event.wasClean,
                type: event.type
            });
            this.clearTimers();

            if (!this.isManualClose) {
                this.setStatus(WebSocketStatus.DISCONNECTED);
                this.scheduleReconnect();
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.setStatus(WebSocketStatus.ERROR);
        };
    }

    private handleMessage(message: WebSocketMessage): void {
        // Handle system messages
        if (message.type === 'ping') {
            this.send({ type: 'pong', data: null });
            return;
        }

        if (message.type === 'pong') {
            // Heartbeat response received
            return;
        }

        // Dispatch to event handlers
        const handlers = this.eventHandlers.get(message.type);
        if (handlers) {
            handlers.forEach(handler => {
                try {
                    handler(message);
                } catch (error) {
                    console.error('Error in WebSocket event handler:', error);
                }
            });
        }

        // Also dispatch to 'message' handlers for all messages
        const messageHandlers = this.eventHandlers.get('message');
        if (messageHandlers) {
            messageHandlers.forEach(handler => {
                try {
                    handler(message);
                } catch (error) {
                    console.error('Error in WebSocket message handler:', error);
                }
            });
        }
    }

    private setStatus(status: WebSocketStatus): void {
        if (this.status !== status) {
            this.status = status;
            this.statusHandlers.forEach(handler => {
                try {
                    handler(status);
                } catch (error) {
                    console.error('Error in WebSocket status handler:', error);
                }
            });
        }
    }

    private scheduleReconnect(): void {
        if (this.isManualClose || this.reconnectAttempts >= this.config.maxReconnectAttempts) {
            return;
        }

        this.setStatus(WebSocketStatus.RECONNECTING);
        this.reconnectAttempts++;

        const delay = Math.min(
            this.config.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1),
            30000 // Max 30 seconds
        );

        console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.config.maxReconnectAttempts})`);

        this.reconnectTimer = setTimeout(() => {
            this.connect();
        }, delay);
    }

    private startHeartbeat(): void {
        this.heartbeatTimer = setInterval(() => {
            if (this.ws?.readyState === WebSocket.OPEN) {
                this.send({ type: 'ping', data: null });
            }
        }, this.config.heartbeatInterval);
    }

    private clearTimers(): void {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }

        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }

    private queueMessage(message: WebSocketMessage): void {
        this.messageQueue.push(message);

        // Limit queue size to prevent memory issues
        if (this.messageQueue.length > 100) {
            this.messageQueue.shift();
        }
    }

    private flushMessageQueue(): void {
        while (this.messageQueue.length > 0 && this.ws?.readyState === WebSocket.OPEN) {
            const message = this.messageQueue.shift()!;
            try {
                this.ws.send(JSON.stringify(message));
            } catch (error) {
                console.error('Failed to send queued message:', error);
                // Put the message back at the front of the queue
                this.messageQueue.unshift(message);
                break;
            }
        }
    }

    private generateMessageId(): string {
        return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
}

// Singleton instance
let wsClient: WebSocketClient | null = null;

export function getWebSocketClient(): WebSocketClient {
    if (!wsClient) {
        wsClient = new WebSocketClient();
    }
    return wsClient;
}

export function createWebSocketClient(config?: Partial<WebSocketConfig>): WebSocketClient {
    return new WebSocketClient(config);
}