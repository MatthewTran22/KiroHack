import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-client';
import { apiClient } from '@/lib/api';
import { tokenManager } from '@/lib/auth';
import { useDocumentStore } from '@/stores/documents';
import { Document, PaginatedResponse } from '@/types';
import type { DocumentFilters, DocumentSortOption } from '@/stores/documents';

// Extended API client for documents
const documentsAPI = {
  async getDocuments(filters?: DocumentFilters, sort?: DocumentSortOption): Promise<PaginatedResponse<Document>> {
    const params = new URLSearchParams();
    
    if (filters?.searchQuery) params.append('search', filters.searchQuery);
    if (filters?.category) params.append('category', filters.category);
    if (filters?.classification) params.append('classification', filters.classification);
    if (filters?.status) params.append('status', filters.status);
    if (filters?.tags?.length) params.append('tags', filters.tags.join(','));
    if (filters?.dateRange) {
      params.append('startDate', filters.dateRange.start.toISOString());
      params.append('endDate', filters.dateRange.end.toISOString());
    }
    if (sort) {
      params.append('sortBy', sort.field);
      params.append('sortOrder', sort.direction);
    }

    const token = tokenManager.getToken();
    const response = await fetch(`http://localhost:8080/api/v1/documents?${params.toString()}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch documents');
    }

    return response.json();
  },

  async getDocument(id: string): Promise<Document> {
    const token = tokenManager.getToken();
    const response = await fetch(`http://localhost:8080/api/v1/documents/${id}`, {
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch document');
    }

    return response.json();
  },

  async uploadDocuments(files: File[], metadata: Record<string, unknown>[]): Promise<Document[]> {
    const formData = new FormData();
    
    files.forEach((file, index) => {
      formData.append(`files`, file);
      formData.append(`metadata_${index}`, JSON.stringify(metadata[index] || {}));
    });

    const response = await fetch('/api/documents/upload', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error('Failed to upload documents');
    }

    return response.json();
  },

  async updateDocument(id: string, updates: Partial<Document>): Promise<Document> {
    const response = await fetch(`/api/documents/${id}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify(updates),
    });

    if (!response.ok) {
      throw new Error('Failed to update document');
    }

    return response.json();
  },

  async deleteDocument(id: string): Promise<void> {
    const response = await fetch(`/api/documents/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to delete document');
    }
  },

  async deleteDocuments(ids: string[]): Promise<void> {
    const response = await fetch('/api/documents/batch', {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
      body: JSON.stringify({ ids }),
    });

    if (!response.ok) {
      throw new Error('Failed to delete documents');
    }
  },

  async searchDocuments(query: string): Promise<Document[]> {
    const response = await fetch(`/api/documents/search?q=${encodeURIComponent(query)}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to search documents');
    }

    return response.json();
  },
};

// Hook for fetching documents with filters and sorting
export function useDocuments() {
  const { filters, sortBy } = useDocumentStore();
  
  return useQuery({
    queryKey: queryKeys.documents.list({ filters, sortBy }),
    queryFn: () => documentsAPI.getDocuments(filters, sortBy),
    staleTime: 2 * 60 * 1000, // 2 minutes
  });
}

// Hook for fetching a single document
export function useDocument(id: string) {
  return useQuery({
    queryKey: queryKeys.documents.detail(id),
    queryFn: () => documentsAPI.getDocument(id),
    enabled: !!id,
  });
}

// Hook for document search
export function useDocumentSearch(query: string) {
  return useQuery({
    queryKey: queryKeys.documents.search(query),
    queryFn: () => documentsAPI.searchDocuments(query),
    enabled: query.length > 2,
    staleTime: 30 * 1000, // 30 seconds
  });
}

// Hook for uploading documents with optimistic updates
export function useUploadDocuments() {
  const queryClient = useQueryClient();
  const { updateUploadProgress, completeUpload, failUpload } = useDocumentStore();

  return useMutation({
    mutationFn: async ({ files, metadata }: { files: File[]; metadata: Record<string, unknown>[] }) => {
      // Start upload progress tracking
      files.forEach((file, index) => {
        const fileId = `upload_${index}_${Date.now()}`;
        updateUploadProgress(fileId, {
          fileId,
          fileName: file.name,
          progress: 0,
          status: 'uploading',
        });
      });

      return documentsAPI.uploadDocuments(files, metadata);
    },
    onSuccess: (documents) => {
      // Update upload progress
      documents.forEach((doc, index) => {
        const fileId = `upload_${index}_${Date.now()}`;
        completeUpload(fileId, doc);
      });

      // Invalidate and refetch documents list
      queryClient.invalidateQueries({ queryKey: queryKeys.documents.lists() });
      
      // Optimistically add documents to cache
      queryClient.setQueryData(
        queryKeys.documents.lists(),
        (oldData: PaginatedResponse<Document> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: [...documents, ...oldData.data],
            pagination: {
              ...oldData.pagination,
              total: oldData.pagination.total + documents.length,
            },
          };
        }
      );
    },
    onError: (error, { files }) => {
      // Update upload progress with error
      files.forEach((file, index) => {
        const fileId = `upload_${index}_${Date.now()}`;
        failUpload(fileId, error instanceof Error ? error.message : 'Upload failed');
      });
    },
  });
}

// Hook for updating a document
export function useUpdateDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<Document> }) =>
      documentsAPI.updateDocument(id, updates),
    onSuccess: (updatedDocument) => {
      // Update the document in cache
      queryClient.setQueryData(
        queryKeys.documents.detail(updatedDocument.id),
        updatedDocument
      );

      // Update the document in lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.documents.lists() },
        (oldData: PaginatedResponse<Document> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: oldData.data.map((doc) =>
              doc.id === updatedDocument.id ? updatedDocument : doc
            ),
          };
        }
      );
    },
  });
}

// Hook for deleting a document
export function useDeleteDocument() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: documentsAPI.deleteDocument,
    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: queryKeys.documents.detail(deletedId) });

      // Remove from lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.documents.lists() },
        (oldData: PaginatedResponse<Document> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: oldData.data.filter((doc) => doc.id !== deletedId),
            pagination: {
              ...oldData.pagination,
              total: oldData.pagination.total - 1,
            },
          };
        }
      );
    },
  });
}

// Hook for bulk deleting documents
export function useDeleteDocuments() {
  const queryClient = useQueryClient();
  const { selectedDocuments, clearSelection } = useDocumentStore();

  return useMutation({
    mutationFn: (ids?: string[]) => documentsAPI.deleteDocuments(ids || selectedDocuments),
    onSuccess: (_, deletedIds) => {
      const idsToDelete = deletedIds || selectedDocuments;
      
      // Remove from cache
      idsToDelete.forEach((id) => {
        queryClient.removeQueries({ queryKey: queryKeys.documents.detail(id) });
      });

      // Remove from lists
      queryClient.setQueriesData(
        { queryKey: queryKeys.documents.lists() },
        (oldData: PaginatedResponse<Document> | undefined) => {
          if (!oldData) return oldData;
          return {
            ...oldData,
            data: oldData.data.filter((doc) => !idsToDelete.includes(doc.id)),
            pagination: {
              ...oldData.pagination,
              total: oldData.pagination.total - idsToDelete.length,
            },
          };
        }
      );

      // Clear selection
      clearSelection();
    },
  });
}