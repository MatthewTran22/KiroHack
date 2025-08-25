'use client';

import { useState, useRef, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import {
    Send,
    Loader2,
    Mic,
    MicOff,
    Bold,
    Italic,
    List,
    Link
} from 'lucide-react';
import { useConsultationStore } from '@/stores/consultations';

interface MessageInputProps {
    value: string;
    onChange: (value: string) => void;
    onSend: (message: string) => void;
    onKeyPress?: (e: React.KeyboardEvent) => void;
    disabled?: boolean;
    isLoading?: boolean;
    placeholder?: string;
}

export function MessageInput({
    value,
    onChange,
    onSend,
    onKeyPress,
    disabled = false,
    isLoading = false,
    placeholder = 'Type your message...',
}: MessageInputProps) {
    const [isFocused, setIsFocused] = useState(false);
    const [showFormatting, setShowFormatting] = useState(false);
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    const {
        isRecording,
        setIsRecording,
        setVoicePanelOpen
    } = useConsultationStore();

    // Auto-resize textarea
    useEffect(() => {
        if (textareaRef.current) {
            textareaRef.current.style.height = 'auto';
            textareaRef.current.style.height = `${Math.min(textareaRef.current.scrollHeight, 120)}px`;
        }
    }, [value]);

    const handleSend = () => {
        if (value.trim() && !disabled && !isLoading) {
            onSend(value);
        }
    };

    const handleVoiceToggle = () => {
        if (isRecording) {
            setIsRecording(false);
        } else {
            setVoicePanelOpen(true);
        }
    };

    const insertFormatting = (before: string, after: string = '') => {
        if (!textareaRef.current) return;

        const textarea = textareaRef.current;
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const selectedText = value.substring(start, end);

        const newValue =
            value.substring(0, start) +
            before +
            selectedText +
            after +
            value.substring(end);

        onChange(newValue);

        // Restore cursor position
        setTimeout(() => {
            if (textarea) {
                const newCursorPos = start + before.length + selectedText.length;
                textarea.setSelectionRange(newCursorPos, newCursorPos);
                textarea.focus();
            }
        }, 0);
    };

    const formatButtons = [
        { icon: Bold, action: () => insertFormatting('**', '**'), tooltip: 'Bold' },
        { icon: Italic, action: () => insertFormatting('*', '*'), tooltip: 'Italic' },
        { icon: List, action: () => insertFormatting('\n- ', ''), tooltip: 'List' },
        { icon: Link, action: () => insertFormatting('[', '](url)'), tooltip: 'Link' },
    ];

    return (
        <div className="p-4 space-y-3">
            {/* Formatting Toolbar */}
            {showFormatting && (
                <div className="flex items-center gap-1 p-2 bg-muted/50 rounded-lg">
                    <div className="flex items-center gap-1">
                        {formatButtons.map((button, index) => {
                            const Icon = button.icon;
                            return (
                                <Button
                                    key={index}
                                    variant="ghost"
                                    size="sm"
                                    onClick={button.action}
                                    className="h-7 w-7 p-0"
                                    title={button.tooltip}
                                >
                                    <Icon className="h-3 w-3" />
                                </Button>
                            );
                        })}
                    </div>
                    <div className="h-4 w-px bg-border mx-2" />
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setShowFormatting(false)}
                        className="h-7 px-2 text-xs"
                    >
                        Hide
                    </Button>
                </div>
            )}

            {/* Input Area */}
            <div className={`relative border rounded-lg transition-all ${isFocused ? 'border-primary ring-1 ring-primary/20' : 'border-border'
                } ${disabled ? 'opacity-50' : ''}`}>
                <div className="flex items-end gap-2 p-3">
                    {/* Textarea */}
                    <div className="flex-1 relative">
                        <Textarea
                            ref={textareaRef}
                            value={value}
                            onChange={(e) => onChange(e.target.value)}
                            onKeyDown={onKeyPress}
                            onFocus={() => setIsFocused(true)}
                            onBlur={() => setIsFocused(false)}
                            placeholder={placeholder}
                            disabled={disabled}
                            className="min-h-[40px] max-h-[120px] resize-none border-0 p-0 focus-visible:ring-0 focus-visible:ring-offset-0 bg-transparent"
                            style={{ height: 'auto' }}
                        />

                        {/* Character count for long messages */}
                        {value.length > 500 && (
                            <div className="absolute bottom-1 right-1 text-xs text-muted-foreground bg-background px-1 rounded">
                                {value.length}/2000
                            </div>
                        )}
                    </div>

                    {/* Action Buttons */}
                    <div className="flex items-center gap-1">
                        {/* Formatting Toggle */}
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setShowFormatting(!showFormatting)}
                            className={`h-8 w-8 p-0 ${showFormatting ? 'bg-accent' : ''}`}
                            disabled={disabled}
                        >
                            <Bold className="h-4 w-4" />
                        </Button>

                        {/* Voice Input */}
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleVoiceToggle}
                            className={`h-8 w-8 p-0 ${isRecording ? 'bg-red-500/10 text-red-600' : ''}`}
                            disabled={disabled}
                            aria-label={isRecording ? "Stop recording" : "Start voice input"}
                        >
                            {isRecording ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4" />}
                        </Button>

                        {/* Send Button */}
                        <Button
                            onClick={handleSend}
                            disabled={disabled || !value.trim() || isLoading}
                            size="sm"
                            className="h-8 w-8 p-0"
                            aria-label={isLoading ? "Sending message" : "Send message"}
                        >
                            {isLoading ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                                <Send className="h-4 w-4" />
                            )}
                        </Button>
                    </div>
                </div>

                {/* Recording Indicator */}
                {isRecording && (
                    <div className="absolute inset-x-0 bottom-0 h-1 bg-red-500/20">
                        <div className="h-full bg-red-500 animate-pulse" />
                    </div>
                )}
            </div>

            {/* Input Hints */}
            <div className="flex items-center justify-between text-xs text-muted-foreground">
                <div className="flex items-center gap-4">
                    <span>Press Enter to send, Shift+Enter for new line</span>
                    {!showFormatting && (
                        <button
                            onClick={() => setShowFormatting(true)}
                            className="hover:text-foreground transition-colors"
                        >
                            Show formatting options
                        </button>
                    )}
                </div>
                {isRecording && (
                    <div className="flex items-center gap-1 text-red-600">
                        <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                        <span>Recording...</span>
                    </div>
                )}
            </div>
        </div>
    );
}