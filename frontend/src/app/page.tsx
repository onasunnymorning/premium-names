'use client';

import { useState } from 'react';
import { LabelsDashboard } from '../components/LabelsDashboard';
import { FileUpload } from '../components/FileUpload';

export default function Home() {
  const [currentTab, setCurrentTab] = useState('dashboard');

  const renderContent = () => {
    switch (currentTab) {
      case 'upload':
        return <FileUpload />;
      case 'tags':
        return (
          <div className="p-6">
            <h1 className="text-2xl font-semibold text-gray-900 mb-4">Tag Management</h1>
            <p className="text-gray-600">Tag management features coming soon...</p>
          </div>
        );
      default:
        return <LabelsDashboard />;
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navigation */}
      <nav className="bg-white shadow">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 justify-between">
            <div className="flex">
              <div className="flex flex-shrink-0 items-center">
                <h1 className="text-xl font-semibold text-gray-900">
                  Domain Labels Manager
                </h1>
              </div>
              <div className="hidden md:ml-6 md:flex md:space-x-8">
                <button
                  onClick={() => setCurrentTab('dashboard')}
                  className={`inline-flex items-center border-b-2 px-1 pt-1 text-sm font-medium ${
                    currentTab === 'dashboard'
                      ? 'border-blue-500 text-gray-900'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  Dashboard
                </button>
                <button
                  onClick={() => setCurrentTab('upload')}
                  className={`inline-flex items-center border-b-2 px-1 pt-1 text-sm font-medium ${
                    currentTab === 'upload'
                      ? 'border-blue-500 text-gray-900'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  Upload
                </button>
                <button
                  onClick={() => setCurrentTab('tags')}
                  className={`inline-flex items-center border-b-2 px-1 pt-1 text-sm font-medium ${
                    currentTab === 'tags'
                      ? 'border-blue-500 text-gray-900'
                      : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
                  }`}
                >
                  Tags
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Mobile menu - simplified for now */}
        <div className="md:hidden px-4 py-2 space-y-1">
          <button
            onClick={() => setCurrentTab('dashboard')}
            className={`block w-full text-left px-3 py-2 rounded-md text-base font-medium ${
              currentTab === 'dashboard'
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-900 hover:bg-gray-50'
            }`}
          >
            Dashboard
          </button>
          <button
            onClick={() => setCurrentTab('upload')}
            className={`block w-full text-left px-3 py-2 rounded-md text-base font-medium ${
              currentTab === 'upload'
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-900 hover:bg-gray-50'
            }`}
          >
            Upload
          </button>
          <button
            onClick={() => setCurrentTab('tags')}
            className={`block w-full text-left px-3 py-2 rounded-md text-base font-medium ${
              currentTab === 'tags'
                ? 'bg-blue-50 text-blue-700'
                : 'text-gray-900 hover:bg-gray-50'
            }`}
          >
            Tags
          </button>
        </div>
      </nav>

      {/* Main content */}
      <main className="mx-auto max-w-7xl">
        {renderContent()}
      </main>
    </div>
  );
}
