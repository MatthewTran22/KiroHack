"use client";

import React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from '@/components/ui/tooltip';
import {
    Clock,
    Send,
    AlertCircle,
    CheckCircle2,
    X
} from 'lucide-react';
import { cn } from '@/lib/utils';

export interface QueuedMessage {
    id: string;
    content: string;
    timestamp: number;
    status: 'pending' | 'sending' | 'sent' | 'failed';
    retryCount?: number;
}

interface MessageQueueStatusProps {
    queuedMessages: QueuedMessage[];
    onRetryMessage?: (messageId: string) => void;
    onCancelMessage?: (messageId: string) => void;
    onRetryAll?: () => void;
    onClearQueue?: () => void;
    className?: string;
}

export function MessageQueueStatus({
    queuedMessages,
    onRetryMessage,
    onCancelMessage,
    onRetryAll,
    onClearQueue,
    className
}: MessageQueueStatusProps) {
    const pendingCount = queuedMessages.filter(m => m.status === 'pending').length;
    const failedCount = queuedMessages.filter(m => m.status === 'failed').length;
    const sendingCount = queuedMessages.filter(m => m.status === 'sending').length;

    if (queuedMessages.length === 0) {
        return null;
    }

    const getStatusIcon = (status: QueuedMessage['status']) => {
        switch (status) {
            case 'pending':
                return <Clock className="h-3 w-3" />;
            case 'sending':
                return <Send className="h-3 w-3 animate-pulse" />;
            case 'sent':
                return <CheckCircle2 className="h-3 w-3 text-green-600" />;
            case 'failed':
                return <AlertCircle className="h-3 w-3 text-red-600" />;
        }
    };

    const getStatusColor = (status: QueuedMessage['status']) => {
        switch (status) {
            case 'pending':
                return 'text-yellow-600 bg-yellow-50';
            case 'sending':
                return 'text-blue-600 bg-blue-50';
            case 'sent':
                return 'text-green-600 bg-green-50';
            case 'failed':
                return 'text-red-600 bg-red-50';
        }
    };

    return (
        <div className={cn('border-t bg-muted/30 p-3', className)}>
            <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                    <Badge variant="outline" className="gap-1">
                        <Clock className="h-3 w-3" />
                        {queuedMessages.length} queued
                    </Badge>

                    {failedCount > 0 && (
                        <Badge variant="destructive" className="gap-1">
                            <AlertCircle className="h-3 w-3" />
                            {failedCount} failed
                        </Badge>
                    )}

                    {sendingCount > 0 && (
                        <Badge variant="secondary" className="gap-1">
                            <Send className="h-3 w-3 animate-pulse" />
                            {sendingCount} sending
                        </Badge>
                    )}
                </div>

                <div className="flex items-center gap-1">
                    {failedCount > 0 && onRetryAll && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={onRetryAll}
                            className="h-6 px-2 text-xs"
                        >
                            Retry All
                        </Button>
                    )}

                    {onClearQueue && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={onClearQueue}
                            className="h-6 px-2 text-xs"
                        >
                            Clear
                        </Button>
                    )}
                </div>
            </div>

            <div className="space-y-1 max-h-32 overflow-y-auto">
                {queuedMessages.map((message) => (
                    <div
                        key={message.id}
                        className={cn(
                            'flex items-center justify-between p-2 rounded text-xs',
                            getStatusColor(message.status)
                        )}
                    >
                        <div className="flex items-center gap-2 flex-1 min-w-0">
                            {getStatusIcon(message.status)}
                            <span className="truncate">
                                {message.content.length > 50
                                    ? `${message.content.substring(0, 50)}...`
                                    : message.content
                                }
                            </span>
                            {message.retryCount && message.retryCount > 0 && (
                                <Badge variant="outline" className="text-xs px-1 py-0">
                                    Retry {message.retryCount}
                                </Badge>
                            )}
                        </div>

                        <div className="flex items-center gap-1 ml-2">
                            <TooltipProvider>
                                <Tooltip>
                                    <TooltipTrigger asChild>
                                        <span className="text-xs text-muted-foreground">
                                            {new Date(message.timestamp).toLocaleTimeString()}
                                        </span>
                                    </TooltipTrigger>
                                    <TooltipContent>
                                        <p>{new Date(message.timestamp).toLocaleString()}</p>
                                    </TooltipContent>
                                </Tooltip>
                            </TooltipProvider>

                            {message.status === 'failed' && onRetryMessage && (
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => onRetryMessage(message.id)}
                                    className="h-4 w-4 p-0"
                                >
                                    <Send className="h-3 w-3" />
                                </Button>
                            )}

                            {onCancelMessage && message.status !== 'sent' && (
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => onCancelMessage(message.id)}
                                    className="h-4 w-4 p-0"
                                >
                                    <X className="h-3 w-3" />
                                </Button>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

// Compact version for minimal UI
export function MessageQueueStatusCompact({
    queuedMessages,
    onRetryAll,
    className
}: Pick<MessageQueueStatusProps, 'queuedMessages' | 'onRetryAll' | 'className'>) {
    const pendingCount = queuedMessages.filter(m => m.status === 'pending').length;
    const failedCount = queuedMessages.filter(m => m.status === 'failed').length;

    if (queuedMessages.length === 0) {
        return null;
    }

    return (
        <div className={cn('flex items-center gap-2', className)}>
            <Badge variant="outline" className="gap-1 text-xs">
                <Clock className="h-3 w-3" />
                {pendingCount}
            </Badge>

            {failedCount > 0 && (
                <>
                    <Badge variant="destructive" className="gap-1 text-xs">
                        <AlertCircle className="h-3 w-3" />
                        {failedCount}
                    </Badge>

                    {onRetryAll && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={onRetryAll}
                            className="h-5 px-1 text-xs"
                        >
                            Retry
                        </Button>
                    )}
                </>
            )}
        </div>
    );
}