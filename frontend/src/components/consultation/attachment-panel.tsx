'use client';

import { useState, useCallback } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
    Paperclip,
    X,
    Upload,
    File,
    FileText,
    Image,
    Trash2,
    Search,
    ExternalLink,
    Plus
} from 'lucide-react';
import { useDropzone } from 'react-dropzone';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api';


interface AttachmentPanelProps {
    sessionId: string;
    onClose: () => void;
}

interface AttachedDocument {
    id: string;
    name: string;
    type: string;
    size: number;
    url?: string;
    isUploaded: boolean;
}

export function AttachmentPanel({ sessionId: _sessionId, onClose }: AttachmentPanelProps) {
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedDocuments, setSelectedDocuments] = useState<string[]>([]);
    const [attachedDocuments, setAttachedDocuments] = useState<AttachedDocument[]>([]);
    const [uploadingFiles, setUploadingFiles] = useState<File[]>([]);

    // Fetch user's documents
    const { data: documentsData, isLoading } = useQuery({
        queryKey: ['documents', { searchQuery }],
        queryFn: () => apiClient.documents.searchDocuments(searchQuery || ''),
        staleTime: 30 * 1000,
    });

    const onDrop = useCallback((acceptedFiles: File[]) => {
        setUploadingFiles(prev => [...prev, ...acceptedFiles]);

        // Simulate upload process
        acceptedFiles.forEach((file, index) => {
            setTimeout(() => {
                const attachedDoc: AttachedDocument = {
                    id: `temp_${Date.now()}_${index}`,
                    name: file.name,
                    type: file.type,
                    size: file.size,
                    isUploaded: true,
                };

                setAttachedDocuments(prev => [...prev, attachedDoc]);
                setUploadingFiles(prev => prev.filter(f => f !== file));
            }, 1000 + index * 500);
        });
    }, []);

    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        onDrop,
        accept: {
            'application/pdf': ['.pdf'],
            'application/msword': ['.doc'],
            'application/vnd.openxmlformats-officedocument.wordprocessingml.document': ['.docx'],
            'application/vnd.ms-excel': ['.xls'],
            'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx'],
            'text/plain': ['.txt'],
            'text/csv': ['.csv'],
            'application/json': ['.json'],
            'image/*': ['.png', '.jpg', '.jpeg', '.gif', '.webp'],
        },
        maxSize: 10 * 1024 * 1024, // 10MB
    });

    const handleDocumentSelect = (documentId: string) => {
        setSelectedDocuments(prev =>
            prev.includes(documentId)
                ? prev.filter(id => id !== documentId)
                : [...prev, documentId]
        );
    };

    const handleAttachSelected = () => {
        const documents = documentsData?.data || [];
        const toAttach = documents.filter(doc => selectedDocuments.includes(doc.id));

        const newAttachments: AttachedDocument[] = toAttach.map(doc => ({
            id: doc.id,
            name: doc.name,
            type: doc.type,
            size: doc.size,
            ...(doc.downloadUrl && { url: doc.downloadUrl }),
            isUploaded: true,
        }));

        setAttachedDocuments(prev => [...prev, ...newAttachments]);
        setSelectedDocuments([]);
    };

    const handleRemoveAttachment = (documentId: string) => {
        setAttachedDocuments(prev => prev.filter(doc => doc.id !== documentId));
    };

    const getFileIcon = (type: string) => {
        if (type.startsWith('image/')) return Image;
        if (type.includes('pdf')) return FileText;
        return File;
    };

    const formatFileSize = (bytes: number) => {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    return (
        <Card className="border-b-0 rounded-b-none">
            <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                    <CardTitle className="text-base flex items-center gap-2">
                        <Paperclip className="h-4 w-4" />
                        Attach Documents
                    </CardTitle>
                    <Button variant="ghost" size="sm" onClick={onClose}>
                        <X className="h-4 w-4" />
                    </Button>
                </div>
            </CardHeader>

            <CardContent className="space-y-4">
                {/* Currently Attached Documents */}
                {attachedDocuments.length > 0 && (
                    <div className="space-y-2">
                        <h4 className="text-sm font-medium">Attached Documents ({attachedDocuments.length})</h4>
                        <div className="space-y-2 max-h-32 overflow-y-auto">
                            {attachedDocuments.map((doc) => {
                                const Icon = getFileIcon(doc.type);
                                return (
                                    <div key={doc.id} className="flex items-center gap-3 p-2 bg-muted/50 rounded-lg">
                                        <Icon className="h-4 w-4 text-muted-foreground" />
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium truncate">{doc.name}</p>
                                            <p className="text-xs text-muted-foreground">{formatFileSize(doc.size)}</p>
                                        </div>
                                        <div className="flex items-center gap-1">
                                            {doc.url && (
                                                <Button
                                                    variant="ghost"
                                                    size="sm"
                                                    onClick={() => window.open(doc.url, '_blank')}
                                                    className="h-7 w-7 p-0"
                                                >
                                                    <ExternalLink className="h-3 w-3" />
                                                </Button>
                                            )}
                                            <Button
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => handleRemoveAttachment(doc.id)}
                                                className="h-7 w-7 p-0 text-destructive hover:text-destructive"
                                            >
                                                <Trash2 className="h-3 w-3" />
                                            </Button>
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    </div>
                )}

                {/* Upload Area */}
                <div
                    {...getRootProps()}
                    className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${isDragActive
                        ? 'border-primary bg-primary/5'
                        : 'border-muted-foreground/25 hover:border-primary/50'
                        }`}
                >
                    <input {...getInputProps()} />
                    <Upload className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                    {isDragActive ? (
                        <p className="text-sm text-primary">Drop files here...</p>
                    ) : (
                        <div className="space-y-1">
                            <p className="text-sm font-medium">Drop files here or click to browse</p>
                            <p className="text-xs text-muted-foreground">
                                Supports PDF, Word, Excel, images, and text files (max 10MB)
                            </p>
                        </div>
                    )}
                </div>

                {/* Uploading Files */}
                {uploadingFiles.length > 0 && (
                    <div className="space-y-2">
                        <h4 className="text-sm font-medium">Uploading...</h4>
                        {uploadingFiles.map((file, index) => (
                            <div key={index} className="flex items-center gap-3 p-2 bg-muted/30 rounded-lg">
                                <File className="h-4 w-4 text-muted-foreground" />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium truncate">{file.name}</p>
                                    <div className="w-full bg-muted rounded-full h-1 mt-1">
                                        <div className="bg-primary h-1 rounded-full animate-pulse" style={{ width: '60%' }} />
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}

                {/* Document Library */}
                <div className="space-y-3">
                    <div className="flex items-center gap-2">
                        <h4 className="text-sm font-medium">Your Documents</h4>
                        {selectedDocuments.length > 0 && (
                            <Button
                                onClick={handleAttachSelected}
                                size="sm"
                                className="gap-1"
                            >
                                <Plus className="h-3 w-3" />
                                Attach {selectedDocuments.length}
                            </Button>
                        )}
                    </div>

                    <div className="flex items-center gap-2">
                        <div className="relative flex-1">
                            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                            <Input
                                placeholder="Search documents..."
                                value={searchQuery}
                                onChange={(e) => setSearchQuery(e.target.value)}
                                className="pl-9"
                            />
                        </div>
                    </div>

                    <div className="max-h-48 overflow-y-auto space-y-1">
                        {isLoading ? (
                            <div className="text-center py-4">
                                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto mb-2"></div>
                                <p className="text-sm text-muted-foreground">Loading documents...</p>
                            </div>
                        ) : documentsData?.data.length === 0 ? (
                            <div className="text-center py-4">
                                <File className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                                <p className="text-sm text-muted-foreground">
                                    {searchQuery ? 'No documents found' : 'No documents available'}
                                </p>
                            </div>
                        ) : (
                            documentsData?.data.map((doc) => {
                                const Icon = getFileIcon(doc.type);
                                const isSelected = selectedDocuments.includes(doc.id);
                                const isAlreadyAttached = attachedDocuments.some(attached => attached.id === doc.id);

                                return (
                                    <div
                                        key={doc.id}
                                        className={`flex items-center gap-3 p-2 rounded-lg cursor-pointer transition-colors ${isSelected
                                            ? 'bg-primary/10 border border-primary/20'
                                            : 'hover:bg-muted/50'
                                            } ${isAlreadyAttached ? 'opacity-50' : ''}`}
                                        onClick={() => !isAlreadyAttached && handleDocumentSelect(doc.id)}
                                    >
                                        <div className="flex items-center gap-2">
                                            <input
                                                type="checkbox"
                                                checked={isSelected}
                                                onChange={() => { }}
                                                disabled={isAlreadyAttached}
                                                className="rounded"
                                            />
                                            <Icon className="h-4 w-4 text-muted-foreground" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <p className="text-sm font-medium truncate">{doc.name}</p>
                                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                                <span>{formatFileSize(doc.size)}</span>
                                                {doc.classification && (
                                                    <Badge variant="outline" className="text-xs">
                                                        {doc.classification}
                                                    </Badge>
                                                )}
                                            </div>
                                        </div>
                                        {isAlreadyAttached && (
                                            <Badge variant="secondary" className="text-xs">
                                                Attached
                                            </Badge>
                                        )}
                                    </div>
                                );
                            })
                        )}
                    </div>
                </div>

                {/* Instructions */}
                <div className="text-xs text-muted-foreground space-y-1">
                    <p>💡 <strong>Tip:</strong> Attached documents provide context for better AI responses</p>
                    <p>📄 <strong>Supported:</strong> PDF, Word, Excel, PowerPoint, images, and text files</p>
                </div>
            </CardContent>
        </Card>
    );
}