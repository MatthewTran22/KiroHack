"use client";

import React from 'react';
import { cn } from '@/lib/utils';

interface TypingIndicatorProps {
    isVisible: boolean;
    userName?: string;
    className?: string;
}

export function TypingIndicator({
    isVisible,
    userName = 'AI Assistant',
    className
}: TypingIndicatorProps) {
    if (!isVisible) return null;

    return (
        <div className={cn(
            'flex items-center gap-2 px-4 py-2 text-sm text-muted-foreground animate-in fade-in-0 slide-in-from-bottom-2',
            className
        )}>
            <div className="flex items-center gap-1">
                <div className="flex gap-1">
                    <div className="w-2 h-2 bg-muted-foreground/60 rounded-full animate-bounce [animation-delay:-0.3s]" />
                    <div className="w-2 h-2 bg-muted-foreground/60 rounded-full animate-bounce [animation-delay:-0.15s]" />
                    <div className="w-2 h-2 bg-muted-foreground/60 rounded-full animate-bounce" />
                </div>
                <span className="ml-2">
                    {userName} is typing...
                </span>
            </div>
        </div>
    );
}

// Compact version for inline use
export function TypingIndicatorCompact({
    isVisible,
    className
}: Omit<TypingIndicatorProps, 'userName'>) {
    if (!isVisible) return null;

    return (
        <div className={cn(
            'flex items-center gap-1 animate-in fade-in-0',
            className
        )}>
            <div className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce [animation-delay:-0.3s]" />
            <div className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce [animation-delay:-0.15s]" />
            <div className="w-1.5 h-1.5 bg-muted-foreground/60 rounded-full animate-bounce" />
        </div>
    );
}