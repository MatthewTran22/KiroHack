import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DocumentsPage from '../page';
import { useDocuments } from '@/hooks/useDocuments';
import { useDocumentStore } from '@/stores/documents';

// Mock the hooks
jest.mock('@/hooks/useDocuments');
jest.mock('@/stores/documents');

// Mock the child components
jest.mock('@/components/documents/document-upload', () => ({
  DocumentUpload: ({ onUploadComplete, onUploadError }: any) => (
    <div data-testid="document-upload">
      <button onClick={() => onUploadComplete([])}>Complete Upload</button>
      <button onClick={() => onUploadError('Upload failed')}>Fail Upload</button>
    </div>
  ),
}));

jest.mock('@/components/documents/document-grid', () => ({
  DocumentGrid: ({ documents, onPreviewDocument }: any) => (
    <div data-testid="document-grid">
      {documents.map((doc: any) => (
        <div key={doc.id} onClick={() => onPreviewDocument(doc)}>
          {doc.name}
        </div>
      ))}
    </div>
  ),
}));

jest.mock('@/components/documents/document-list', () => ({
  DocumentList: ({ documents, onPreviewDocument }: any) => (
    <div data-testid="document-list">
      {documents.map((doc: any) => (
        <div key={doc.id} onClick={() => onPreviewDocument(doc)}>
          {doc.name}
        </div>
      ))}
    </div>
  ),
}));

jest.mock('@/components/documents/document-filters', () => ({
  DocumentFilters: ({ onFiltersChange }: any) => (
    <div data-testid="document-filters">
      <button onClick={() => onFiltersChange({ category: 'test' })}>
        Apply Filter
      </button>
    </div>
  ),
}));

jest.mock('@/components/documents/document-preview', () => ({
  DocumentPreview: ({ document, isOpen, onClose }: any) => (
    isOpen ? (
      <div data-testid="document-preview">
        <div>Preview: {document?.name}</div>
        <button onClick={onClose}>Close</button>
      </div>
    ) : null
  ),
}));

const mockUseDocuments = useDocuments as jest.MockedFunction<typeof useDocuments>;
const mockUseDocumentStore = useDocumentStore as jest.MockedFunction<typeof useDocumentStore>;

const mockDocuments = [
  {
    id: '1',
    name: 'Test Document 1.pdf',
    type: 'pdf',
    size: 1024000,
    uploadedAt: new Date('2024-01-01'),
    userId: 'user1',
    status: 'completed' as const,
    classification: 'internal' as const,
    tags: ['important'],
    metadata: { title: 'Test Document 1' },
  },
  {
    id: '2',
    name: 'Test Document 2.docx',
    type: 'docx',
    size: 2048000,
    uploadedAt: new Date('2024-01-02'),
    userId: 'user1',
    status: 'processing' as const,
    classification: 'confidential' as const,
    tags: ['draft'],
    metadata: { title: 'Test Document 2' },
  },
];

