'use client';

import React from 'react';
import { FileText, Download, Eye, Edit, Trash2, MoreHorizontal } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Document } from '@/types';
import { cn } from '@/lib/utils';

interface DocumentGridProps {
  documents: Document[];
  selectedDocuments: string[];
  onSelectDocument: (id: string) => void;
  onDeselectDocument: (id: string) => void;
  onPreviewDocument: (document: Document) => void;
  onEditDocument: (document: Document) => void;
  onDownloadDocument: (document: Document) => void;
  onDeleteDocument: (document: Document) => void;
  className?: string;
}

export function DocumentGrid({
  documents,
  selectedDocuments,
  onSelectDocument,
  onDeselectDocument,
  onPreviewDocument,
  onEditDocument,
  onDownloadDocument,
  onDeleteDocument,
  className,
}: DocumentGridProps) {
  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    const size = parseFloat((bytes / Math.pow(k, i)).toFixed(2));
    return `${size} ${sizes[i]}`;
  };

  const formatDate = (date: Date) => {
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(date));
  };

  const getFileIcon = (type: string) => {
    // You could expand this with more specific icons based on file type
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

  const handleSelectChange = (documentId: string, checked: boolean) => {
    if (checked) {
      onSelectDocument(documentId);
    } else {
      onDeselectDocument(documentId);
    }
  };

  if (documents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <FileText className="h-12 w-12 text-muted-foreground mb-4" />
        <h3 className="text-lg font-medium text-muted-foreground mb-2">No documents found</h3>
        <p className="text-sm text-muted-foreground">
          Upload some documents or adjust your filters to see results.
        </p>
      </div>
    );
  }

  return (
    <div className={cn('grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4', className)}>
      {documents.map((document) => (
        <Card
          key={document.id}
          className={cn(
            'group hover:shadow-md transition-shadow cursor-pointer',
            selectedDocuments.includes(document.id) && 'ring-2 ring-primary'
          )}
        >
          <CardContent className="p-4">
            {/* Header with checkbox and menu */}
            <div className="flex items-start justify-between mb-3">
              <Checkbox
                checked={selectedDocuments.includes(document.id)}
                onCheckedChange={(checked) => handleSelectChange(document.id, checked as boolean)}
                className="mt-1"
              />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => onPreviewDocument(document)}>
                    <Eye className="h-4 w-4 mr-2" />
                    Preview
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => onDownloadDocument(document)}>
                    <Download className="h-4 w-4 mr-2" />
                    Download
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => onEditDocument(document)}>
                    <Edit className="h-4 w-4 mr-2" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem 
                    onClick={() => onDeleteDocument(document)}
                    className="text-red-600"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            {/* File icon and preview */}
            <div 
              className="flex flex-col items-center mb-4 cursor-pointer"
              onClick={() => onPreviewDocument(document)}
            >
              {document.thumbnail ? (
                <img
                  src={document.thumbnail}
                  alt={document.name}
                  className="w-16 h-20 object-cover rounded border"
                />
              ) : (
                getFileIcon(document.type)
              )}
            </div>

            {/* Document info */}
            <div className="space-y-2">
              <h3 
                className="font-medium text-sm line-clamp-2 cursor-pointer hover:text-primary"
                onClick={() => onPreviewDocument(document)}
                title={document.name}
              >
                {document.name}
              </h3>

              <div className="flex items-center justify-between text-xs text-muted-foreground">
                <span>{formatFileSize(document.size)}</span>
                <span>{document.type.toUpperCase()}</span>
              </div>

              <div className="flex flex-wrap gap-1">
                <Badge className={getStatusColor(document.status)} variant="secondary">
                  {document.status}
                </Badge>
                {document.classification && (
                  <Badge className={getClassificationColor(document.classification)} variant="secondary">
                    {document.classification}
                  </Badge>
                )}
              </div>

              {document.tags.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {document.tags.slice(0, 2).map((tag, index) => (
                    <Badge key={index} variant="outline" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                  {document.tags.length > 2 && (
                    <Badge variant="outline" className="text-xs">
                      +{document.tags.length - 2}
                    </Badge>
                  )}
                </div>
              )}

              <div className="text-xs text-muted-foreground">
                {formatDate(document.uploadedAt)}
              </div>
            </div>

            {/* Quick actions */}
            <div className="flex justify-between mt-3 pt-3 border-t opacity-0 group-hover:opacity-100 transition-opacity">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onPreviewDocument(document)}
              >
                <Eye className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDownloadDocument(document)}
              >
                <Download className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onEditDocument(document)}
              >
                <Edit className="h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}