import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { labelsApi, workflowApi } from '../lib/api';
import { LabelFilters } from '../types/api';

// Query keys for cache management
export const queryKeys = {
  labels: ['labels'] as const,
  labelsList: (filters: LabelFilters) => [...queryKeys.labels, 'list', filters] as const,
  labelsDetail: (id: number) => [...queryKeys.labels, 'detail', id] as const,
  tags: ['tags'] as const,
  tagsList: () => [...queryKeys.tags, 'list'] as const,
  tagsStats: () => [...queryKeys.tags, 'stats'] as const,
  workflow: ['workflow'] as const,
  workflowStatus: (id: string) => [...queryKeys.workflow, 'status', id] as const,
};

// Labels hooks
export function useLabels(filters: LabelFilters = {}) {
  return useQuery({
    queryKey: queryKeys.labelsList(filters),
    queryFn: () => labelsApi.getLabels(filters),
    placeholderData: (previousData) => previousData,
  });
}

export function useLabel(id: number, enabled = true) {
  return useQuery({
    queryKey: queryKeys.labelsDetail(id),
    queryFn: () => labelsApi.getLabel(id),
    enabled: enabled && !!id,
  });
}

export function useUpdateLabelTags() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, tags }: { id: number; tags: string[] }) =>
      labelsApi.updateLabelTags(id, tags),
    onSuccess: (data) => {
      // Update the single label cache
      queryClient.setQueryData(queryKeys.labelsDetail(data.id), data);
      
      // Invalidate all labels list queries to refresh pagination
      queryClient.invalidateQueries({
        queryKey: queryKeys.labels,
        exact: false,
      });
    },
  });
}

export function useDeleteLabel() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: number) => labelsApi.deleteLabel(id),
    onSuccess: () => {
      // Invalidate all labels queries to refresh the list
      queryClient.invalidateQueries({
        queryKey: queryKeys.labels,
        exact: false,
      });
    },
  });
}

// Tags hooks
export function useTags() {
  return useQuery({
    queryKey: queryKeys.tagsList(),
    queryFn: () => labelsApi.getTags(),
    staleTime: 10 * 60 * 1000, // Tags don't change often, cache for 10 minutes
  });
}

export function useTagStats() {
  return useQuery({
    queryKey: queryKeys.tagsStats(),
    queryFn: () => labelsApi.getTagStats(),
  });
}

// Upload hook (traditional upload)
export function useUploadFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (formData: FormData) => labelsApi.uploadFile(formData),
    onSuccess: () => {
      // Refresh labels and tag stats after successful upload
      queryClient.invalidateQueries({
        queryKey: queryKeys.labels,
        exact: false,
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.tagsStats(),
      });
    },
  });
}

// Workflow hooks
export function useStartWorkflow() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: workflowApi.startWorkflow,
    onSuccess: () => {
      // Don't invalidate immediately for workflows as they're async
      // The user will need to poll for status
    },
  });
}

export function useWorkflowStatus(workflowId: string | null, enabled = true) {
  return useQuery({
    queryKey: queryKeys.workflowStatus(workflowId || ''),
    queryFn: () => workflowApi.getWorkflowStatus(workflowId!),
    enabled: enabled && !!workflowId,
    refetchInterval: (query) => {
      // Poll every 5 seconds if workflow is still running
      const data = query.state.data;
      if (data?.status === 'RUNNING' || data?.status === 'PENDING') {
        return 5000;
      }
      return false;
    },
  });
}
