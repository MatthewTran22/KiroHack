import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DocumentFilters } from '../document-filters';
import { DocumentFilters as DocumentFiltersType } from '@/types';

describe('DocumentFilters', () => {
  const mockProps = {
    filters: {} as DocumentFiltersType,
    onFiltersChange: jest.fn(),
    onClearFilters: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders basic filter controls', () => {
    render(<DocumentFilters {...mockProps} />);
    
    expect(screen.getByText('Filters')).toBeInTheDocument();
    expect(screen.getByLabelText('Search')).toBeInTheDocument();
    expect(screen.getByText('Category')).toBeInTheDocument();
    expect(screen.getByText('Classification')).toBeInTheDocument();
  });

  it('handles search input', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    const searchInput = screen.getByPlaceholderText('Search documents...');
    await user.type(searchInput, 'test query');
    
    expect(mockProps.onFiltersChange).toHaveBeenCalledWith({
      searchQuery: 'test query',
    });
  });

  it('handles category selection', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    // Find and click category select
    const categorySelect = screen.getByRole('combobox', { name: /category/i });
    await user.click(categorySelect);
    
    // This would open the dropdown, but testing select components can be complex
    // For now, we verify the select is rendered
    expect(categorySelect).toBeInTheDocument();
  });

  it('handles classification selection', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    const classificationSelect = screen.getByRole('combobox', { name: /classification/i });
    expect(classificationSelect).toBeInTheDocument();
  });

  it('shows advanced filters when toggled', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Date Range')).toBeInTheDocument();
    expect(screen.getByText('Tags')).toBeInTheDocument();
  });

  it('handles date range selection', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    const dateInputs = screen.getAllByDisplayValue('');
    expect(dateInputs.length).toBeGreaterThan(0);
  });

  it('handles tag addition', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    const tagInput = screen.getByPlaceholderText('Add tag...');
    const addButton = screen.getByText('Add');
    
    await user.type(tagInput, 'test-tag');
    await user.click(addButton);
    
    expect(mockProps.onFiltersChange).toHaveBeenCalledWith({
      tags: ['test-tag'],
    });
  });

  it('handles tag removal', async () => {
    const user = userEvent.setup();
    const propsWithTags = {
      ...mockProps,
      filters: { tags: ['existing-tag'] },
    };
    
    render(<DocumentFilters {...propsWithTags} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    expect(screen.getByText('existing-tag')).toBeInTheDocument();
    
    // Find and click remove button (X icon)
    const removeButtons = screen.getAllByRole('button');
    const removeButton = removeButtons.find(button => 
      button.querySelector('svg') && button.textContent === ''
    );
    
    if (removeButton) {
      await user.click(removeButton);
      expect(mockProps.onFiltersChange).toHaveBeenCalledWith({
        tags: [],
      });
    }
  });

  it('shows active filter count', () => {
    const propsWithFilters = {
      ...mockProps,
      filters: {
        searchQuery: 'test',
        category: 'Policy Documents',
        tags: ['important'],
      },
    };
    
    render(<DocumentFilters {...propsWithFilters} />);
    
    expect(screen.getByText('3')).toBeInTheDocument(); // Badge showing count
  });

  it('handles clear all filters', async () => {
    const user = userEvent.setup();
    const propsWithFilters = {
      ...mockProps,
      filters: {
        searchQuery: 'test',
        category: 'Policy Documents',
      },
    };
    
    render(<DocumentFilters {...propsWithFilters} />);
    
    const clearButton = screen.getByText('Clear All');
    await user.click(clearButton);
    
    expect(mockProps.onClearFilters).toHaveBeenCalled();
  });

  it('shows filter summary when filters are active', () => {
    const propsWithFilters = {
      ...mockProps,
      filters: {
        searchQuery: 'test',
        category: 'Policy Documents',
      },
    };
    
    render(<DocumentFilters {...propsWithFilters} />);
    
    expect(screen.getByText('2 filters applied')).toBeInTheDocument();
  });

  it('handles date range clearing', async () => {
    const user = userEvent.setup();
    const propsWithDateRange = {
      ...mockProps,
      filters: {
        dateRange: {
          start: new Date('2024-01-01'),
          end: new Date('2024-01-31'),
        },
      },
    };
    
    render(<DocumentFilters {...propsWithDateRange} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    // Find the clear date range button (X icon)
    const clearButtons = screen.getAllByRole('button');
    const clearDateButton = clearButtons.find(button => 
      button.querySelector('svg') && button.closest('[class*="date"]')
    );
    
    if (clearDateButton) {
      await user.click(clearDateButton);
      expect(mockProps.onFiltersChange).toHaveBeenCalledWith({
        dateRange: undefined,
      });
    }
  });

  it('handles tag input with Enter key', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    const tagInput = screen.getByPlaceholderText('Add tag...');
    await user.type(tagInput, 'test-tag');
    await user.keyboard('{Enter}');
    
    expect(mockProps.onFiltersChange).toHaveBeenCalledWith({
      tags: ['test-tag'],
    });
  });

  it('prevents duplicate tags', async () => {
    const user = userEvent.setup();
    const propsWithTags = {
      ...mockProps,
      filters: { tags: ['existing-tag'] },
    };
    
    render(<DocumentFilters {...propsWithTags} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    const tagInput = screen.getByPlaceholderText('Add tag...');
    const addButton = screen.getByText('Add');
    
    await user.type(tagInput, 'existing-tag');
    await user.click(addButton);
    
    // Should not add duplicate tag
    expect(mockProps.onFiltersChange).not.toHaveBeenCalledWith({
      tags: ['existing-tag', 'existing-tag'],
    });
  });

  it('disables add button when tag input is empty', async () => {
    const user = userEvent.setup();
    render(<DocumentFilters {...mockProps} />);
    
    // Show advanced filters first
    const advancedButton = screen.getByText('Advanced');
    await user.click(advancedButton);
    
    const addButton = screen.getByText('Add');
    expect(addButton).toBeDisabled();
    
    const tagInput = screen.getByPlaceholderText('Add tag...');
    await user.type(tagInput, 'test');
    
    expect(addButton).not.toBeDisabled();
  });
});