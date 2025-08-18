'use client';

import React from 'react';
import { FileText, Download, Eye, Edit, Trash2, MoreHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Document } from '@/types';
import { cn } from '@/lib/utils';

interface DocumentListProps {
  documents: Document[];
  selectedDocuments: string[];
  onSelectDocument: (id: string) => void;
  onDeselectDocument: (id: string) => void;
  onSelectAll: (documentIds: string[]) => void;
  onPreviewDocument: (document: Document) => void;
  onEditDocument: (document: Document) => void;
  onDownloadDocument: (document: Document) => void;
  onDeleteDocument: (document: Document) => void;
  onSort?: (field: string, direction: 'asc' | 'desc') => void;
  sortField?: string;
  sortDirection?: 'asc' | 'desc';
  className?: string;
}

export function DocumentList({
  documents,
  selectedDocuments,
  onSelectDocument,
  onDeselectDocument,
  onSelectAll,
  onPreviewDocument,
  onEditDocument,
  onDownloadDocument,
  onDeleteDocument,
  onSort,
  sortField,
  sortDirection,
  className,
}: DocumentListProps) {
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
    return <FileText className="h-4 w-4 text-muted-foreground" />;
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

  const handleSelectAllChange = (checked: boolean) => {
    if (checked) {
      onSelectAll(documents.map(doc => doc.id));
    } else {
      onSelectAll([]);
    }
  };

  const isAllSelected = documents.length > 0 && selectedDocuments.length === documents.length;
  const isPartiallySelected = selectedDocuments.length > 0 && selectedDocuments.length < documents.length;

  const handleSort = (field: string) => {
    if (!onSort) return;
    
    const newDirection = sortField === field && sortDirection === 'asc' ? 'desc' : 'asc';
    onSort(field, newDirection);
  };

  const SortableHeader = ({ field, children }: { field: string; children: React.ReactNode }) => (
    <TableHead 
      className={cn(
        onSort && 'cursor-pointer hover:bg-muted/50',
        sortField === field && 'bg-muted/30'
      )}
      onClick={() => handleSort(field)}
    >
      <div className="flex items-center space-x-1">
        <span>{children}</span>
        {sortField === field && (
          <span className="text-xs">
            {sortDirection === 'asc' ? '↑' : '↓'}
          </span>
        )}
      </div>
    </TableHead>
  );

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
    <div className={cn('border rounded-lg', className)}>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-12">
              <Checkbox
                checked={isAllSelected}
                ref={(el) => {
                  if (el) {
                    el.indeterminate = isPartiallySelected;
                  }
                }}
                onCheckedChange={handleSelectAllChange}
              />
            </TableHead>
            <SortableHeader field="name">Name</SortableHeader>
            <SortableHeader field="type">Type</SortableHeader>
            <SortableHeader field="size">Size</SortableHeader>
            <TableHead>Status</TableHead>
            <TableHead>Classification</TableHead>
            <TableHead>Tags</TableHead>
            <SortableHeader field="uploadedAt">Uploaded</SortableHeader>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {documents.map((document) => (
            <TableRow
              key={document.id}
              className={cn(
                'group',
                selectedDocuments.includes(document.id) && 'bg-muted/50'
              )}
            >
              <TableCell>
                <Checkbox
                  checked={selectedDocuments.includes(document.id)}
                  onCheckedChange={(checked) => handleSelectChange(document.id, checked as boolean)}
                />
              </TableCell>
              
              <TableCell>
                <div className="flex items-center space-x-3">
                  {document.thumbnail ? (
                    <img
                      src={document.thumbnail}
                      alt={document.name}
                      className="w-8 h-10 object-cover rounded border"
                    />
                  ) : (
                    getFileIcon(document.type)
                  )}
                  <div className="min-w-0 flex-1">
                    <button
                      className="font-medium text-sm hover:text-primary text-left truncate block w-full"
                      onClick={() => onPreviewDocument(document)}
                      title={document.name}
                    >
                      {document.name}
                    </button>
                    {document.metadata.description && (
                      <p className="text-xs text-muted-foreground truncate">
                        {document.metadata.description}
                      </p>
                    )}
                  </div>
                </div>
              </TableCell>
              
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {document.type.toUpperCase()}
                </span>
              </TableCell>
              
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {formatFileSize(document.size)}
                </span>
              </TableCell>
              
              <TableCell>
                <Badge className={getStatusColor(document.status)} variant="secondary">
                  {document.status}
                </Badge>
              </TableCell>
              
              <TableCell>
                {document.classification && (
                  <Badge className={getClassificationColor(document.classification)} variant="secondary">
                    {document.classification}
                  </Badge>
                )}
              </TableCell>
              
              <TableCell>
                <div className="flex flex-wrap gap-1 max-w-32">
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
              </TableCell>
              
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {formatDate(document.uploadedAt)}
                </span>
              </TableCell>
              
              <TableCell>
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
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}