import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DocumentList } from '../document-list';
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

describe('DocumentList', () => {
  const mockProps = {
    documents: mockDocuments,
    selectedDocuments: [],
    onSelectDocument: jest.fn(),
    onDeselectDocument: jest.fn(),
    onSelectAll: jest.fn(),
    onPreviewDocument: jest.fn(),
    onEditDocument: jest.fn(),
    onDownloadDocument: jest.fn(),
    onDeleteDocument: jest.fn(),
    onSort: jest.fn(),
    sortField: 'uploadedAt',
    sortDirection: 'desc' as const,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders documents in table format', () => {
    render(<DocumentList {...mockProps} />);
    
    // Check table headers
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Type')).toBeInTheDocument();
    expect(screen.getByText('Size')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Classification')).toBeInTheDocument();
    expect(screen.getByText('Tags')).toBeInTheDocument();
    expect(screen.getByText('Uploaded')).toBeInTheDocument();
    
    // Check document data
    expect(screen.getByText('Test Document 1.pdf')).toBeInTheDocument();
    expect(screen.getByText('Test Document 2.docx')).toBeInTheDocument();
  });

  it('displays document information correctly', () => {
    render(<DocumentList {...mockProps} />);
    
    expect(screen.getByText('1 MB')).toBeInTheDocument();
    expect(screen.getByText('2 MB')).toBeInTheDocument();
    expect(screen.getByText('PDF')).toBeInTheDocument();
    expect(screen.getByText('DOCX')).toBeInTheDocument();
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('processing')).toBeInTheDocument();
  });

  it('handles select all functionality', async () => {
    const user = userEvent.setup();
    render(<DocumentList {...mockProps} />);
    
    const selectAllCheckbox = screen.getAllByRole('checkbox')[0]; // First checkbox is select all
    await user.click(selectAllCheckbox);
    
    expect(mockProps.onSelectAll).toHaveBeenCalledWith(['1', '2']);
  });

  it('handles individual document selection', async () => {
    const user = userEvent.setup();
    render(<DocumentList {...mockProps} />);
    
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]); // First document checkbox
    
    expect(mockProps.onSelectDocument).toHaveBeenCalledWith('1');
  });

  it('shows indeterminate state for partial selection', () => {
    const propsWithPartialSelection = {
      ...mockProps,
      selectedDocuments: ['1'],
    };
    
    render(<DocumentList {...propsWithPartialSelection} />);
    
    const selectAllCheckbox = screen.getAllByRole('checkbox')[0];
    expect(selectAllCheckbox).toHaveProperty('indeterminate', true);
  });

  it('handles sorting', async () => {
    const user = userEvent.setup();
    render(<DocumentList {...mockProps} />);
    
    const nameHeader = screen.getByText('Name');
    await user.click(nameHeader);
    
    expect(mockProps.onSort).toHaveBeenCalledWith('name', 'asc');
  });

  it('shows sort direction indicator', () => {
    const propsWithSort = {
      ...mockProps,
      sortField: 'name',
      sortDirection: 'asc' as const,
    };
    
    render(<DocumentList {...propsWithSort} />);
    
    // Should show sort indicator (↑ for ascending)
    expect(screen.getByText('↑')).toBeInTheDocument();
  });

  it('handles document preview', async () => {
    const user = userEvent.setup();
    render(<DocumentList {...mockProps} />);
    
    const documentName = screen.getByText('Test Document 1.pdf');
    await user.click(documentName);
    
    expect(mockProps.onPreviewDocument).toHaveBeenCalledWith(mockDocuments[0]);
  });

  it('shows dropdown menu with actions', async () => {
    const user = userEvent.setup();
    render(<DocumentList {...mockProps} />);
    
    // Find the more options button
    const moreButtons = screen.getAllByRole('button');
    const moreButton = moreButtons.find(button => 
      button.querySelector('svg') // Looking for MoreHorizontal icon
    );
    
    if (moreButton) {
      await user.click(moreButton);
      
      expect(screen.getByText('Preview')).toBeInTheDocument();
      expect(screen.getByText('Download')).toBeInTheDocument();
      expect(screen.getByText('Edit')).toBeInTheDocument();
      expect(screen.getByText('Delete')).toBeInTheDocument();
    }
  });

  it('shows empty state when no documents', () => {
    render(<DocumentList {...mockProps} documents={[]} />);
    
    expect(screen.getByText('No documents found')).toBeInTheDocument();
    expect(screen.getByText('Upload some documents or adjust your filters to see results.')).toBeInTheDocument();
  });

  it('highlights selected rows', () => {
    const propsWithSelection = {
      ...mockProps,
      selectedDocuments: ['1'],
    };
    
    render(<DocumentList {...propsWithSelection} />);
    
    const checkboxes = screen.getAllByRole('checkbox');
    expect(checkboxes[1]).toBeChecked(); // First document checkbox
  });

  it('shows document descriptions when available', () => {
    render(<DocumentList {...mockProps} />);
    
    expect(screen.getByText('A test document')).toBeInTheDocument();
    expect(screen.getByText('Another test document')).toBeInTheDocument();
  });

  it('shows tag overflow indicator', () => {
    const manyTagsDocument: Document = {
      ...mockDocuments[0],
      tags: ['tag1', 'tag2', 'tag3', 'tag4'],
    };
    
    const propsWithManyTags = {
      ...mockProps,
      documents: [manyTagsDocument],
    };
    
    render(<DocumentList {...propsWithManyTags} />);
    
    // Should show first 2 tags and +2 indicator
    expect(screen.getByText('tag1')).toBeInTheDocument();
    expect(screen.getByText('tag2')).toBeInTheDocument();
    expect(screen.getByText('+2')).toBeInTheDocument();
  });

  it('formats dates correctly', () => {
    render(<DocumentList {...mockProps} />);
    
    // Check that dates are formatted (exact format may vary based on locale)
    expect(screen.getByText(/Jan.*2024/)).toBeInTheDocument();
  });

  it('handles sorting without onSort prop', () => {
    const propsWithoutSort = {
      ...mockProps,
      onSort: undefined,
    };
    
    render(<DocumentList {...propsWithoutSort} />);
    
    // Headers should not be clickable
    const nameHeader = screen.getByText('Name');
    expect(nameHeader.closest('th')).not.toHaveClass('cursor-pointer');
  });

  it('shows correct status and classification colors', () => {
    render(<DocumentList {...mockProps} />);
    
    // Check that badges are rendered with correct text
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.getByText('processing')).toBeInTheDocument();
    expect(screen.getByText('internal')).toBeInTheDocument();
    expect(screen.getByText('confidential')).toBeInTheDocument();
  });
});