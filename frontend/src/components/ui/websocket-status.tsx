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
    Wifi,
    WifiOff,
    Loader2,
    AlertTriangle,
    RefreshCw
} from 'lucide-react';
import { WebSocketStatus } from '@/lib/websocket';
import { cn } from '@/lib/utils';

interface WebSocketStatusProps {
    status: WebSocketStatus;
    onReconnect?: () => void;
    className?: string;
    showText?: boolean;
    size?: 'sm' | 'md' | 'lg';
}

export function WebSocketStatusIndicator({
    status,
    onReconnect,
    className,
    showText = false,
    size = 'md'
}: WebSocketStatusProps) {
    const getStatusConfig = () => {
        switch (status) {
            case WebSocketStatus.CONNECTED:
                return {
                    icon: Wifi,
                    text: 'Connected',
                    variant: 'default' as const,
                    color: 'text-green-600',
                    bgColor: 'bg-green-100',
                    description: 'Real-time connection is active',
                };
            case WebSocketStatus.CONNECTING:
                return {
                    icon: Loader2,
                    text: 'Connecting',
                    variant: 'secondary' as const,
                    color: 'text-blue-600',
                    bgColor: 'bg-blue-100',
                    description: 'Establishing connection...',
                    animate: true,
                };
            case WebSocketStatus.RECONNECTING:
                return {
                    icon: RefreshCw,
                    text: 'Reconnecting',
                    variant: 'secondary' as const,
                    color: 'text-yellow-600',
                    bgColor: 'bg-yellow-100',
                    description: 'Attempting to reconnect...',
                    animate: true,
                };
            case WebSocketStatus.ERROR:
                return {
                    icon: AlertTriangle,
                    text: 'Error',
                    variant: 'destructive' as const,
                    color: 'text-red-600',
                    bgColor: 'bg-red-100',
                    description: 'Connection error occurred',
                };
            case WebSocketStatus.DISCONNECTED:
            default:
                return {
                    icon: WifiOff,
                    text: 'Disconnected',
                    variant: 'outline' as const,
                    color: 'text-gray-600',
                    bgColor: 'bg-gray-100',
                    description: 'No real-time connection',
                };
        }
    };

    const config = getStatusConfig();
    const Icon = config.icon;

    const iconSize = {
        sm: 'h-3 w-3',
        md: 'h-4 w-4',
        lg: 'h-5 w-5',
    }[size];

    const content = (
        <div className={cn('flex items-center gap-2', className)}>
            <Icon
                className={cn(
                    iconSize,
                    config.color,
                    config.animate && 'animate-spin'
                )}
            />
            {showText && (
                <span className={cn(
                    'text-sm font-medium',
                    config.color
                )}>
                    {config.text}
                </span>
            )}
        </div>
    );

    const badgeContent = showText ? (
        <Badge variant={config.variant} className={cn('gap-2', className)}>
            <Icon
                className={cn(
                    iconSize,
                    config.animate && 'animate-spin'
                )}
            />
            {config.text}
        </Badge>
    ) : (
        <div className={cn(
            'flex items-center justify-center rounded-full p-1.5',
            config.bgColor,
            className
        )}>
            <Icon
                className={cn(
                    iconSize,
                    config.color,
                    config.animate && 'animate-spin'
                )}
            />
        </div>
    );

    return (
        <TooltipProvider>
            <Tooltip>
                <TooltipTrigger asChild>
                    <div className="flex items-center gap-2">
                        {badgeContent}
                        {(status === WebSocketStatus.DISCONNECTED || status === WebSocketStatus.ERROR) && onReconnect && (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={onReconnect}
                                className="h-6 px-2 text-xs"
                            >
                                <RefreshCw className="h-3 w-3 mr-1" />
                                Retry
                            </Button>
                        )}
                    </div>
                </TooltipTrigger>
                <TooltipContent>
                    <p>{config.description}</p>
                    {(status === WebSocketStatus.DISCONNECTED || status === WebSocketStatus.ERROR) && (
                        <p className="text-xs text-muted-foreground mt-1">
                            Click retry to reconnect
                        </p>
                    )}
                </TooltipContent>
            </Tooltip>
        </TooltipProvider>
    );
}

// Compact version for header/status bar
export function WebSocketStatusCompact({
    status,
    onReconnect,
    className
}: Omit<WebSocketStatusProps, 'showText' | 'size'>) {
    return (
        <WebSocketStatusIndicator
            status={status}
            onReconnect={onReconnect}
            className={className}
            showText={false}
            size="sm"
        />
    );
}

// Full version with text for settings/debug
export function WebSocketStatusFull({
    status,
    onReconnect,
    className
}: Omit<WebSocketStatusProps, 'showText' | 'size'>) {
    return (
        <WebSocketStatusIndicator
            status={status}
            onReconnect={onReconnect}
            className={className}
            showText={true}
            size="md"
        />
    );
}