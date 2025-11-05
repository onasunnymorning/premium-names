import axios from 'axios';
import { 
  DomainLabel, 
  LabelsResponse, 
  TagsResponse, 
  TagStatsResponse, 
  LabelFilters,
  StartWorkflowRequest,
  StartWorkflowResponse,
  WorkflowStatus
} from '../types/api';

// Create axios instance with base configuration
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for debugging
api.interceptors.request.use(
  (config) => {
    console.log(`API Request: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error.response?.data || error.message);
    return Promise.reject(error);
  }
);

export const labelsApi = {
  // Get paginated labels with filters
  getLabels: async (filters: LabelFilters = {}): Promise<LabelsResponse> => {
    const response = await api.get('/labels', { params: filters });
    return response.data;
  },

  // Get single label by ID
  getLabel: async (id: number): Promise<DomainLabel> => {
    const response = await api.get(`/labels/${id}`);
    return response.data;
  },

  // Update label tags
  updateLabelTags: async (id: number, tags: string[]): Promise<DomainLabel> => {
    const response = await api.put(`/labels/${id}/tags`, { tags });
    return response.data;
  },

  // Delete label
  deleteLabel: async (id: number): Promise<void> => {
    await api.delete(`/labels/${id}`);
  },

  // Get all tags
  getTags: async (): Promise<TagsResponse> => {
    const response = await api.get('/tags');
    return response.data;
  },

  // Get tag statistics
  getTagStats: async (): Promise<TagStatsResponse> => {
    const response = await api.get('/tags/stats');
    return response.data;
  },

  // Upload file (traditional upload)
  uploadFile: async (formData: FormData): Promise<any> => {
    const response = await api.post('/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  },
};

export const workflowApi = {
  // Start domain labels workflow
  startWorkflow: async (request: StartWorkflowRequest): Promise<StartWorkflowResponse> => {
    const response = await api.post('/workflows/domain-labels', request);
    return response.data;
  },

  // Get workflow status
  getWorkflowStatus: async (workflowId: string): Promise<WorkflowStatus> => {
    const response = await api.get(`/workflows/${workflowId}/status`);
    return response.data;
  },
};

export default api;
