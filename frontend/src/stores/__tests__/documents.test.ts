import { act, renderHook } from '@testing-library/react';
import { useDocumentStore } from '../documents';
import type { DocumentFilters, DocumentSortOption } from '../documents';

describe('Document Store', () => {
  beforeEach(() => {
    // Reset store state before each test
    useDocumentStore.setState({
      selectedDocuments: [],
      viewMode: 'grid',
      filters: {},
      sortBy: { field: 'uploadedAt', direction: 'desc' },
      uploadProgress: {},
      isUploading: false,
      showUploadModal: false,
    });
  });

  describe('Selection Management', () => {
    it('should select a document', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.selectDocument('doc-1');
      });

      expect(result.current.selectedDocuments).toEqual(['doc-1']);
    });

    it('should not duplicate selected documents', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.selectDocument('doc-1');
        result.current.selectDocument('doc-1');
      });

      expect(result.current.selectedDocuments).toEqual(['doc-1']);
    });

    it('should select multiple documents', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.selectDocument('doc-1');
        result.current.selectDocument('doc-2');
      });

      expect(result.current.selectedDocuments).toEqual(['doc-1', 'doc-2']);
    });

    it('should deselect a document', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.selectDocument('doc-1');
        result.current.selectDocument('doc-2');
        result.current.deselectDocument('doc-1');
      });

      expect(result.current.selectedDocuments).toEqual(['doc-2']);
    });

    it('should select all documents', () => {
      const { result } = renderHook(() => useDocumentStore());
      const documentIds = ['doc-1', 'doc-2', 'doc-3'];

      act(() => {
        result.current.selectAllDocuments(documentIds);
      });

      expect(result.current.selectedDocuments).toEqual(documentIds);
    });

    it('should clear selection', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.selectDocument('doc-1');
        result.current.selectDocument('doc-2');
        result.current.clearSelection();
      });

      expect(result.current.selectedDocuments).toEqual([]);
    });
  });

  describe('View Management', () => {
    it('should set view mode', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.setViewMode('list');
      });

      expect(result.current.viewMode).toBe('list');
    });

    it('should set filters', () => {
      const { result } = renderHook(() => useDocumentStore());
      const filters: Partial<DocumentFilters> = {
        category: 'policy',
        searchQuery: 'test',
      };

      act(() => {
        result.current.setFilters(filters);
      });

      expect(result.current.filters).toEqual(filters);
    });

    it('should merge filters', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.setFilters({ category: 'policy' });
        result.current.setFilters({ searchQuery: 'test' });
      });

      expect(result.current.filters).toEqual({
        category: 'policy',
        searchQuery: 'test',
      });
    });

    it('should clear filters', () => {
      const { result } = renderHook(() => useDocumentStore());

      act(() => {
        result.current.setFilters({ category: 'policy', searchQuery: 'test' });
        result.current.clearFilters();
      });

      expect(result.current.filters).toEqual({});
    });

    it('should set sort options', () => {
      const { result } = renderHook(() => useDocumentStore());
      const sortBy: DocumentSortOption = { field: 'name', direction: 'asc' };

      act(() => {
        result.current.setSortBy(sortBy);
      });

      expect(result.current.sortBy).toEqual(sortBy);
    });
  });

  describe('Upload Management', () => {
    it('should start upload', () => {
      const { result } = renderHook(() => useDocumentStore());
      const files = [
        new File(['content1'], 'file1.txt', { type: 'text/plain' }),
        new File(['content2'], 'file2.txt', { type: 'text/plain' }),
      ];

      act(() => {
        result.current.startUpload(files);
      });

      expect(result.current.isUploading).toBe(true);
      expect(result.current.showUploadModal).toBe(true);
      expect(Object.keys(result.current.uploadProgress)).toHaveLength(2);

      const progressEntries = Object.values(result.current.uploadProgress);
      expect(progressEntries[0]).toMatchObject({
        fileName: 'file1.txt',
        progress: 0,
        status: 'pending',
      });
      expect(progressEntries[1]).toMatchObject({
        fileName: 'file2.txt',
        progress: 0,
        status: 'pending',
      });
    });

    it('should update upload progress', () => {
      const { result } = renderHook(() => useDocumentStore());
      const fileId = 'test-file-id';

      // Set initial progress
      act(() => {
        result.current.updateUploadProgress(fileId, {
          fileId,
          fileName: 'test.txt',
          progress: 0,
          status: 'pending',
        });
      });

      // Update progress
      act(() => {
        result.current.updateUploadProgress(fileId, {
          progress: 50,
          status: 'uploading',
        });
      });

      expect(result.current.uploadProgress[fileId]).toMatchObject({
        fileId,
        fileName: 'test.txt',
        progress: 50,
        status: 'uploading',
      });
    });

    it('should complete upload', () => {
      const { result } = renderHook(() => useDocumentStore());
      const fileId = 'test-file-id';

      // Set initial progress
      act(() => {
        result.current.updateUploadProgress(fileId, {
          fileId,
          fileName: 'test.txt',
          progress: 0,
          status: 'uploading',
        });
      });

      // Complete upload
      act(() => {
        result.current.completeUpload(fileId);
      });

      expect(result.current.uploadProgress[fileId]).toMatchObject({
        progress: 100,
        status: 'complete',
      });
      expect(result.current.isUploading).toBe(false);
    });

    it('should handle upload failure', () => {
      const { result } = renderHook(() => useDocumentStore());
      const fileId = 'test-file-id';
      const error = 'Upload failed';

      // Set initial progress
      act(() => {
        result.current.updateUploadProgress(fileId, {
          fileId,
          fileName: 'test.txt',
          progress: 50,
          status: 'uploading',
        });
      });

      // Fail upload
      act(() => {
        result.current.failUpload(fileId, error);
      });

      expect(result.current.uploadProgress[fileId]).toMatchObject({
        status: 'error',
        error,
      });
    });

    it('should track uploading state correctly', () => {
      const { result } = renderHook(() => useDocumentStore());
      const fileId1 = 'file-1';
      const fileId2 = 'file-2';

      // Start uploads - need to set isUploading to true first
      act(() => {
        result.current.updateUploadProgress(fileId1, {
          fileId: fileId1,
          fileName: 'file1.txt',
          progress: 0,
          status: 'uploading',
        });
        result.current.updateUploadProgress(fileId2, {
          fileId: fileId2,
          fileName: 'file2.txt',
          progress: 0,
          status: 'uploading',
        });
        // Manually set uploading state since we're not using startUpload
        result.current.setShowUploadModal(true);
      });

      // Set uploading state manually for this test
      act(() => {
        useDocumentStore.setState({ isUploading: true });
      });

      expect(result.current.isUploading).toBe(true);

      // Complete first upload
      act(() => {
        result.current.completeUpload(fileId1);
      });

      expect(result.current.isUploading).toBe(true); // Still uploading second file

      // Complete second upload
      act(() => {
        result.current.completeUpload(fileId2);
      });

      expect(result.current.isUploading).toBe(false); // All uploads complete
    });

    it('should clear upload progress', () => {
      const { result } = renderHook(() => useDocumentStore());

      // Set some progress
      act(() => {
        result.current.updateUploadProgress('file-1', {
          fileId: 'file-1',
          fileName: 'test.txt',
          progress: 50,
          status: 'uploading',
        });
      });

      expect(Object.keys(result.current.uploadProgress)).toHaveLength(1);

      // Clear progress
      act(() => {
        result.current.clearUploadProgress();
      });

      expect(result.current.uploadProgress).toEqual({});
      expect(result.current.isUploading).toBe(false);
    });

    it('should toggle upload modal', () => {
      const { result } = renderHook(() => useDocumentStore());

      expect(result.current.showUploadModal).toBe(false);

      act(() => {
        result.current.setShowUploadModal(true);
      });

      expect(result.current.showUploadModal).toBe(true);

      act(() => {
        result.current.setShowUploadModal(false);
      });

      expect(result.current.showUploadModal).toBe(false);
    });
  });

  describe('Store Persistence', () => {
    it('should maintain state across hook instances', () => {
      const { result: result1 } = renderHook(() => useDocumentStore());
      
      act(() => {
        result1.current.selectDocument('doc-1');
        result1.current.setViewMode('list');
      });

      const { result: result2 } = renderHook(() => useDocumentStore());
      
      expect(result2.current.selectedDocuments).toEqual(['doc-1']);
      expect(result2.current.viewMode).toBe('list');
    });
  });
});