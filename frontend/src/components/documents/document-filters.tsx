'use client';

import React, { useState } from 'react';
import { Search, Filter, X, Calendar, Tag, FileType, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { DocumentFilters as DocumentFiltersType } from '@/types';
import { cn } from '@/lib/utils';

interface DocumentFiltersProps {
  filters: DocumentFiltersType;
  onFiltersChange: (filters: DocumentFiltersType) => void;
  onClearFilters: () => void;
  className?: string;
}

const DOCUMENT_CATEGORIES = [
  'Policy Documents',
  'Reports',
  'Presentations',
  'Spreadsheets',
  'Legal Documents',
  'Technical Documentation',
  'Meeting Minutes',
  'Correspondence',
  'Research Papers',
  'Training Materials',
];

const DOCUMENT_TYPES = [
  'pdf',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'txt',
  'csv',
  'json',
  'xml',
];

const CLASSIFICATION_LEVELS = [
  { value: 'public', label: 'Public' },
  { value: 'internal', label: 'Internal' },
  { value: 'confidential', label: 'Confidential' },
  { value: 'secret', label: 'Secret' },
];

const STATUS_OPTIONS = [
  { value: 'uploading', label: 'Uploading' },
  { value: 'processing', label: 'Processing' },
  { value: 'completed', label: 'Completed' },
  { value: 'error', label: 'Error' },
];

export function DocumentFilters({
  filters,
  onFiltersChange,
  onClearFilters,
  className,
}: DocumentFiltersProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [tagInput, setTagInput] = useState('');

  const updateFilter = (key: keyof DocumentFiltersType, value: unknown) => {
    // Handle "all" values by setting to undefined
    const actualValue = value === 'all' ? undefined : value;
    onFiltersChange({
      ...filters,
      [key]: actualValue,
    });
  };

  const addTag = () => {
    if (!tagInput.trim()) return;
    
    const currentTags = filters.tags || [];
    if (!currentTags.includes(tagInput.trim())) {
      updateFilter('tags', [...currentTags, tagInput.trim()]);
    }
    setTagInput('');
  };

  const removeTag = (tagToRemove: string) => {
    const currentTags = filters.tags || [];
    updateFilter('tags', currentTags.filter(tag => tag !== tagToRemove));
  };

  const setDateRange = (field: 'start' | 'end', value: string) => {
    const currentRange = filters.dateRange || { start: new Date(), end: new Date() };
    updateFilter('dateRange', {
      ...currentRange,
      [field]: new Date(value),
    });
  };

  const clearDateRange = () => {
    updateFilter('dateRange', undefined);
  };

  const hasActiveFilters = () => {
    return !!(
      filters.searchQuery ||
      filters.category ||
      filters.classification ||
      filters.status ||
      (filters.tags && filters.tags.length > 0) ||
      filters.dateRange
    );
  };

  const getActiveFilterCount = () => {
    let count = 0;
    if (filters.searchQuery) count++;
    if (filters.category) count++;
    if (filters.classification) count++;
    if (filters.status) count++;
    if (filters.tags && filters.tags.length > 0) count++;
    if (filters.dateRange) count++;
    return count;
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg flex items-center gap-2">
            <Filter className="h-5 w-5" />
            Filters
            {hasActiveFilters() && (
              <Badge variant="secondary" className="ml-2">
                {getActiveFilterCount()}
              </Badge>
            )}
          </CardTitle>
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowAdvanced(!showAdvanced)}
            >
              {showAdvanced ? 'Simple' : 'Advanced'}
            </Button>
            {hasActiveFilters() && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onClearFilters}
                className="text-muted-foreground"
              >
                Clear All
              </Button>
            )}
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Search */}
        <div className="space-y-2">
          <Label htmlFor="search" className="flex items-center gap-2">
            <Search className="h-4 w-4" />
            Search
          </Label>
          <Input
            id="search"
            placeholder="Search documents..."
            value={filters.searchQuery || ''}
            onChange={(e) => updateFilter('searchQuery', e.target.value)}
          />
        </div>

        {/* Basic Filters */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <FileType className="h-4 w-4" />
              Category
            </Label>
            <Select
              value={filters.category || ''}
              onValueChange={(value) => updateFilter('category', value || undefined)}
            >
              <SelectTrigger>
                <SelectValue placeholder="All categories" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All categories</SelectItem>
                {DOCUMENT_CATEGORIES.map((category) => (
                  <SelectItem key={category} value={category}>
                    {category}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label className="flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Classification
            </Label>
            <Select
              value={filters.classification || ''}
              onValueChange={(value) => updateFilter('classification', value || undefined)}
            >
              <SelectTrigger>
                <SelectValue placeholder="All classifications" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All classifications</SelectItem>
                {CLASSIFICATION_LEVELS.map((level) => (
                  <SelectItem key={level.value} value={level.value}>
                    {level.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Advanced Filters */}
        {showAdvanced && (
          <div className="space-y-4 pt-4 border-t">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  value={filters.status || ''}
                  onValueChange={(value) => updateFilter('status', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All statuses" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All statuses</SelectItem>
                    {STATUS_OPTIONS.map((status) => (
                      <SelectItem key={status.value} value={status.value}>
                        {status.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className="flex items-center gap-2">
                  <Calendar className="h-4 w-4" />
                  Date Range
                </Label>
                <div className="flex items-center gap-2">
                  <Input
                    type="date"
                    value={filters.dateRange?.start ? filters.dateRange.start.toISOString().split('T')[0] : ''}
                    onChange={(e) => setDateRange('start', e.target.value)}
                    className="flex-1"
                  />
                  <span className="text-muted-foreground">to</span>
                  <Input
                    type="date"
                    value={filters.dateRange?.end ? filters.dateRange.end.toISOString().split('T')[0] : ''}
                    onChange={(e) => setDateRange('end', e.target.value)}
                    className="flex-1"
                  />
                  {filters.dateRange && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={clearDateRange}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              </div>
            </div>

            {/* Tags */}
            <div className="space-y-2">
              <Label className="flex items-center gap-2">
                <Tag className="h-4 w-4" />
                Tags
              </Label>
              
              {/* Current tags */}
              {filters.tags && filters.tags.length > 0 && (
                <div className="flex flex-wrap gap-2 mb-2">
                  {filters.tags.map((tag, index) => (
                    <Badge key={index} variant="secondary" className="flex items-center gap-1">
                      {tag}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="h-4 w-4 p-0 hover:bg-transparent"
                        onClick={() => removeTag(tag)}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </Badge>
                  ))}
                </div>
              )}

              {/* Add tag input */}
              <div className="flex gap-2">
                <Input
                  placeholder="Add tag..."
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  onKeyPress={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault();
                      addTag();
                    }
                  }}
                />
                <Button
                  type="button"
                  onClick={addTag}
                  disabled={!tagInput.trim()}
                >
                  Add
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Active Filters Summary */}
        {hasActiveFilters() && (
          <div className="pt-4 border-t">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">
                {getActiveFilterCount()} filter{getActiveFilterCount() !== 1 ? 's' : ''} applied
              </span>
              <Button
                variant="outline"
                size="sm"
                onClick={onClearFilters}
              >
                Clear All
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}