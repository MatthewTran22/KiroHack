'use client';

import React, { useState } from 'react';
import { X, Download, Edit, Trash2, FileText, Image, Video, Music, Archive, Code } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Document } from '@/types';
import { cn } from '@/lib/utils';

interface DocumentPreviewProps {
  document: Document | null;
  isOpen: boolean;
  onClose: () => void;
  onEdit: (document: Document) => void;
  onDownload: (document: Document) => void;
  onDelete: (document: Document) => void;
}

export function DocumentPreview({
  document,
  isOpen,
  onClose,
  onEdit,
  onDownload,
  onDelete,
}: DocumentPreviewProps) {
  const [activeTab, setActiveTab] = useState('preview');

  if (!document) return null;

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatDate = (date: Date) => {
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(date));
  };

  const getFileIcon = (type: string) => {
    const lowerType = type.toLowerCase();
    
    if (lowerType.includes('image')) {
      return <Image className="h-8 w-8 text-blue-500" />;
    }
    if (lowerType.includes('video')) {
      return <Video className="h-8 w-8 text-purple-500" />;
    }
    if (lowerType.includes('audio')) {
      return <Music className="h-8 w-8 text-green-500" />;
    }
    if (lowerType.includes('zip') || lowerType.includes('archive')) {
      return <Archive className="h-8 w-8 text-orange-500" />;
    }
    if (lowerType.includes('json') || lowerType.includes('xml') || lowerType.includes('code')) {
      return <Code className="h-8 w-8 text-gray-500" />;
    }
    
    return <FileText className="h-8 w-8 text-muted-foreground" />;
  };

  const getStatusColor = (status: Document['status']) => {
    switch (status) {
      case 'completed':
        return 'bg-green-100 text-green-800';
      case 'processing':
        return 'bg-blue-100 text-blue-800';
      case 'error':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getClassificationColor = (classification?: Document['classification']) => {
    switch (classification) {
      case 'secret':
        return 'bg-red-100 text-red-800';
      case 'confidential':
        return 'bg-orange-100 text-orange-800';
      case 'internal':
        return 'bg-yellow-100 text-yellow-800';
      case 'public':
        return 'bg-green-100 text-green-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const canPreview = (type: string) => {
    const previewableTypes = ['pdf', 'txt', 'json', 'xml', 'csv'];
    return previewableTypes.some(t => type.toLowerCase().includes(t));
  };

  const renderPreview = () => {
    const type = document.type.toLowerCase();

    if (document.previewUrl) {
      if (type.includes('image')) {
        return (
          <div className="flex justify-center p-4">
            <img
              src={document.previewUrl}
              alt={document.name}
              className="max-w-full max-h-96 object-contain rounded border"
            />
          </div>
        );
      }

      if (type.includes('pdf')) {
        return (
          <div className="w-full h-96">
            <iframe
              src={document.previewUrl}
              className="w-full h-full border rounded"
              title={document.name}
            />
          </div>
        );
      }
    }

    if (canPreview(type)) {
      return (
        <div className="p-4 bg-muted/30 rounded border">
          <p className="text-sm text-muted-foreground text-center">
            Preview not available. Click download to view the full document.
          </p>
        </div>
      );
    }

    return (
      <div className="flex flex-col items-center justify-center p-8 text-center">
        {getFileIcon(document.type)}
        <h3 className="mt-4 text-lg font-medium">Preview not available</h3>
        <p className="text-sm text-muted-foreground mt-2">
          This file type cannot be previewed in the browser.
          <br />
          Download the file to view its contents.
        </p>
        <Button
          onClick={() => onDownload(document)}
          className="mt-4"
        >
          <Download className="h-4 w-4 mr-2" />
          Download File
        </Button>
      </div>
    );
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <div className="flex items-start justify-between">
            <div className="flex-1 min-w-0">
              <DialogTitle className="text-xl truncate pr-4">
                {document.name}
              </DialogTitle>
              <div className="flex items-center gap-2 mt-2">
                <Badge className={getStatusColor(document.status)} variant="secondary">
                  {document.status}
                </Badge>
                {document.classification && (
                  <Badge className={getClassificationColor(document.classification)} variant="secondary">
                    {document.classification}
                  </Badge>
                )}
                <span className="text-sm text-muted-foreground">
                  {formatFileSize(document.size)} • {document.type.toUpperCase()}
                </span>
              </div>
            </div>
            <div className="flex items-center gap-2 flex-shrink-0">
              <Button
                variant="outline"
                size="sm"
                onClick={() => onDownload(document)}
              >
                <Download className="h-4 w-4 mr-2" />
                Download
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onEdit(document)}
              >
                <Edit className="h-4 w-4 mr-2" />
                Edit
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onDelete(document)}
                className="text-red-600 hover:text-red-700"
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-hidden">
          <Tabs value={activeTab} onValueChange={setActiveTab} className="h-full flex flex-col">
            <TabsList className="grid w-full grid-cols-2 flex-shrink-0">
              <TabsTrigger value="preview">Preview</TabsTrigger>
              <TabsTrigger value="details">Details</TabsTrigger>
            </TabsList>

            <TabsContent value="preview" className="flex-1 overflow-auto mt-4">
              <Card className="h-full">
                <CardContent className="p-0">
                  {renderPreview()}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="details" className="flex-1 overflow-auto mt-4">
              <div className="space-y-6">
                {/* Basic Information */}
                <Card>
                  <CardHeader>
                    <CardTitle>Basic Information</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">File Name</label>
                        <p className="text-sm">{document.name}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">File Type</label>
                        <p className="text-sm">{document.type.toUpperCase()}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">File Size</label>
                        <p className="text-sm">{formatFileSize(document.size)}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Uploaded</label>
                        <p className="text-sm">{formatDate(document.uploadedAt)}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Metadata */}
                <Card>
                  <CardHeader>
                    <CardTitle>Metadata</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 gap-4">
                      {document.metadata.title && (
                        <div>
                          <label className="text-sm font-medium text-muted-foreground">Title</label>
                          <p className="text-sm">{document.metadata.title}</p>
                        </div>
                      )}
                      {document.metadata.description && (
                        <div>
                          <label className="text-sm font-medium text-muted-foreground">Description</label>
                          <p className="text-sm">{document.metadata.description}</p>
                        </div>
                      )}
                      <div className="grid grid-cols-2 gap-4">
                        {document.metadata.author && (
                          <div>
                            <label className="text-sm font-medium text-muted-foreground">Author</label>
                            <p className="text-sm">{document.metadata.author}</p>
                          </div>
                        )}
                        {document.metadata.department && (
                          <div>
                            <label className="text-sm font-medium text-muted-foreground">Department</label>
                            <p className="text-sm">{document.metadata.department}</p>
                          </div>
                        )}
                        {document.metadata.category && (
                          <div>
                            <label className="text-sm font-medium text-muted-foreground">Category</label>
                            <p className="text-sm">{document.metadata.category}</p>
                          </div>
                        )}
                        {document.metadata.version && (
                          <div>
                            <label className="text-sm font-medium text-muted-foreground">Version</label>
                            <p className="text-sm">{document.metadata.version}</p>
                          </div>
                        )}
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Tags */}
                {document.tags.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Tags</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="flex flex-wrap gap-2">
                        {document.tags.map((tag, index) => (
                          <Badge key={index} variant="outline">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Security */}
                <Card>
                  <CardHeader>
                    <CardTitle>Security & Access</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Classification</label>
                        <div className="mt-1">
                          {document.classification ? (
                            <Badge className={getClassificationColor(document.classification)} variant="secondary">
                              {document.classification}
                            </Badge>
                          ) : (
                            <span className="text-sm text-muted-foreground">Not classified</span>
                          )}
                        </div>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-muted-foreground">Status</label>
                        <div className="mt-1">
                          <Badge className={getStatusColor(document.status)} variant="secondary">
                            {document.status}
                          </Badge>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsContent>
          </Tabs>
        </div>
      </DialogContent>
    </Dialog>
  );
}