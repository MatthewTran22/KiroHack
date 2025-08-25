'use client';

import { useState } from 'react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
    User,
    Bot,
    MoreVertical,
    Copy,
    ThumbsUp,
    ThumbsDown,
    Bookmark,
    Share,
    Volume2,
    VolumeX,
    ChevronDown,
    ChevronUp,
    ExternalLink,
    FileText,
    Clock,
    Zap
} from 'lucide-react';
import { useConsultationStore } from '@/stores/consultations';
import { useSynthesizeSpeech } from '@/hooks/useConsultations';
import type { Message } from '@/stores/consultations';

interface MessageListProps {
    messages: Message[];
    isTyping?: boolean;
    sessionId: string;
}

interface MessageItemProps {
    message: Message;
    onReaction: (messageId: string, type: 'helpful' | 'not_helpful' | 'save' | 'share') => void;
}

function MessageItem({ message, onReaction }: MessageItemProps) {
    const [isExpanded, setIsExpanded] = useState(false);
    const [isPlaying, setIsPlaying] = useState(false);
    const [showSources, setShowSources] = useState(false);

    const { voiceSettings, addMessageReaction } = useConsultationStore();
    const synthesizeSpeech = useSynthesizeSpeech();

    const isUser = message.type === 'user';
    const isAssistant = message.type === 'assistant';
    const isSystem = message.type === 'system';

    const handleCopy = () => {
        navigator.clipboard.writeText(message.content);
    };

    const handleReaction = (type: 'helpful' | 'not_helpful' | 'save' | 'share') => {
        const reaction = {
            type,
            timestamp: new Date(),
        };
        addMessageReaction(message.id, reaction);
        onReaction(message.id, type);
    };

    const handlePlayAudio = async () => {
        if (message.audioUrl) {
            // Play existing audio
            const audio = new Audio(message.audioUrl);
            setIsPlaying(true);
            audio.onended = () => setIsPlaying(false);
            audio.play();
        } else if (isAssistant) {
            // Synthesize speech for assistant messages
            try {
                setIsPlaying(true);
                const audioBlob = await synthesizeSpeech.mutateAsync({
                    text: message.content,
                    options: {
                        voice: voiceSettings.voice,
                        rate: voiceSettings.speechRate,
                    },
                });

                const audioUrl = URL.createObjectURL(audioBlob);
                const audio = new Audio(audioUrl);
                audio.onended = () => {
                    setIsPlaying(false);
                    URL.revokeObjectURL(audioUrl);
                };
                audio.play();
            } catch (error) {
                console.error('Failed to synthesize speech:', error);
                setIsPlaying(false);
            }
        }
    };

    const formatTime = (date: Date) => {
        return new Intl.DateTimeFormat('en-US', {
            hour: '2-digit',
            minute: '2-digit',
        }).format(date);
    };

    const shouldTruncate = message.content.length > 500;
    const displayContent = shouldTruncate && !isExpanded
        ? message.content.substring(0, 500) + '...'
        : message.content;

    if (isSystem) {
        return (
            <div className="flex justify-center my-4">
                <Badge variant="secondary" className="text-xs">
                    {message.content}
                </Badge>
            </div>
        );
    }

    return (
        <div className={`flex gap-3 p-4 ${isUser ? 'flex-row-reverse' : 'flex-row'}`}>
            {/* Avatar */}
            <Avatar className="h-8 w-8 shrink-0">
                <AvatarFallback className={isUser ? 'bg-primary text-primary-foreground' : 'bg-muted'}>
                    {isUser ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
                </AvatarFallback>
            </Avatar>

            {/* Message Content */}
            <div className={`flex-1 max-w-[80%] ${isUser ? 'items-end' : 'items-start'} flex flex-col`}>
                <Card className={`p-4 ${isUser
                    ? 'bg-primary text-primary-foreground ml-auto'
                    : 'bg-muted/50'
                    }`}>
                    {/* Message Header */}
                    <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">
                                {isUser ? 'You' : 'AI Assistant'}
                            </span>
                            {message.inputMethod === 'voice' && (
                                <Badge variant="outline" className="text-xs">
                                    <Volume2 className="h-3 w-3 mr-1" />
                                    Voice
                                </Badge>
                            )}
                            {message.confidence && (
                                <Badge variant="outline" className="text-xs">
                                    <Zap className="h-3 w-3 mr-1" />
                                    {Math.round(message.confidence * 100)}%
                                </Badge>
                            )}
                        </div>
                        <span className="text-xs opacity-70">
                            {formatTime(message.timestamp)}
                        </span>
                    </div>

                    {/* Message Content */}
                    <div className="prose prose-sm max-w-none dark:prose-invert">
                        <div className="whitespace-pre-wrap break-words">
                            {displayContent}
                        </div>

                        {shouldTruncate && (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setIsExpanded(!isExpanded)}
                                className="mt-2 p-0 h-auto text-xs"
                            >
                                {isExpanded ? (
                                    <>
                                        <ChevronUp className="h-3 w-3 mr-1" />
                                        Show less
                                    </>
                                ) : (
                                    <>
                                        <ChevronDown className="h-3 w-3 mr-1" />
                                        Show more
                                    </>
                                )}
                            </Button>
                        )}
                    </div>

                    {/* Sources */}
                    {message.sources && message.sources.length > 0 && (
                        <div className="mt-3 pt-3 border-t border-border/20">
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setShowSources(!showSources)}
                                className="p-0 h-auto text-xs mb-2"
                            >
                                <FileText className="h-3 w-3 mr-1" />
                                {message.sources.length} source{message.sources.length > 1 ? 's' : ''}
                                {showSources ? <ChevronUp className="h-3 w-3 ml-1" /> : <ChevronDown className="h-3 w-3 ml-1" />}
                            </Button>

                            {showSources && (
                                <div className="space-y-2">
                                    {message.sources.map((source, index) => (
                                        <div key={index} className="text-xs p-2 bg-background/50 rounded border">
                                            <div className="flex items-center justify-between mb-1">
                                                <span className="font-medium truncate">{source.title}</span>
                                                <Badge variant="outline" className="text-xs">
                                                    {Math.round(source.confidence * 100)}%
                                                </Badge>
                                            </div>
                                            <p className="text-muted-foreground line-clamp-2">{source.excerpt}</p>
                                            {source.url && (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    className="p-0 h-auto mt-1 text-xs"
                                                    onClick={() => window.open(source.url, '_blank')}
                                                >
                                                    <ExternalLink className="h-3 w-3 mr-1" />
                                                    View source
                                                </Button>
                                            )}
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    )}

                    {/* Metadata */}
                    {message.metadata && (
                        <div className="mt-2 pt-2 border-t border-border/20 text-xs opacity-70">
                            <div className="flex items-center gap-3">
                                {message.metadata.processingTime && (
                                    <span className="flex items-center gap-1">
                                        <Clock className="h-3 w-3" />
                                        {message.metadata.processingTime}ms
                                    </span>
                                )}
                                {message.metadata.tokensUsed && (
                                    <span>{message.metadata.tokensUsed} tokens</span>
                                )}
                                {message.metadata.modelUsed && (
                                    <span>{message.metadata.modelUsed}</span>
                                )}
                            </div>
                        </div>
                    )}
                </Card>

                {/* Message Actions */}
                <div className="flex items-center gap-1 mt-2">
                    {/* Audio Playback */}
                    {(message.audioUrl || isAssistant) && (
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={handlePlayAudio}
                            disabled={isPlaying || synthesizeSpeech.isPending}
                            className="h-7 px-2"
                        >
                            {isPlaying ? <VolumeX className="h-3 w-3" /> : <Volume2 className="h-3 w-3" />}
                        </Button>
                    )}

                    {/* Copy */}
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={handleCopy}
                        className="h-7 px-2"
                    >
                        <Copy className="h-3 w-3" />
                    </Button>

                    {/* Reactions (for assistant messages) */}
                    {isAssistant && (
                        <>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleReaction('helpful')}
                                className="h-7 px-2"
                            >
                                <ThumbsUp className="h-3 w-3" />
                            </Button>
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleReaction('not_helpful')}
                                className="h-7 px-2"
                            >
                                <ThumbsDown className="h-3 w-3" />
                            </Button>
                        </>
                    )}

                    {/* More Actions */}
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                                <MoreVertical className="h-3 w-3" />
                            </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align={isUser ? 'end' : 'start'} className="w-40">
                            <DropdownMenuItem onClick={() => handleReaction('save')}>
                                <Bookmark className="h-4 w-4 mr-2" />
                                Save
                            </DropdownMenuItem>
                            <DropdownMenuItem onClick={() => handleReaction('share')}>
                                <Share className="h-4 w-4 mr-2" />
                                Share
                            </DropdownMenuItem>
                        </DropdownMenuContent>
                    </DropdownMenu>
                </div>

                {/* Reactions Display */}
                {message.reactions && message.reactions.length > 0 && (
                    <div className="flex items-center gap-1 mt-1">
                        {message.reactions.map((reaction, index) => (
                            <Badge key={index} variant="outline" className="text-xs">
                                {reaction.type === 'helpful' && '👍'}
                                {reaction.type === 'not_helpful' && '👎'}
                                {reaction.type === 'save' && '🔖'}
                                {reaction.type === 'share' && '📤'}
                            </Badge>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}

function TypingIndicator() {
    return (
        <div className="flex gap-3 p-4">
            <Avatar className="h-8 w-8 shrink-0">
                <AvatarFallback className="bg-muted">
                    <Bot className="h-4 w-4" />
                </AvatarFallback>
            </Avatar>
            <Card className="p-4 bg-muted/50">
                <div className="flex items-center gap-2">
                    <div className="flex gap-1">
                        <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                        <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                        <div className="w-2 h-2 bg-muted-foreground rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                    </div>
                    <span className="text-sm text-muted-foreground">AI is thinking...</span>
                </div>
            </Card>
        </div>
    );
}

export function MessageList({ messages, isTyping, sessionId }: MessageListProps) {
    const handleReaction = (messageId: string, type: 'helpful' | 'not_helpful' | 'save' | 'share') => {
        // This would typically make an API call to save the reaction
        console.log('Reaction:', { messageId, type, sessionId });
    };

    return (
        <div className="space-y-1">
            {messages.map((message) => (
                <MessageItem
                    key={message.id}
                    message={message}
                    onReaction={handleReaction}
                />
            ))}
            {isTyping && <TypingIndicator />}
        </div>
    );
}