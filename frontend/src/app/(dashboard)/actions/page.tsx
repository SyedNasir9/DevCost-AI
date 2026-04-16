'use client';

import { useEffect, useState } from 'react';
import { mockApi, Action } from '@/lib/mock-service';
import { Card, Badge, Button, SearchInput, EmptyState, StatusIndicator } from '@/components/ui';

export default function ActionsPage() {
  const [actions, setActions] = useState<Action[]>([]);
  const [filteredActions, setFilteredActions] = useState<Action[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [refreshing, setRefreshing] = useState(false);

  const loadActionsData = async () => {
    try {
      setError(null);
      const data = await mockApi.getActions();
      setActions(data.actions);
      setFilteredActions(data.actions);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load actions');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    mockApi.clearCache();
    await loadActionsData();
  };

  useEffect(() => {
    loadActionsData();
  }, []);

  useEffect(() => {
    let filtered = actions;

    // Filter by search term
    if (searchTerm) {
      filtered = filtered.filter(item => 
        item.action_type.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.resource_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.resource_type.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter(item => item.status === statusFilter);
    }

    setFilteredActions(filtered);
  }, [actions, searchTerm, statusFilter]);

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed': return 'success';
      case 'failed': return 'error';
      case 'running': return 'info';
      case 'pending': return 'warning';
      default: return 'default';
    }
  };

  const getStatusVariant = (status: string): 'online' | 'offline' | 'warning' | 'error' => {
    switch (status.toLowerCase()) {
      case 'completed': return 'online';
      case 'failed': return 'error';
      case 'running': return 'warning';
      case 'pending': return 'offline';
      default: return 'offline';
    }
  };

  const getActionIcon = (actionType: string) => {
    switch (actionType.toLowerCase()) {
      case 'stop': return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
        </svg>
      );
      case 'start': return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
      case 'restart': return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
      );
      case 'delete': return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      );
      case 'resize': return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
        </svg>
      );
      default: return (
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      );
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  const formatDuration = (durationMs?: number) => {
    if (!durationMs) return '-';
    if (durationMs < 1000) return `${durationMs}ms`;
    if (durationMs < 60000) return `${(durationMs / 1000).toFixed(1)}s`;
    return `${(durationMs / 60000).toFixed(1)}m`;
  };

  const completedActions = filteredActions.filter(item => item.status === 'completed').length;
  const failedActions = filteredActions.filter(item => item.status === 'failed').length;
  const runningActions = filteredActions.filter(item => item.status === 'running').length;

  if (loading) {
    return (
      <div className="space-y-6">
        {/* Header Skeleton */}
        <div className="flex items-center justify-between">
          <div>
            <div className="h-8 bg-neutral-200 rounded w-56 mb-2"></div>
            <div className="h-4 bg-neutral-200 rounded w-72"></div>
          </div>
          <div className="h-10 bg-neutral-200 rounded w-24"></div>
        </div>

        {/* Stats Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-neutral-200 p-4">
              <div className="animate-pulse">
                <div className="h-4 bg-neutral-200 rounded w-16 mb-2"></div>
                <div className="h-6 bg-neutral-200 rounded w-12"></div>
              </div>
            </div>
          ))}
        </div>

        {/* Actions Skeleton */}
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-neutral-200 p-6">
              <div className="animate-pulse space-y-4">
                <div className="h-6 bg-neutral-200 rounded w-1/3"></div>
                <div className="h-4 bg-neutral-200 rounded w-full"></div>
                <div className="h-4 bg-neutral-200 rounded w-2/3"></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-red-800">Error Loading Actions</h3>
            <p className="text-red-600 text-sm mt-1">{error}</p>
          </div>
          <Button variant="secondary" onClick={handleRefresh}>
            Retry
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-neutral-900">Execution History</h1>
          <p className="text-neutral-600 mt-1">Track the status of automated optimization actions</p>
        </div>
        <div className="flex items-center space-x-3">
          <div className="text-right">
            <div className="text-sm text-neutral-500">Total Actions</div>
            <div className="text-lg font-semibold text-neutral-900">{filteredActions.length}</div>
          </div>
          <Button 
            variant="ghost" 
            onClick={handleRefresh} 
            loading={refreshing}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Total</p>
              <p className="text-2xl font-bold text-neutral-900">{filteredActions.length}</p>
            </div>
            <div className="p-2 bg-blue-50 rounded-lg">
              <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
              </svg>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Completed</p>
              <p className="text-2xl font-bold text-green-600">{completedActions}</p>
            </div>
            <StatusIndicator status="online" label="Success" />
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Failed</p>
              <p className="text-2xl font-bold text-red-600">{failedActions}</p>
            </div>
            <StatusIndicator status="error" label="Failed" />
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Running</p>
              <p className="text-2xl font-bold text-blue-600">{runningActions}</p>
            </div>
            <StatusIndicator status="warning" label="Active" />
          </div>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center space-x-4">
        <div className="flex-1 max-w-md">
          <SearchInput
            placeholder="Search actions..."
            value={searchTerm}
            onSearch={setSearchTerm}
          />
        </div>
        
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border border-neutral-300 rounded-lg bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="all">All Statuses</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
          <option value="running">Running</option>
          <option value="pending">Pending</option>
        </select>
      </div>

      {/* Actions Timeline */}
      {filteredActions.length === 0 ? (
        <Card>
          <div className="p-12">
            <EmptyState
              icon={
                <svg className="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              }
              title="No Actions Found"
              description={searchTerm || statusFilter !== 'all' ? 'No actions match your filters' : 'No actions have been executed yet'}
              action={
                (searchTerm || statusFilter !== 'all') && (
                  <Button variant="secondary" onClick={() => {
                    setSearchTerm('');
                    setStatusFilter('all');
                  }}>
                    Clear Filters
                  </Button>
                )
              }
            />
          </div>
        </Card>
      ) : (
        <div className="space-y-4">
          {filteredActions.map((action) => (
            <Card key={action.id} className="p-6">
              <div className="flex items-start justify-between">
                <div className="flex items-start space-x-4">
                  <div className="p-2 bg-neutral-50 rounded-lg">
                    {getActionIcon(action.action_type)}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center space-x-3 mb-2">
                      <h3 className="text-lg font-semibold text-neutral-900 capitalize">{action.action_type}</h3>
                      <Badge variant={getStatusColor(action.status) as any}>
                        {action.status}
                      </Badge>
                      <StatusIndicator 
                        status={getStatusVariant(action.status)} 
                        label={action.status} 
                      />
                    </div>
                    
                    <div className="flex items-center space-x-6 text-sm text-neutral-600 mb-3">
                      <div className="flex items-center space-x-2">
                        <span>Resource:</span>
                        <span className="font-medium text-neutral-900">{action.resource_id}</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span>Type:</span>
                        <Badge variant="info">{action.resource_type}</Badge>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-6 text-xs text-neutral-500">
                      <div className="flex items-center space-x-2">
                        <span>Created:</span>
                        <span>{formatDate(action.created_at)}</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span>Executed:</span>
                        <span>{formatDate(action.executed_at)}</span>
                      </div>
                      <div className="flex items-center space-x-2">
                        <span>Duration:</span>
                        <span>{formatDuration(action.duration_ms)}</span>
                      </div>
                    </div>
                    
                    {action.error_message && (
                      <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded-lg">
                        <div className="flex items-center space-x-2">
                          <svg className="w-4 h-4 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          <span className="text-sm text-red-700">{action.error_message}</span>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
