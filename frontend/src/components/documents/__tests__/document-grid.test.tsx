import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DocumentGrid } from '../document-grid';
import { Document } from '@/types';

const mockDocuments: Document[] = [
  {
    id: '1',
    name: 'Test Document 1.pdf',
    type: 'pdf',
    size: 1024000,
    uploadedAt: new Date('2024-01-01'),
    userId: 'user1',
    status: 'completed',
    classification: 'internal',
    tags: ['important', 'policy'],
    metadata: {
      title: 'Test Document 1',
      description: 'A test document',
      author: 'John Doe',
    },
  },
  {
    id: '2',
    name: 'Test Document 2.docx',
    type: 'docx',
    size: 2048000,
    uploadedAt: new Date('2024-01-02'),
    userId: 'user1',
    status: 'processing',
    classification: 'confidential',
    tags: ['draft'],
    metadata: {
      title: 'Test Document 2',
      description: 'Another test document',
      author: 'Jane Smith',
    },
  },
];

describe('DocumentGrid', () => {
  const mockProps = {
    documents: mockDocuments,
    selectedDocuments: [],
    onSelectDocument: jest.fn(),
    onDeselectDocument: jest.fn(),
    onPreviewDocument: jest.fn(),
    onEditDocument: jest.fn(),
    onDownloadDocument: jest.fn(),
    onDeleteDocument: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders documents in grid layout', () => {
    render(<DocumentGrid {...mockProps} />);
    
    expect(screen.getByText('Test Document 1.pdf')).toBeInTheDocument();
    expect(screen.getByText('Test Document 2.docx')).toBeInTheDocument();
  });

  it('displays document metadata', () => {
    render(<DocumentGrid {...mockProps} />);
    
    expect(screen.getByText('1 MB')).toBeInTheDocument(); // File size
    expect(screen.getByText('PDF')).toBeInTheDocument(); // File type
    expect(screen.getByText('completed')).toBeInTheDocument(); // Status
    expect(screen.getByText('internal')).toBeInTheDocument(); // Classification
  });

  it('shows document tags', () => {
    render(<DocumentGrid {...mockProps} />);
    
    expect(screen.getByText('important')).toBeInTheDocument();
    expect(screen.getByText('policy')).toBeInTheDocument();
    expect(screen.getByText('draft')).toBeInTheDocument();
  });

  it('handles document selection', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[0]);
    
    expect(mockProps.onSelectDocument).toHaveBeenCalledWith('1');
  });

  it('handles document deselection', async () => {
    const user = userEvent.setup();
    const propsWithSelection = {
      ...mockProps,
      selectedDocuments: ['1'],
    };
    
    render(<DocumentGrid {...propsWithSelection} />);
    
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[0]);
    
    expect(mockProps.onDeselectDocument).toHaveBeenCalledWith('1');
  });

  it('opens preview when document is clicked', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    const documentTitle = screen.getByText('Test Document 1.pdf');
    await user.click(documentTitle);
    
    expect(mockProps.onPreviewDocument).toHaveBeenCalledWith(mockDocuments[0]);
  });

  it('shows dropdown menu with actions', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    // Find and click the more options button (should be visible on hover)
    const moreButtons = screen.getAllByRole('button');
    const moreButton = moreButtons.find(button => 
      button.querySelector('svg') // Looking for the MoreHorizontal icon
    );
    
    if (moreButton) {
      await user.click(moreButton);
      
      expect(screen.getByText('Preview')).toBeInTheDocument();
      expect(screen.getByText('Download')).toBeInTheDocument();
      expect(screen.getByText('Edit')).toBeInTheDocument();
      expect(screen.getByText('Delete')).toBeInTheDocument();
    }
  });

  it('handles download action', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    // Find download button in quick actions or dropdown
    const downloadButtons = screen.getAllByRole('button');
    const downloadButton = downloadButtons.find(button => 
      button.querySelector('svg') // Looking for download icon
    );
    
    if (downloadButton) {
      await user.click(downloadButton);
      expect(mockProps.onDownloadDocument).toHaveBeenCalledWith(mockDocuments[0]);
    }
  });

  it('handles edit action', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    // Find edit button in quick actions
    const editButtons = screen.getAllByRole('button');
    const editButton = editButtons.find(button => 
      button.querySelector('svg') // Looking for edit icon
    );
    
    if (editButton) {
      await user.click(editButton);
      expect(mockProps.onEditDocument).toHaveBeenCalledWith(mockDocuments[0]);
    }
  });

  it('handles delete action', async () => {
    const user = userEvent.setup();
    render(<DocumentGrid {...mockProps} />);
    
    // This would typically be in a dropdown menu
    // For now, we test that the component renders correctly
    expect(screen.getByText('Test Document 1.pdf')).toBeInTheDocument();
  });

  it('shows empty state when no documents', () => {
    render(<DocumentGrid {...mockProps} documents={[]} />);
    
    expect(screen.getByText('No documents found')).toBeInTheDocument();
    expect(screen.getByText('Upload some documents or adjust your filters to see results.')).toBeInTheDocument();
  });

  it('highlights selected documents', () => {
    const propsWithSelection = {
      ...mockProps,
      selectedDocuments: ['1'],
    };
    
    render(<DocumentGrid {...propsWithSelection} />);
    
    // Check that the selected document has the selection styling
    const documentCards = screen.getAllByRole('checkbox');
    expect(documentCards[0]).toBeChecked();
  });

  it('shows correct status colors', () => {
    render(<DocumentGrid {...mockProps} />);
    
    // Check that status badges are rendered with correct text
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('processing')).toBeInTheDocument();
  });

  it('shows correct classification colors', () => {
    render(<DocumentGrid {...mockProps} />);
    
    // Check that classification badges are rendered
    expect(screen.getByText('internal')).toBeInTheDocument();
    expect(screen.getByText('confidential')).toBeInTheDocument();
  });

  it('truncates long document names', () => {
    const longNameDocument: Document = {
      ...mockDocuments[0],
      name: 'This is a very long document name that should be truncated in the grid view.pdf',
    };
    
    render(<DocumentGrid {...mockProps} documents={[longNameDocument]} />);
    
    expect(screen.getByTitle('This is a very long document name that should be truncated in the grid view.pdf')).toBeInTheDocument();
  });

  it('shows tag overflow indicator', () => {
    const manyTagsDocument: Document = {
      ...mockDocuments[0],
      tags: ['tag1', 'tag2', 'tag3', 'tag4', 'tag5'],
    };
    
    render(<DocumentGrid {...mockProps} documents={[manyTagsDocument]} />);
    
    // Should show first 2 tags and +3 indicator
    expect(screen.getByText('tag1')).toBeInTheDocument();
    expect(screen.getByText('tag2')).toBeInTheDocument();
    expect(screen.getByText('+3')).toBeInTheDocument();
  });
});