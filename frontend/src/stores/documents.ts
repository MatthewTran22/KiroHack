import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { Document } from '@/types';

// Document-specific types
export interface DocumentFilters {
  category?: string;
  dateRange?: {
    start: Date;
    end: Date;
  };
  classification?: 'public' | 'internal' | 'confidential' | 'restricted';
  tags?: string[];
  searchQuery?: string;
  status?: 'processing' | 'ready' | 'error';
}

export interface DocumentSortOption {
  field: 'name' | 'uploadedAt' | 'size' | 'type';
  direction: 'asc' | 'desc';
}

export interface DocumentUploadProgress {
  fileId: string;
  fileName: string;
  progress: number;
  status: 'pending' | 'uploading' | 'processing' | 'complete' | 'error';
  error?: string;
}

interface DocumentState {
  // Selection state
  selectedDocuments: string[];
  
  // View state
  viewMode: 'grid' | 'list';
  filters: DocumentFilters;
  sortBy: DocumentSortOption;
  
  // Upload state
  uploadProgress: Record<string, DocumentUploadProgress>;
  
  // UI state
  isUploading: boolean;
  showUploadModal: boolean;
}

interface DocumentActions {
  // Selection actions
  selectDocument: (id: string) => void;
  deselectDocument: (id: string) => void;
  selectAllDocuments: (documentIds: string[]) => void;
  clearSelection: () => void;
  
  // View actions
  setViewMode: (mode: 'grid' | 'list') => void;
  setFilters: (filters: Partial<DocumentFilters>) => void;
  clearFilters: () => void;
  setSortBy: (sortBy: DocumentSortOption) => void;
  
  // Upload actions
  startUpload: (files: File[]) => void;
  updateUploadProgress: (fileId: string, progress: Partial<DocumentUploadProgress>) => void;
  completeUpload: (fileId: string, document?: Document) => void;
  failUpload: (fileId: string, error: string) => void;
  clearUploadProgress: () => void;
  
  // UI actions
  setShowUploadModal: (show: boolean) => void;
}

type DocumentStore = DocumentState & DocumentActions;

const initialState: DocumentState = {
  selectedDocuments: [],
  viewMode: 'grid',
  filters: {},
  sortBy: { field: 'uploadedAt', direction: 'desc' },
  uploadProgress: {},
  isUploading: false,
  showUploadModal: false,
};

export const useDocumentStore = create<DocumentStore>()(
  devtools(
    (set, get) => ({
      ...initialState,

      // Selection actions
      selectDocument: (id: string) => {
        set((state) => ({
          selectedDocuments: state.selectedDocuments.includes(id)
            ? state.selectedDocuments
            : [...state.selectedDocuments, id],
        }));
      },

      deselectDocument: (id: string) => {
        set((state) => ({
          selectedDocuments: state.selectedDocuments.filter((docId) => docId !== id),
        }));
      },

      selectAllDocuments: (documentIds: string[]) => {
        set({ selectedDocuments: documentIds });
      },

      clearSelection: () => {
        set({ selectedDocuments: [] });
      },

      // View actions
      setViewMode: (mode: 'grid' | 'list') => {
        set({ viewMode: mode });
      },

      setFilters: (newFilters: Partial<DocumentFilters>) => {
        set((state) => ({
          filters: { ...state.filters, ...newFilters },
        }));
      },

      clearFilters: () => {
        set({ filters: {} });
      },

      setSortBy: (sortBy: DocumentSortOption) => {
        set({ sortBy });
      },

      // Upload actions
      startUpload: (files: File[]) => {
        const uploadProgress: Record<string, DocumentUploadProgress> = {};
        
        files.forEach((file) => {
          const fileId = crypto.randomUUID();
          uploadProgress[fileId] = {
            fileId,
            fileName: file.name,
            progress: 0,
            status: 'pending',
          };
        });

        set({
          uploadProgress,
          isUploading: true,
          showUploadModal: true,
        });
      },

      updateUploadProgress: (fileId: string, progress: Partial<DocumentUploadProgress>) => {
        set((state) => ({
          uploadProgress: {
            ...state.uploadProgress,
            [fileId]: {
              ...state.uploadProgress[fileId],
              ...progress,
            },
          },
        }));
      },

      completeUpload: (fileId: string, document?: Document) => {
        set((state) => {
          const newProgress = { ...state.uploadProgress };
          if (newProgress[fileId]) {
            newProgress[fileId] = {
              ...newProgress[fileId],
              progress: 100,
              status: 'complete',
            };
          }

          const isStillUploading = Object.values(newProgress).some(
            (p) => p.status === 'pending' || p.status === 'uploading' || p.status === 'processing'
          );

          return {
            uploadProgress: newProgress,
            isUploading: isStillUploading,
          };
        });
      },

      failUpload: (fileId: string, error: string) => {
        set((state) => ({
          uploadProgress: {
            ...state.uploadProgress,
            [fileId]: {
              ...state.uploadProgress[fileId],
              status: 'error',
              error,
            },
          },
        }));
      },

      clearUploadProgress: () => {
        set({
          uploadProgress: {},
          isUploading: false,
        });
      },

      // UI actions
      setShowUploadModal: (show: boolean) => {
        set({ showUploadModal: show });
      },
    }),
    {
      name: 'document-store',
    }
  )
);