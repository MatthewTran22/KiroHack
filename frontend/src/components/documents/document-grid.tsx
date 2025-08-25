'use client';

import React from 'react';
import { FileText, Download, Eye, Edit, Trash2, MoreHorizontal } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { DocumentPreviewThumbnail } from './document-preview-thumbnail';
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
    <div className={cn('grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6', className)}>
      {documents.map((document) => (
        <div
          key={document.id}
          className={cn(
            'relative group',
            selectedDocuments.includes(document.id) && 'ring-2 ring-primary ring-offset-2 rounded-lg'
          )}
        >
          {/* Selection checkbox */}
          <div className="absolute top-2 left-2 z-10">
            <Checkbox
              checked={selectedDocuments.includes(document.id)}
              onCheckedChange={(checked) => handleSelectChange(document.id, checked as boolean)}
              className="bg-white/80 backdrop-blur-sm border-white shadow-sm data-[state=checked]:bg-primary data-[state=checked]:border-primary"
            />
          </div>

          {/* More options menu */}
          <div className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="secondary"
                  size="sm"
                  className="h-8 w-8 p-0 bg-white/80 backdrop-blur-sm hover:bg-white shadow-sm"
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

          {/* Document Preview Thumbnail */}
          <DocumentPreviewThumbnail
            document={document}
            onClick={() => onPreviewDocument(document)}
            className="h-48"
          />

          {/* Quick actions overlay */}
          <div className="absolute bottom-2 left-2 right-2 flex justify-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
            <Button
              variant="secondary"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onDownloadDocument(document);
              }}
              className="h-8 px-2 bg-white/80 backdrop-blur-sm hover:bg-white shadow-sm"
            >
              <Download className="h-3 w-3" />
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onEditDocument(document);
              }}
              className="h-8 px-2 bg-white/80 backdrop-blur-sm hover:bg-white shadow-sm"
            >
              <Edit className="h-3 w-3" />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}