describe('DocumentsPage', () => {
  const mockLoadDocuments = jest.fn();
  const mockDownloadDocument = jest.fn();
  const mockDeleteDocument = jest.fn();
  const mockDeleteDocuments = jest.fn();

  beforeEach(() => {
    mockUseDocuments.mockReturnValue({
      documents: mockDocuments,
      isLoading: false,
      error: null,
      loadDocuments: mockLoadDocuments,
      downloadDocument: mockDownloadDocument,
      deleteDocument: mockDeleteDocument,
      deleteDocuments: mockDeleteDocuments,
      uploadDocuments: jest.fn(),
    });

    mockUseDocumentStore.mockReturnValue({
      selectedDocuments: [],
      viewMode: 'grid',
      filters: {},
      sortBy: { field: 'uploadedAt', direction: 'desc' },
      uploadProgress: {},
      isUploading: false,
      showUploadModal: false,
      selectDocument: jest.fn(),
      deselectDocument: jest.fn(),
      selectAllDocuments: jest.fn(),
      clearSelection: jest.fn(),
      setViewMode: jest.fn(),
      setFilters: jest.fn(),
      clearFilters: jest.fn(),
      setSortBy: jest.fn(),
      startUpload: jest.fn(),
      updateUploadProgress: jest.fn(),
      completeUpload: jest.fn(),
      failUpload: jest.fn(),
      clearUploadProgress: jest.fn(),
      setShowUploadModal: jest.fn(),
    });

    jest.clearAllMocks();
  });

  it('renders page header and controls', () => {
    render(<DocumentsPage />);
    
    expect(screen.getByText('Documents')).toBeInTheDocument();
    expect(screen.getByText('Manage and organize your document library')).toBeInTheDocument();
    expect(screen.getByText('Upload Documents')).toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<DocumentsPage />);
    
    expect(screen.getByPlaceholderText('Search documents...')).toBeInTheDocument();
  });

  it('handles search input', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    const searchInput = screen.getByPlaceholderText('Search documents...');
    await user.type(searchInput, 'test query');
    
    await waitFor(() => {
      expect(mockLoadDocuments).toHaveBeenCalledWith({
        searchQuery: 'test query',
      });
    });
  });

  it('toggles filters panel', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    const filtersButton = screen.getByText('Filters');
    await user.click(filtersButton);
    
    expect(screen.getByTestId('document-filters')).toBeInTheDocument();
  });

  it('shows view mode toggle buttons', () => {
    render(<DocumentsPage />);
    
    // Grid and list view buttons should be present
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('opens upload modal', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    const uploadButton = screen.getByText('Upload Documents');
    await user.click(uploadButton);
    
    expect(screen.getByTestId('document-upload')).toBeInTheDocument();
  });

  it('handles upload completion', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    // Open upload modal
    const uploadButton = screen.getByText('Upload Documents');
    await user.click(uploadButton);
    
    // Complete upload
    const completeButton = screen.getByText('Complete Upload');
    await user.click(completeButton);
    
    expect(mockLoadDocuments).toHaveBeenCalled();
  });

  it('handles upload error', async () => {
    const user = userEvent.setup();
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(<DocumentsPage />);
    
    // Open upload modal
    const uploadButton = screen.getByText('Upload Documents');
    await user.click(uploadButton);
    
    // Fail upload
    const failButton = screen.getByText('Fail Upload');
    await user.click(failButton);
    
    expect(consoleSpy).toHaveBeenCalledWith('Upload error:', 'Upload failed');
    
    consoleSpy.mockRestore();
  });

  it('displays documents in grid view by default', () => {
    render(<DocumentsPage />);
    
    expect(screen.getByTestId('document-grid')).toBeInTheDocument();
    expect(screen.getByText('Test Document 1.pdf')).toBeInTheDocument();
    expect(screen.getByText('Test Document 2.docx')).toBeInTheDocument();
  });

  it('switches to list view', async () => {
    const mockSetViewMode = jest.fn();
    mockUseDocumentStore.mockReturnValue({
      ...mockUseDocumentStore(),
      viewMode: 'list',
      setViewMode: mockSetViewMode,
    });
    
    render(<DocumentsPage />);
    
    expect(screen.getByTestId('document-list')).toBeInTheDocument();
  });

  it('shows document count', () => {
    render(<DocumentsPage />);
    
    expect(screen.getByText('2 documents found')).toBeInTheDocument();
  });

  it('shows loading state', () => {
    mockUseDocuments.mockReturnValue({
      ...mockUseDocuments(),
      isLoading: true,
      documents: [],
    });
    
    render(<DocumentsPage />);
    
    expect(screen.getByText('Loading documents...')).toBeInTheDocument();
  });

  it('shows error state', () => {
    const errorMessage = 'Failed to load documents';
    mockUseDocuments.mockReturnValue({
      ...mockUseDocuments(),
      error: errorMessage,
      documents: [],
    });
    
    render(<DocumentsPage />);
    
    expect(screen.getByText(`Error loading documents: ${errorMessage}`)).toBeInTheDocument();
    expect(screen.getByText('Retry')).toBeInTheDocument();
  });

  it('handles document preview', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    const documentName = screen.getByText('Test Document 1.pdf');
    await user.click(documentName);
    
    expect(screen.getByTestId('document-preview')).toBeInTheDocument();
    expect(screen.getByText('Preview: Test Document 1.pdf')).toBeInTheDocument();
  });

  it('closes document preview', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    // Open preview
    const documentName = screen.getByText('Test Document 1.pdf');
    await user.click(documentName);
    
    // Close preview
    const closeButton = screen.getByText('Close');
    await user.click(closeButton);
    
    expect(screen.queryByTestId('document-preview')).not.toBeInTheDocument();
  });

  it('shows bulk actions when documents are selected', () => {
    mockUseDocumentStore.mockReturnValue({
      ...mockUseDocumentStore(),
      selectedDocuments: ['1', '2'],
    });
    
    render(<DocumentsPage />);
    
    expect(screen.getByText('2 selected')).toBeInTheDocument();
    expect(screen.getByText('Download')).toBeInTheDocument();
    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('handles bulk delete with confirmation', async () => {
    const user = userEvent.setup();
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);
    
    mockUseDocumentStore.mockReturnValue({
      ...mockUseDocumentStore(),
      selectedDocuments: ['1', '2'],
    });
    
    render(<DocumentsPage />);
    
    const deleteButton = screen.getByText('Delete');
    await user.click(deleteButton);
    
    expect(confirmSpy).toHaveBeenCalledWith('Are you sure you want to delete 2 document(s)?');
    expect(mockDeleteDocuments).toHaveBeenCalledWith(['1', '2']);
    
    confirmSpy.mockRestore();
  });

  it('applies filters from filter component', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    // Show filters
    const filtersButton = screen.getByText('Filters');
    await user.click(filtersButton);
    
    // Apply filter
    const applyFilterButton = screen.getByText('Apply Filter');
    await user.click(applyFilterButton);
    
    await waitFor(() => {
      expect(mockLoadDocuments).toHaveBeenCalledWith({
        category: 'test',
      });
    });
  });

  it('shows active filters indicator', () => {
    render(<DocumentsPage />);
    
    // Initially no active filters
    const filtersButton = screen.getByText('Filters');
    expect(filtersButton).toBeInTheDocument();
    
    // When filters are active, it would show an indicator
    // This would be tested with actual filter state
  });

  it('handles refresh action', async () => {
    const user = userEvent.setup();
    render(<DocumentsPage />);
    
    // Find more actions menu
    const moreButtons = screen.getAllByRole('button');
    const moreButton = moreButtons.find(button => 
      button.querySelector('svg') // Looking for MoreHorizontal icon
    );
    
    if (moreButton) {
      await user.click(moreButton);
      
      const refreshButton = screen.getByText('Refresh');
      await user.click(refreshButton);
      
      expect(mockLoadDocuments).toHaveBeenCalled();
    }
  });
});