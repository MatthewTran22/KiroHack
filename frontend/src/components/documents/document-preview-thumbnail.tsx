'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { configurePDFWorker } from '@/lib/pdf-config';
import { cn } from '@/lib/utils';
import { FileIcon, FileTextIcon, ImageIcon, VideoIcon, Volume2Icon } from 'lucide-react';
import type { Document as DocumentType } from '@/types';

interface DocumentPreviewThumbnailProps {
    document: DocumentType;
    className?: string;
    onClick?: () => void;
}

const getFileTypeIcon = (contentType: string, size: number = 24) => {
    const iconProps = { size, className: 'text-muted-foreground' };

    if (contentType.startsWith('image/')) {
        return <ImageIcon {...iconProps} />;
    }
    if (contentType.startsWith('video/')) {
        return <VideoIcon {...iconProps} />;
    }
    if (contentType.startsWith('audio/')) {
        return <Volume2Icon {...iconProps} />;
    }
    if (contentType.includes('pdf')) {
        return <FileTextIcon {...iconProps} />;
    }
    if (contentType.includes('text')) {
        return <FileTextIcon {...iconProps} />;
    }
    return <FileIcon {...iconProps} />;
};

export function DocumentPreviewThumbnail({
    document,
    className,
    onClick
}: DocumentPreviewThumbnailProps) {
    const [isLoading, setIsLoading] = useState(true);
    const [hasError, setHasError] = useState(false);
    const [textContent, setTextContent] = useState<string>('');

    const isPDF = document.type === 'application/pdf' || document.name.toLowerCase().endsWith('.pdf');
    const isImage = document.type.startsWith('image/');
    const isText = document.type.startsWith('text/') || document.type.includes('text');

    // Get document URL for preview
    const getDocumentUrl = useCallback(() => {
        // Use the content endpoint since file endpoint isn't available yet
        const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
        return `${baseUrl}/api/v1/documents/${document.id}/content`;
    }, [document.id]);

    // Get document URL with auth token
    const getAuthenticatedDocumentOptions = useCallback(() => {
        const token = localStorage.getItem('token');
        return {
            headers: {
                'Authorization': `Bearer ${token}`,
            },
        };
    }, []);

    // Initialize PDF worker when component mounts
    useEffect(() => {
        configurePDFWorker();
    }, []);

    // Load text content for text documents
    React.useEffect(() => {
        if (isText && !textContent) {
            const documentUrl = getDocumentUrl();
            const options = getAuthenticatedDocumentOptions();

            fetch(documentUrl, options)
                .then(response => response.json())
                .then(data => {
                    // Extract content from the JSON response
                    const content = data.content || 'No content available';
                    // Show first 300 characters
                    setTextContent(content.substring(0, 300));
                    setIsLoading(false);
                })
                .catch(() => {
                    setHasError(true);
                    setIsLoading(false);
                });
        }
    }, [isText, textContent, getDocumentUrl, getAuthenticatedDocumentOptions]);

    const renderPDFPreview = () => {
        // For now, since the backend doesn't serve actual PDF files, 
        // show a styled PDF representation with extracted content
        return (
            <div className="relative h-full overflow-hidden bg-white rounded p-3">
                <div className="flex flex-col items-center justify-center h-full">
                    <FileTextIcon size={48} className="text-blue-600 mb-3" />
                    <div className="text-center">
                        <div className="text-sm font-semibold text-gray-800 mb-1">
                            PDF Document
                        </div>
                        <div className="text-xs text-gray-600 mb-2">
                            {document.name.split('.')[0]}
                        </div>
                        <div className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                            {(document.size / 1024).toFixed(1)} KB
                        </div>
                    </div>
                </div>

                {/* PDF-like background pattern */}
                <div className="absolute inset-0 bg-gradient-to-b from-gray-50 to-white opacity-50 pointer-events-none" />
                <div className="absolute top-4 left-4 right-4 space-y-1 pointer-events-none">
                    <div className="h-1 bg-gray-200 rounded w-3/4" />
                    <div className="h-1 bg-gray-200 rounded w-1/2" />
                    <div className="h-1 bg-gray-200 rounded w-2/3" />
                </div>

                {/* Document type indicator */}
                <div className="absolute top-2 right-2 bg-red-500 text-white text-xs px-1.5 py-0.5 rounded font-medium">
                    PDF
                </div>
            </div>
        );
    };

    const renderImagePreview = () => {
        return (
            <div className="relative h-full overflow-hidden rounded">
                <img
                    src={getDocumentUrl()}
                    alt={document.name}
                    className="w-full h-full object-cover"
                    onError={() => setHasError(true)}
                    onLoad={() => setIsLoading(false)}
                // Note: For proper authentication with images, you'd need to use fetch + blob URL
                // This is a simplified version that may not work with auth headers
                />
                {isLoading && (
                    <div className="absolute inset-0 flex items-center justify-center bg-muted/30">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary" />
                    </div>
                )}
                {hasError && (
                    <div className="absolute inset-0 flex flex-col items-center justify-center bg-muted/30">
                        <ImageIcon size={32} className="text-muted-foreground mb-2" />
                        <span className="text-xs text-muted-foreground text-center">Image Preview Unavailable</span>
                    </div>
                )}
            </div>
        );
    };

    const renderTextPreview = () => {
        if (hasError) {
            return (
                <div className="flex flex-col items-center justify-center h-full bg-muted/30 rounded">
                    <FileTextIcon size={32} className="text-muted-foreground mb-2" />
                    <span className="text-xs text-muted-foreground text-center">Text Preview Unavailable</span>
                </div>
            );
        }

        return (
            <div className="relative h-full bg-white rounded p-3 overflow-hidden">
                {isLoading && !textContent ? (
                    <div className="flex flex-col items-center justify-center h-full">
                        <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mb-2" />
                        <span className="text-xs text-muted-foreground">Loading text...</span>
                    </div>
                ) : (
                    <>
                        <div className="text-xs leading-relaxed text-gray-700 whitespace-pre-wrap">
                            {textContent}
                            {textContent.length >= 300 && '...'}
                        </div>
                        <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-white pointer-events-none" />
                    </>
                )}
            </div>
        );
    };

    const renderDefaultPreview = () => {
        return (
            <div className="flex flex-col items-center justify-center h-full bg-muted/30 rounded">
                {getFileTypeIcon(document.type, 32)}
                <div className="mt-2 text-xs text-muted-foreground text-center font-medium">
                    {document.type.split('/')[1]?.toUpperCase() || 'FILE'}
                </div>
                <div className="text-xs text-muted-foreground mt-1">
                    {(document.size / 1024).toFixed(1)} KB
                </div>
            </div>
        );
    };

    const renderPreview = () => {
        if (isPDF) {
            return renderPDFPreview();
        }
        if (isImage) {
            return renderImagePreview();
        }
        if (isText) {
            return renderTextPreview();
        }
        return renderDefaultPreview();
    };

    return (
        <div
            className={cn(
                'relative aspect-[3/4] w-full bg-background border border-border rounded-lg overflow-hidden cursor-pointer',
                'hover:shadow-md transition-shadow duration-200',
                'group',
                className
            )}
            onClick={onClick}
        >
            {/* Preview Content */}
            <div className="h-full">
                {renderPreview()}
            </div>

            {/* Overlay with document info */}
            <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 via-black/20 to-transparent p-2">
                <div className="text-white">
                    <div className="text-xs font-medium truncate" title={document.name}>
                        {document.name}
                    </div>
                    <div className="text-xs text-white/80 flex items-center justify-between mt-1">
                        <span>{document.category}</span>
                        <span className="bg-white/20 px-1.5 py-0.5 rounded text-xs">
                            {document.classification}
                        </span>
                    </div>
                </div>
            </div>

            {/* Hover overlay */}
            <div className="absolute inset-0 bg-primary/5 opacity-0 group-hover:opacity-100 transition-opacity duration-200" />
        </div>
    );
}
