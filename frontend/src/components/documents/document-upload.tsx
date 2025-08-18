'use client';

import React, { useCallback, useState } from 'react';
import { useDropzone } from 'react-dropzone';
import { Upload, X, File, AlertCircle, CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { useDocumentStore } from '@/stores/documents';
import { useDocuments } from '@/hooks/useDocuments';
import { DocumentMetadata } from '@/types';
import { cn } from '@/lib/utils';

interface FileWithMetadata {
  file: File;
  id: string;
  metadata: DocumentMetadata;
  classification: 'public' | 'internal' | 'confidential' | 'secret';
  tags: string[];
  progress: number;
  status: 'pending' | 'uploading' | 'processing' | 'complete' | 'error';
  error?: string;
}

interface DocumentUploadProps {
  onUploadComplete?: (documents: any[]) => void;
  onUploadError?: (error: string) => void;
  maxFiles?: number;
  acceptedTypes?: string[];
  className?: string;
}

const ACCEPTED_FILE_TYPES = {
  'application/pdf': ['.pdf'],
  'application/msword': ['.doc'],
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': ['.docx'],
  'application/vnd.ms-excel': ['.xls'],
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx'],
  'application/vnd.ms-powerpoint': ['.ppt'],
  'application/vnd.openxmlformats-officedocument.presentationml.presentation': ['.pptx'],
  'text/plain': ['.txt'],
  'text/csv': ['.csv'],
  'application/json': ['.json'],
  'application/xml': ['.xml'],
  'text/xml': ['.xml'],
};

const MAX_FILE_SIZE = 50 * 1024 * 1024; // 50MB

export function DocumentUpload({
  onUploadComplete,
  onUploadError,
  maxFiles = 10,
  acceptedTypes = Object.keys(ACCEPTED_FILE_TYPES),
  className,
}: DocumentUploadProps) {
  const [files, setFiles] = useState<FileWithMetadata[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [showMetadataModal, setShowMetadataModal] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);
  const [tagInput, setTagInput] = useState('');

  const { uploadDocuments } = useDocuments();

  const onDrop = useCallback((acceptedFiles: File[], rejectedFiles: any[]) => {
    // Handle rejected files
    if (rejectedFiles.length > 0) {
      const errors = rejectedFiles.map(({ file, errors }) => 
        `${file.name}: ${errors.map((e: any) => e.message).join(', ')}`
      );
      onUploadError?.(errors.join('\n'));
      return;
    }

    // Check file count limit
    if (files.length + acceptedFiles.length > maxFiles) {
      onUploadError?.(`Maximum ${maxFiles} files allowed`);
      return;
    }

    // Add files with default metadata
    const newFiles: FileWithMetadata[] = acceptedFiles.map(file => ({
      file,
      id: crypto.randomUUID(),
      metadata: {
        title: file.name.replace(/\.[^/.]+$/, ''),
        description: '',
        author: '',
        department: '',
        category: '',
        keywords: [],
        language: 'en',
        version: '1.0',
      },
      classification: 'internal',
      tags: [],
      progress: 0,
      status: 'pending',
    }));

    setFiles(prev => [...prev, ...newFiles]);
  }, [files.length, maxFiles, onUploadError]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: acceptedTypes.reduce((acc, type) => {
      acc[type] = ACCEPTED_FILE_TYPES[type as keyof typeof ACCEPTED_FILE_TYPES] || [];
      return acc;
    }, {} as Record<string, string[]>),
    maxSize: MAX_FILE_SIZE,
    multiple: true,
  });

  const removeFile = (fileId: string) => {
    setFiles(prev => prev.filter(f => f.id !== fileId));
  };

  const updateFileMetadata = (fileId: string, updates: Partial<FileWithMetadata>) => {
    setFiles(prev => prev.map(f => f.id === fileId ? { ...f, ...updates } : f));
  };

  const openMetadataModal = (fileId: string) => {
    setSelectedFileId(fileId);
    setShowMetadataModal(true);
  };

  const closeMetadataModal = () => {
    setSelectedFileId(null);
    setShowMetadataModal(false);
    setTagInput('');
  };

  const addTag = (fileId: string, tag: string) => {
    if (!tag.trim()) return;
    
    updateFileMetadata(fileId, {
      tags: [...(files.find(f => f.id === fileId)?.tags || []), tag.trim()]
    });
  };

  const removeTag = (fileId: string, tagIndex: number) => {
    const file = files.find(f => f.id === fileId);
    if (!file) return;
    
    updateFileMetadata(fileId, {
      tags: file.tags.filter((_, index) => index !== tagIndex)
    });
  };

  const handleUpload = async () => {
    if (files.length === 0) return;

    setIsUploading(true);

    try {
      // Update all files to uploading status
      setFiles(prev => prev.map(f => ({ ...f, status: 'uploading' as const })));

      // Prepare upload requests
      const uploadRequests = files.map(fileData => ({
        file: fileData.file,
        metadata: fileData.metadata,
        classification: fileData.classification,
        tags: fileData.tags,
      }));

      // Simulate progress updates
      const progressInterval = setInterval(() => {
        setFiles(prev => prev.map(f => ({
          ...f,
          progress: Math.min(f.progress + Math.random() * 20, 90)
        })));
      }, 500);

      // Upload documents
      const uploadedDocuments = await uploadDocuments(uploadRequests);

      clearInterval(progressInterval);

      // Update files to complete status
      setFiles(prev => prev.map(f => ({
        ...f,
        progress: 100,
        status: 'complete' as const
      })));

      onUploadComplete?.(uploadedDocuments);

      // Clear files after successful upload
      setTimeout(() => {
        setFiles([]);
      }, 2000);

    } catch (error) {
      // Update files to error status
      setFiles(prev => prev.map(f => ({
        ...f,
        status: 'error' as const,
        error: error instanceof Error ? error.message : 'Upload failed'
      })));

      onUploadError?.(error instanceof Error ? error.message : 'Upload failed');
    } finally {
      setIsUploading(false);
    }
  };

  const selectedFile = selectedFileId ? files.find(f => f.id === selectedFileId) : null;

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getStatusIcon = (status: FileWithMetadata['status']) => {
    switch (status) {
      case 'complete':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-500" />;
      default:
        return <File className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getStatusColor = (status: FileWithMetadata['status']) => {
    switch (status) {
      case 'complete':
        return 'bg-green-100 text-green-800';
      case 'error':
        return 'bg-red-100 text-red-800';
      case 'uploading':
      case 'processing':
        return 'bg-blue-100 text-blue-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className={cn('space-y-4', className)}>
      {/* Drop Zone */}
      <Card>
        <CardContent className="p-6">
          <div
            {...getRootProps()}
            className={cn(
              'border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors',
              isDragActive
                ? 'border-primary bg-primary/5'
                : 'border-muted-foreground/25 hover:border-primary/50'
            )}
          >
            <input {...getInputProps()} />
            <Upload className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <div className="space-y-2">
              <p className="text-lg font-medium">
                {isDragActive ? 'Drop files here' : 'Drag & drop files here'}
              </p>
              <p className="text-sm text-muted-foreground">
                or click to browse files
              </p>
              <p className="text-xs text-muted-foreground">
                Supports PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, TXT, CSV, JSON, XML
                <br />
                Maximum file size: {formatFileSize(MAX_FILE_SIZE)}
                <br />
                Maximum {maxFiles} files
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* File List */}
      {files.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>Files ({files.length})</span>
              <Button
                onClick={handleUpload}
                disabled={isUploading || files.some(f => f.status === 'uploading')}
                className="ml-auto"
              >
                {isUploading ? 'Uploading...' : 'Upload All'}
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {files.map((fileData) => (
              <div
                key={fileData.id}
                className="flex items-center space-x-3 p-3 border rounded-lg"
              >
                {getStatusIcon(fileData.status)}
                
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium truncate">
                      {fileData.file.name}
                    </p>
                    <div className="flex items-center space-x-2">
                      <Badge className={getStatusColor(fileData.status)}>
                        {fileData.status}
                      </Badge>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => openMetadataModal(fileData.id)}
                        disabled={fileData.status === 'uploading'}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeFile(fileData.id)}
                        disabled={fileData.status === 'uploading'}
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                  
                  <div className="flex items-center justify-between mt-1">
                    <p className="text-xs text-muted-foreground">
                      {formatFileSize(fileData.file.size)} • {fileData.classification}
                    </p>
                    {fileData.tags.length > 0 && (
                      <div className="flex space-x-1">
                        {fileData.tags.slice(0, 3).map((tag, index) => (
                          <Badge key={index} variant="secondary" className="text-xs">
                            {tag}
                          </Badge>
                        ))}
                        {fileData.tags.length > 3 && (
                          <Badge variant="secondary" className="text-xs">
                            +{fileData.tags.length - 3}
                          </Badge>
                        )}
                      </div>
                    )}
                  </div>

                  {(fileData.status === 'uploading' || fileData.status === 'processing') && (
                    <Progress value={fileData.progress} className="mt-2" />
                  )}

                  {fileData.status === 'error' && fileData.error && (
                    <p className="text-xs text-red-500 mt-1">{fileData.error}</p>
                  )}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Metadata Modal */}
      <Dialog open={showMetadataModal} onOpenChange={setShowMetadataModal}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Document Metadata</DialogTitle>
            <DialogDescription>
              {selectedFile?.file.name}
            </DialogDescription>
          </DialogHeader>

          {selectedFile && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="title">Title</Label>
                  <Input
                    id="title"
                    value={selectedFile.metadata.title || ''}
                    onChange={(e) => updateFileMetadata(selectedFile.id, {
                      metadata: { ...selectedFile.metadata, title: e.target.value }
                    })}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="author">Author</Label>
                  <Input
                    id="author"
                    value={selectedFile.metadata.author || ''}
                    onChange={(e) => updateFileMetadata(selectedFile.id, {
                      metadata: { ...selectedFile.metadata, author: e.target.value }
                    })}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={selectedFile.metadata.description || ''}
                  onChange={(e) => updateFileMetadata(selectedFile.id, {
                    metadata: { ...selectedFile.metadata, description: e.target.value }
                  })}
                  rows={3}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="department">Department</Label>
                  <Input
                    id="department"
                    value={selectedFile.metadata.department || ''}
                    onChange={(e) => updateFileMetadata(selectedFile.id, {
                      metadata: { ...selectedFile.metadata, department: e.target.value }
                    })}
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="category">Category</Label>
                  <Input
                    id="category"
                    value={selectedFile.metadata.category || ''}
                    onChange={(e) => updateFileMetadata(selectedFile.id, {
                      metadata: { ...selectedFile.metadata, category: e.target.value }
                    })}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="classification">Classification</Label>
                <Select
                  value={selectedFile.classification}
                  onValueChange={(value: any) => updateFileMetadata(selectedFile.id, {
                    classification: value
                  })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="public">Public</SelectItem>
                    <SelectItem value="internal">Internal</SelectItem>
                    <SelectItem value="confidential">Confidential</SelectItem>
                    <SelectItem value="secret">Secret</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>Tags</Label>
                <div className="flex flex-wrap gap-2 mb-2">
                  {selectedFile.tags.map((tag, index) => (
                    <Badge key={index} variant="secondary" className="flex items-center gap-1">
                      {tag}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-4 w-4 p-0 hover:bg-transparent"
                        onClick={() => removeTag(selectedFile.id, index)}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </Badge>
                  ))}
                </div>
                <div className="flex space-x-2">
                  <Input
                    placeholder="Add tag..."
                    value={tagInput}
                    onChange={(e) => setTagInput(e.target.value)}
                    onKeyPress={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        addTag(selectedFile.id, tagInput);
                        setTagInput('');
                      }
                    }}
                  />
                  <Button
                    type="button"
                    onClick={() => {
                      addTag(selectedFile.id, tagInput);
                      setTagInput('');
                    }}
                    disabled={!tagInput.trim()}
                  >
                    Add
                  </Button>
                </div>
              </div>

              <div className="flex justify-end space-x-2 pt-4">
                <Button variant="outline" onClick={closeMetadataModal}>
                  Close
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}