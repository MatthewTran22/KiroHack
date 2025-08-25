'use client';

import React, { useState, useEffect } from 'react';
import { Upload, Grid, List, Search, Filter, Download, Trash2, MoreHorizontal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { DocumentUpload } from '@/components/documents/document-upload';
import { DocumentGrid } from '@/components/documents/document-grid';
import { DocumentList } from '@/components/documents/document-list';
import { DocumentFilters } from '@/components/documents/document-filters';
import { DocumentPreview } from '@/components/documents/document-preview';
import { useDocuments, useDeleteDocument, useDeleteDocuments } from '@/hooks/useDocuments';
import { useDocumentStore } from '@/stores/documents';
import { Document, DocumentFilters as DocumentFiltersType } from '@/types';
import { cn } from '@/lib/utils';
import { useQueryClient } from '@tanstack/react-query';

export default function DocumentsPage() {
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [previewDocument, setPreviewDocument] = useState<Document | null>(null);
  const [editDocument, setEditDocument] = useState<Document | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [filters, setFilters] = useState<DocumentFiltersType>({});
  const [sortField, setSortField] = useState<string>('uploadedAt');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('desc');

  const {
    selectedDocuments,
    viewMode,
    selectDocument,
    deselectDocument,
    selectAllDocuments,
    clearSelection,
    setViewMode,
  } = useDocumentStore();

  // TanStack Query hooks
  const queryClient = useQueryClient();
  const { data: documentsResponse, isLoading, error, refetch } = useDocuments();
  const { mutate: deleteDocument } = useDeleteDocument();
  const { mutate: deleteSelectedDocuments } = useDeleteDocuments();

  // Extract documents from the paginated response
  const documents = documentsResponse?.data || [];

  // Refetch when filters change
  useEffect(() => {
    refetch();
  }, [filters, searchQuery, refetch]);

  const handleSearch = (query: string) => {
    setSearchQuery(query);
  };

  const handleFiltersChange = (newFilters: DocumentFiltersType) => {
    setFilters(newFilters);
  };

  const handleClearFilters = () => {
    setFilters({});
    setSearchQuery('');
  };

  const handleSort = (field: string, direction: 'asc' | 'desc') => {
    setSortField(field);
    setSortDirection(direction);

    // Sort documents locally for now
    // In a real app, this would be handled by the API
  };

  const handlePreviewDocument = (document: Document) => {
    setPreviewDocument(document);
  };

  const handleEditDocument = (document: Document) => {
    setEditDocument(document);
    // TODO: Implement edit modal
  };

  const handleDownloadDocument = async (document: Document) => {
    try {
      // TODO: Implement download functionality
      window.open(`http://localhost:8080/api/v1/documents/${document.id}/content`, '_blank');
    } catch (error) {
      console.error('Download failed:', error);
    }
  };

  const handleDeleteDocument = async (document: Document) => {
    if (confirm(`Are you sure you want to delete "${document.name}"?`)) {
      try {
        deleteDocument(document.id);
      } catch (error) {
        console.error('Delete failed:', error);
      }
    }
  };

  const handleBulkDownload = async () => {
    // TODO: Implement bulk download
    console.log('Bulk download:', selectedDocuments);
  };

  const handleBulkDelete = async () => {
    if (confirm(`Are you sure you want to delete ${selectedDocuments.length} document(s)?`)) {
      try {
        deleteSelectedDocuments(selectedDocuments);
        clearSelection();
      } catch (error) {
        console.error('Bulk delete failed:', error);
      }
    }
  };

  const handleUploadComplete = (uploadedDocuments: Document[]) => {
    setShowUploadModal(false);
    // Force refresh documents list to show newly uploaded documents
    refetch();

    // Also clear any cached data to ensure fresh fetch
    // This follows TanStack Query best practices for data freshness
    queryClient.removeQueries({ queryKey: ['documents'] });
    setTimeout(() => refetch(), 100); // Small delay to ensure cache is cleared
  };

  const handleUploadError = (error: string) => {
    console.error('Upload error:', error);
    // TODO: Show error toast
  };

  // Sort documents
  const sortedDocuments = React.useMemo(() => {
    if (!documents) return [];

    return [...documents].sort((a, b) => {
      let aValue: any = a[sortField as keyof Document];
      let bValue: any = b[sortField as keyof Document];

      // Handle date sorting
      if (sortField === 'uploadedAt') {
        aValue = new Date(aValue).getTime();
        bValue = new Date(bValue).getTime();
      }

      // Handle string sorting
      if (typeof aValue === 'string') {
        aValue = aValue.toLowerCase();
        bValue = bValue.toLowerCase();
      }

      if (sortDirection === 'asc') {
        return aValue < bValue ? -1 : aValue > bValue ? 1 : 0;
      } else {
        return aValue > bValue ? -1 : aValue < bValue ? 1 : 0;
      }
    });
  }, [documents, sortField, sortDirection]);

  const hasActiveFilters = !!(
    searchQuery ||
    filters.category ||
    filters.classification ||
    filters.status ||
    (filters.tags && filters.tags.length > 0) ||
    filters.dateRange
  );

  return (
    <div className="container mx-auto py-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Documents</h1>
          <p className="text-muted-foreground">
            Manage and organize your document library
          </p>
        </div>
        <Button onClick={() => setShowUploadModal(true)}>
          <Upload className="h-4 w-4 mr-2" />
          Upload Documents
        </Button>
      </div>

      {/* Search and Controls */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4 flex-1">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search documents..."
                  value={searchQuery}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="pl-10"
                />
              </div>
              <Button
                variant="outline"
                onClick={() => setShowFilters(!showFilters)}
                className={cn(hasActiveFilters && 'bg-primary/10 border-primary')}
              >
                <Filter className="h-4 w-4 mr-2" />
                Filters
                {hasActiveFilters && (
                  <Badge variant="secondary" className="ml-2">
                    Active
                  </Badge>
                )}
              </Button>
            </div>

            <div className="flex items-center gap-2">
              {/* Bulk Actions */}
              {selectedDocuments.length > 0 && (
                <div className="flex items-center gap-2 mr-4">
                  <span className="text-sm text-muted-foreground">
                    {selectedDocuments.length} selected
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleBulkDownload}
                  >
                    <Download className="h-4 w-4 mr-2" />
                    Download
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleBulkDelete}
                    className="text-red-600 hover:text-red-700"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Delete
                  </Button>
                </div>
              )}

              {/* View Toggle */}
              <div className="flex items-center border rounded-lg">
                <Button
                  variant={viewMode === 'grid' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('grid')}
                  className="rounded-r-none"
                >
                  <Grid className="h-4 w-4" />
                </Button>
                <Button
                  variant={viewMode === 'list' ? 'default' : 'ghost'}
                  size="sm"
                  onClick={() => setViewMode('list')}
                  className="rounded-l-none"
                >
                  <List className="h-4 w-4" />
                </Button>
              </div>

              {/* More Actions */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm">
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => refetch()}>
                    Refresh
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={clearSelection}>
                    Clear Selection
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Filters Sidebar */}
        {showFilters && (
          <div className="lg:col-span-1">
            <DocumentFilters
              filters={filters}
              onFiltersChange={handleFiltersChange}
              onClearFilters={handleClearFilters}
            />
          </div>
        )}

        {/* Documents Content */}
        <div className={cn('space-y-6', showFilters ? 'lg:col-span-3' : 'lg:col-span-4')}>
          {/* Results Summary */}
          <div className="flex items-center justify-between">
            <div className="text-sm text-muted-foreground">
              {isLoading ? (
                'Loading documents...'
              ) : (
                `${sortedDocuments.length} document${sortedDocuments.length !== 1 ? 's' : ''} found`
              )}
            </div>
            {sortedDocuments.length > 0 && (
              <div className="text-sm text-muted-foreground">
                Sorted by {sortField} ({sortDirection === 'asc' ? 'ascending' : 'descending'})
              </div>
            )}
          </div>

          {/* Error State */}
          {error && (
            <Card className="border-red-200 bg-red-50">
              <CardContent className="p-4">
                <p className="text-red-600">Error loading documents: {error}</p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => refetch()}
                  className="mt-2"
                >
                  Retry
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Loading State */}
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                <p className="text-muted-foreground">Loading documents...</p>
              </div>
            </div>
          )}

          {/* Documents Display */}
          {!isLoading && !error && (
            <>
              {viewMode === 'grid' ? (
                <DocumentGrid
                  documents={sortedDocuments}
                  selectedDocuments={selectedDocuments}
                  onSelectDocument={selectDocument}
                  onDeselectDocument={deselectDocument}
                  onPreviewDocument={handlePreviewDocument}
                  onEditDocument={handleEditDocument}
                  onDownloadDocument={handleDownloadDocument}
                  onDeleteDocument={handleDeleteDocument}
                />
              ) : (
                <DocumentList
                  documents={sortedDocuments}
                  selectedDocuments={selectedDocuments}
                  onSelectDocument={selectDocument}
                  onDeselectDocument={deselectDocument}
                  onSelectAll={selectAllDocuments}
                  onPreviewDocument={handlePreviewDocument}
                  onEditDocument={handleEditDocument}
                  onDownloadDocument={handleDownloadDocument}
                  onDeleteDocument={handleDeleteDocument}
                  onSort={handleSort}
                  sortField={sortField}
                  sortDirection={sortDirection}
                />
              )}
            </>
          )}
        </div>
      </div>

      {/* Upload Modal */}
      <Dialog open={showUploadModal} onOpenChange={setShowUploadModal}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Upload Documents</DialogTitle>
          </DialogHeader>
          <DocumentUpload
            onUploadComplete={handleUploadComplete}
            onUploadError={handleUploadError}
            maxFiles={10}
          />
        </DialogContent>
      </Dialog>

      {/* Document Preview */}
      <DocumentPreview
        document={previewDocument}
        isOpen={!!previewDocument}
        onClose={() => setPreviewDocument(null)}
        onEdit={handleEditDocument}
        onDownload={handleDownloadDocument}
        onDelete={handleDeleteDocument}
      />
    </div>
  );
}