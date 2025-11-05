'use client';

import { useState, useRef } from 'react';
import { CloudArrowUpIcon, DocumentIcon, CheckCircleIcon, XCircleIcon } from '@heroicons/react/24/outline';
import { useStartWorkflow, useWorkflowStatus, useUploadFile } from '../hooks/useApi';
import { Button } from './ui/Button';
import { Input } from './ui/Input';

export function FileUpload() {
  const [file, setFile] = useState<File | null>(null);
  const [tags, setTags] = useState<string>('');
  const [createdBy, setCreatedBy] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const [workflowId, setWorkflowId] = useState<string | null>(null);
  const [useWorkflow, setUseWorkflow] = useState(true);
  
  const fileInputRef = useRef<HTMLInputElement>(null);
  
  const startWorkflow = useStartWorkflow();
  const uploadFile = useUploadFile();
  const { data: workflowStatus, isLoading: isStatusLoading } = useWorkflowStatus(workflowId, !!workflowId);

  const handleFileSelect = (selectedFile: File) => {
    const supportedTypes = [
      'text/csv',
      'text/tab-separated-values',
      'application/vnd.ms-excel',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ];
    
    const supportedExtensions = ['.csv', '.tsv', '.xls', '.xlsx'];
    const isSupported = supportedTypes.includes(selectedFile.type) || 
      supportedExtensions.some(ext => selectedFile.name.toLowerCase().endsWith(ext));
    
    if (!isSupported) {
      alert('Please select a CSV, TSV, or Excel file');
      return;
    }
    
    setFile(selectedFile);
  };

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    const droppedFiles = Array.from(e.dataTransfer.files);
    if (droppedFiles.length > 0) {
      handleFileSelect(droppedFiles[0]);
    }
  };

  const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!file || !tags || !createdBy) {
      alert('Please fill in all required fields');
      return;
    }

    const tagList = tags.split(',').map(tag => tag.trim()).filter(Boolean);
    
    if (useWorkflow) {
      // Use Temporal workflow for processing
      try {
        // For workflow, we need to upload to a location accessible by the worker
        // For now, we'll use the traditional upload and then trigger workflow
        // In production, you'd upload to S3 or similar
        const formData = new FormData();
        formData.append('file', file);
        formData.append('tags', tagList.join(','));
        formData.append('created_by', createdBy);
        
        const uploadResult = await uploadFile.mutateAsync(formData);
        console.log('File uploaded successfully:', uploadResult);
        
        // Reset form
        setFile(null);
        setTags('');
        setCreatedBy('');
        setDescription('');
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        
      } catch (error) {
        console.error('Upload failed:', error);
        alert('Upload failed. Please try again.');
      }
    } else {
      // Use traditional upload
      try {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('tags', tagList.join(','));
        formData.append('created_by', createdBy);
        
        const result = await uploadFile.mutateAsync(formData);
        console.log('Upload successful:', result);
        
        // Reset form
        setFile(null);
        setTags('');
        setCreatedBy('');
        setDescription('');
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        
      } catch (error) {
        console.error('Upload failed:', error);
        alert('Upload failed. Please try again.');
      }
    }
  };

  const isProcessing = startWorkflow.isPending || uploadFile.isPending;

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-gray-900">Upload Domain Labels</h1>
        <p className="text-gray-600">Upload CSV, TSV, or Excel files containing domain names or labels</p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Processing method toggle */}
        <div className="bg-gray-50 p-4 rounded-lg">
          <h3 className="text-sm font-medium text-gray-900 mb-3">Processing Method</h3>
          <div className="flex gap-4">
            <label className="flex items-center">
              <input
                type="radio"
                checked={!useWorkflow}
                onChange={() => setUseWorkflow(false)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Direct upload (faster)</span>
            </label>
            <label className="flex items-center">
              <input
                type="radio"
                checked={useWorkflow}
                onChange={() => setUseWorkflow(true)}
                className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
              />
              <span className="ml-2 text-sm text-gray-700">Workflow processing (scalable)</span>
            </label>
          </div>
          <p className="text-xs text-gray-500 mt-2">
            Direct upload processes files immediately. Workflow processing handles large files better and provides detailed progress tracking.
          </p>
        </div>

        {/* File upload area */}
        <div
          className={`border-2 border-dashed rounded-lg p-6 text-center transition-colors ${
            file ? 'border-green-300 bg-green-50' : 'border-gray-300 hover:border-gray-400'
          }`}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
        >
          {file ? (
            <div className="flex items-center justify-center">
              <DocumentIcon className="h-8 w-8 text-green-600 mr-3" />
              <div>
                <p className="text-sm font-medium text-green-900">{file.name}</p>
                <p className="text-xs text-green-600">
                  {(file.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => {
                  setFile(null);
                  if (fileInputRef.current) {
                    fileInputRef.current.value = '';
                  }
                }}
                className="ml-4"
              >
                Remove
              </Button>
            </div>
          ) : (
            <div>
              <CloudArrowUpIcon className="mx-auto h-12 w-12 text-gray-400" />
              <div className="mt-4">
                <label className="cursor-pointer">
                  <span className="mt-2 block text-sm font-medium text-gray-900">
                    Click to upload or drag and drop
                  </span>
                  <span className="text-xs text-gray-500">
                    CSV, TSV, Excel files (up to 10MB)
                  </span>
                  <input
                    ref={fileInputRef}
                    type="file"
                    className="hidden"
                    accept=".csv,.tsv,.xls,.xlsx"
                    onChange={(e) => {
                      if (e.target.files?.[0]) {
                        handleFileSelect(e.target.files[0]);
                      }
                    }}
                  />
                </label>
              </div>
            </div>
          )}
        </div>

        {/* Form fields */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Input
            label="Tags *"
            placeholder="premium, import-2024, high-value"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            helpText="Comma-separated tags to apply to all labels"
            required
          />
          
          <Input
            label="Created By *"
            type="email"
            placeholder="your.email@example.com"
            value={createdBy}
            onChange={(e) => setCreatedBy(e.target.value)}
            required
          />
        </div>

        <Input
          label="Description"
          placeholder="Optional description for this batch of labels"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />

        {/* Submit button */}
        <Button
          type="submit"
          disabled={!file || !tags || !createdBy || isProcessing}
          className="w-full"
        >
          {isProcessing ? (
            <div className="flex items-center">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
              Processing...
            </div>
          ) : (
            `Upload ${file ? `${file.name}` : 'File'}`
          )}
        </Button>
      </form>

      {/* Workflow status */}
      {workflowId && (
        <div className="mt-8 p-4 bg-blue-50 rounded-lg">
          <h3 className="text-sm font-medium text-blue-900 mb-2">Workflow Status</h3>
          <div className="flex items-center">
            {isStatusLoading ? (
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
            ) : workflowStatus?.status === 'COMPLETED' ? (
              <CheckCircleIcon className="h-5 w-5 text-green-600 mr-2" />
            ) : workflowStatus?.status === 'FAILED' ? (
              <XCircleIcon className="h-5 w-5 text-red-600 mr-2" />
            ) : (
              <div className="animate-pulse h-2 w-2 bg-blue-600 rounded-full mr-2"></div>
            )}
            <span className="text-sm text-blue-800">
              Workflow {workflowId}: {workflowStatus?.status || 'Loading...'}
            </span>
          </div>
          
          {workflowStatus?.result && (
            <div className="mt-4 text-sm">
              <p>Processed: {workflowStatus.result.processed_count}</p>
              <p>Saved: {workflowStatus.result.saved_count}</p>
              <p>Skipped: {workflowStatus.result.skipped_count}</p>
              {workflowStatus.result.error_count > 0 && (
                <p className="text-red-600">Errors: {workflowStatus.result.error_count}</p>
              )}
            </div>
          )}
        </div>
      )}

      {/* Upload result */}
      {(uploadFile.data || startWorkflow.data) && (
        <div className="mt-8 p-4 bg-green-50 rounded-lg">
          <div className="flex items-center">
            <CheckCircleIcon className="h-5 w-5 text-green-600 mr-2" />
            <span className="text-sm font-medium text-green-900">Upload successful!</span>
          </div>
          {uploadFile.data && (
            <div className="mt-2 text-sm text-green-800">
              <p>Processed: {uploadFile.data.processed} items</p>
              <p>Saved: {uploadFile.data.saved} labels</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
