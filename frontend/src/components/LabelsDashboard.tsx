'use client';

import { useState } from 'react';
import { MagnifyingGlassIcon, FunnelIcon } from '@heroicons/react/24/outline';
import { useLabels, useTagStats, useDeleteLabel, useUpdateLabelTags } from '../hooks/useApi';
import { LabelFilters } from '../types/api';
import { Input } from './ui/Input';
import { Button } from './ui/Button';
import { Badge } from './ui/Badge';
import { TagList } from './TagList';

export function LabelsDashboard() {
  const [filters, setFilters] = useState<LabelFilters>({
    page: 1,
    limit: 25,
    order_by: 'created_at',
    order_dir: 'desc',
  });

  const { data: labelsData, isLoading, error } = useLabels(filters);
  const { data: tagStatsData } = useTagStats();
  const deleteLabel = useDeleteLabel();
  const updateLabelTags = useUpdateLabelTags();

  const handleSearch = (query: string) => {
    setFilters(prev => ({ ...prev, q: query, page: 1 }));
  };

  const handleTagFilter = (tagName: string) => {
    setFilters(prev => ({ 
      ...prev, 
      tag: prev.tag === tagName ? undefined : tagName,
      page: 1 
    }));
  };

  const handlePageChange = (page: number) => {
    setFilters(prev => ({ ...prev, page }));
  };

  const handleDeleteLabel = async (id: number) => {
    if (window.confirm('Are you sure you want to delete this label?')) {
      try {
        await deleteLabel.mutateAsync(id);
      } catch (error) {
        console.error('Failed to delete label:', error);
        alert('Failed to delete label');
      }
    }
  };

  if (error) {
    return (
      <div className="p-6">
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <h3 className="text-red-800 font-medium">Error loading labels</h3>
          <p className="text-red-600 text-sm mt-1">
            {error instanceof Error ? error.message : 'An unexpected error occurred'}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Domain Labels</h1>
        <p className="text-gray-600">Manage and search your domain labels with tags</p>
      </div>

      {/* Search and filters */}
      <div className="mb-6 space-y-4">
        <div className="flex gap-4">
          <div className="flex-1">
            <Input
              placeholder="Search labels..."
              value={filters.q || ''}
              onChange={(e) => handleSearch(e.target.value)}
            />
          </div>
          <Button variant="outline" size="md">
            <FunnelIcon className="h-4 w-4 mr-2" />
            Filters
          </Button>
        </div>

        {/* Tag filter chips */}
        {tagStatsData?.tag_stats && (
          <div className="flex flex-wrap gap-2">
            <span className="text-sm font-medium text-gray-700">Filter by tag:</span>
            {tagStatsData.tag_stats.map((stat) => (
              <Badge
                key={stat.tag_id}
                variant={filters.tag === stat.tag_name ? 'default' : 'outline'}
                className="cursor-pointer"
                onClick={() => handleTagFilter(stat.tag_name)}
              >
                {stat.tag_name} ({stat.count})
              </Badge>
            ))}
          </div>
        )}

        {/* Active filters */}
        {(filters.tag || filters.q) && (
          <div className="flex gap-2 items-center">
            <span className="text-sm text-gray-600">Active filters:</span>
            {filters.q && (
              <Badge variant="secondary">
                Search: {filters.q}
                <button
                  onClick={() => setFilters(prev => ({ ...prev, q: undefined }))}
                  className="ml-1 text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </Badge>
            )}
            {filters.tag && (
              <Badge variant="secondary">
                Tag: {filters.tag}
                <button
                  onClick={() => setFilters(prev => ({ ...prev, tag: undefined }))}
                  className="ml-1 text-gray-400 hover:text-gray-600"
                >
                  ×
                </button>
              </Badge>
            )}
          </div>
        )}
      </div>

      {/* Results */}
      {isLoading ? (
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="text-gray-600 mt-4">Loading labels...</p>
        </div>
      ) : (
        <>
          <div className="bg-white shadow overflow-hidden sm:rounded-md">
            <ul className="divide-y divide-gray-200">
              {labelsData?.labels.map((label) => (
                <li key={label.id} className="px-6 py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center">
                        <h3 className="text-lg font-medium text-gray-900">
                          {label.label}
                        </h3>
                        {label.original && label.original !== label.label && (
                          <span className="ml-2 text-sm text-gray-500">
                            (from: {label.original})
                          </span>
                        )}
                      </div>
                      
                      <div className="mt-2 flex items-center gap-4">
                        <TagList 
                          tags={label.tags}
                          onTagClick={handleTagFilter}
                          selectedTag={filters.tag}
                        />
                      </div>
                      
                      <div className="mt-2 text-sm text-gray-500">
                        Created by {label.created_by} on{' '}
                        {new Date(label.created_at).toLocaleDateString()}
                      </div>
                    </div>
                    
                    <div className="flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          // TODO: Open edit dialog
                          console.log('Edit label:', label.id);
                        }}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDeleteLabel(label.id)}
                        disabled={deleteLabel.isPending}
                      >
                        Delete
                      </Button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </div>

          {/* Pagination */}
          {labelsData?.pagination && labelsData.pagination.total_pages > 1 && (
            <div className="mt-6 flex items-center justify-between">
              <div className="text-sm text-gray-700">
                Showing {((labelsData.pagination.page - 1) * labelsData.pagination.limit) + 1} to{' '}
                {Math.min(labelsData.pagination.page * labelsData.pagination.limit, labelsData.pagination.total)} of{' '}
                {labelsData.pagination.total} results
              </div>
              
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handlePageChange(labelsData.pagination.page - 1)}
                  disabled={labelsData.pagination.page <= 1}
                >
                  Previous
                </Button>
                
                {/* Page numbers */}
                {Array.from({ length: Math.min(5, labelsData.pagination.total_pages) }, (_, i) => {
                  const page = i + 1;
                  return (
                    <Button
                      key={page}
                      variant={page === labelsData.pagination.page ? 'primary' : 'outline'}
                      size="sm"
                      onClick={() => handlePageChange(page)}
                    >
                      {page}
                    </Button>
                  );
                })}
                
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handlePageChange(labelsData.pagination.page + 1)}
                  disabled={labelsData.pagination.page >= labelsData.pagination.total_pages}
                >
                  Next
                </Button>
              </div>
            </div>
          )}

          {/* Empty state */}
          {labelsData?.labels.length === 0 && (
            <div className="text-center py-12">
              <MagnifyingGlassIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-semibold text-gray-900">No labels found</h3>
              <p className="mt-1 text-sm text-gray-500">
                {filters.q || filters.tag ? 'Try adjusting your search or filters.' : 'Get started by uploading a file.'}
              </p>
            </div>
          )}
        </>
      )}
    </div>
  );
}
