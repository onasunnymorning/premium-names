// API types that match the backend models
export interface DomainLabel {
  id: number;
  label: string;
  original?: string;
  tags: Tag[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: number;
  name: string;
  description?: string;
  color?: string;
  created_at: string;
  updated_at: string;
}

export interface TagStat {
  tag_id: number;
  tag_name: string;
  count: number;
  color: string;
}

export interface PaginationInfo {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export interface LabelsResponse {
  labels: DomainLabel[];
  pagination: PaginationInfo;
}

export interface TagsResponse {
  tags: Tag[];
}

export interface TagStatsResponse {
  tag_stats: TagStat[];
}

// Workflow types
export interface StartWorkflowRequest {
  file_uri: string;
  tags: string[];
  created_by: string;
  description?: string;
}

export interface StartWorkflowResponse {
  workflow_id: string;
  run_id: string;
}

export interface ProcessedDomainLabel {
  id: number;
  label: string;
  original: string;
  tags: string[];
  created: boolean;
}

export interface WorkflowResult {
  processed_count: number;
  saved_count: number;
  skipped_count: number;
  error_count: number;
  labels: ProcessedDomainLabel[];
  errors: string[];
}

export interface WorkflowStatus {
  workflow_id: string;
  status: string;
  start_time?: string;
  result?: WorkflowResult;
}

// Filter and search options
export interface LabelFilters {
  page?: number;
  limit?: number;
  q?: string; // search query
  tag?: string; // filter by tag
  created_by?: string; // filter by creator
  order_by?: string; // field to order by
  order_dir?: 'asc' | 'desc'; // order direction
}
