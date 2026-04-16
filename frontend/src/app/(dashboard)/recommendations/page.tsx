'use client';

import { useEffect, useState } from 'react';
import { mockApi, Recommendation } from '@/lib/mock-service';
import { Card, Badge, Button, SearchInput, EmptyState } from '@/components/ui';

export default function RecommendationsPage() {
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [filteredRecommendations, setFilteredRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [priorityFilter, setPriorityFilter] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [refreshing, setRefreshing] = useState(false);
  const [executing, setExecuting] = useState(false);

  const loadRecommendationsData = async () => {
    try {
      setError(null);
      const data = await mockApi.getRecommendations();
      setRecommendations(data.recommendations);
      setFilteredRecommendations(data.recommendations);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load recommendations');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    mockApi.clearCache();
    await loadRecommendationsData();
  };

  const handleExecuteRecommendations = async (recommendationIds?: string[]) => {
    setExecuting(true);
    try {
      await mockApi.executeRecommendations(recommendationIds);
      await loadRecommendationsData();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to execute recommendations');
    } finally {
      setExecuting(false);
    }
  };

  useEffect(() => {
    loadRecommendationsData();
  }, []);

  useEffect(() => {
    let filtered = recommendations;

    // Filter by search term
    if (searchTerm) {
      filtered = filtered.filter(item => 
        item.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.resource_name?.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    // Filter by priority
    if (priorityFilter !== 'all') {
      filtered = filtered.filter(item => item.priority === priorityFilter);
    }

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter(item => item.status === statusFilter);
    }

    setFilteredRecommendations(filtered);
  }, [recommendations, searchTerm, priorityFilter, statusFilter]);

  const getPriorityColor = (priority: string) => {
    switch (priority.toLowerCase()) {
      case 'critical': return 'error';
      case 'high': return 'warning';
      case 'medium': return 'info';
      case 'low': return 'success';
      default: return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active': return 'info';
      case 'completed': return 'success';
      case 'pending': return 'warning';
      default: return 'default';
    }
  };

  const formatCurrency = (value: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);

  const totalSavings = filteredRecommendations.reduce((sum, item) => sum + item.estimated_savings_usd, 0);
  const activeRecommendations = filteredRecommendations.filter(item => item.status === 'active');

  if (loading) {
    return (
      <div className="space-y-6">
        {/* Header Skeleton */}
        <div className="flex items-center justify-between">
          <div>
            <div className="h-8 bg-neutral-200 rounded w-64 mb-2"></div>
            <div className="h-4 bg-neutral-200 rounded w-80"></div>
          </div>
          <div className="flex items-center space-x-3">
            <div className="h-10 bg-neutral-200 rounded w-32"></div>
            <div className="h-10 bg-neutral-200 rounded w-24"></div>
          </div>
        </div>

        {/* Stats Skeleton */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-neutral-200 p-4">
              <div className="animate-pulse">
                <div className="h-4 bg-neutral-200 rounded w-20 mb-2"></div>
                <div className="h-6 bg-neutral-200 rounded w-16"></div>
              </div>
            </div>
          ))}
        </div>

        {/* Recommendations Skeleton */}
        <div className="space-y-4">
          {[1, 2].map((i) => (
            <div key={i} className="bg-white rounded-xl border border-neutral-200 p-6">
              <div className="animate-pulse space-y-4">
                <div className="h-6 bg-neutral-200 rounded w-3/4"></div>
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
            <h3 className="text-lg font-medium text-red-800">Error Loading Recommendations</h3>
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
          <h1 className="text-2xl font-bold text-neutral-900">Optimization Recommendations</h1>
          <p className="text-neutral-600 mt-1">AI-powered suggestions to reduce costs and improve efficiency</p>
        </div>
        <div className="flex items-center space-x-3">
          <Button 
            variant="primary"
            onClick={() => handleExecuteRecommendations(activeRecommendations.map(r => r.id))}
            loading={executing}
            disabled={activeRecommendations.length === 0}
          >
            Execute All ({activeRecommendations.length})
          </Button>
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
              <p className="text-2xl font-bold text-neutral-900">{filteredRecommendations.length}</p>
            </div>
            <div className="p-2 bg-blue-50 rounded-lg">
              <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Active</p>
              <p className="text-2xl font-bold text-blue-600">{activeRecommendations.length}</p>
            </div>
            <Badge variant="info">Active</Badge>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">High Priority</p>
              <p className="text-2xl font-bold text-orange-600">
                {filteredRecommendations.filter(item => item.priority === 'high' || item.priority === 'critical').length}
              </p>
            </div>
            <Badge variant="warning">High</Badge>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Est. Savings</p>
              <p className="text-2xl font-bold text-green-600">{formatCurrency(totalSavings)}</p>
            </div>
            <div className="p-2 bg-green-50 rounded-lg">
              <svg className="w-5 h-5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
          </div>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center space-x-4">
        <div className="flex-1 max-w-md">
          <SearchInput
            placeholder="Search recommendations..."
            value={searchTerm}
            onSearch={setSearchTerm}
          />
        </div>
        
        <select
          value={priorityFilter}
          onChange={(e) => setPriorityFilter(e.target.value)}
          className="px-3 py-2 border border-neutral-300 rounded-lg bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="all">All Priorities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border border-neutral-300 rounded-lg bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="all">All Statuses</option>
          <option value="active">Active</option>
          <option value="pending">Pending</option>
          <option value="completed">Completed</option>
        </select>
      </div>

      {/* Recommendations List */}
      {filteredRecommendations.length === 0 ? (
        <Card>
          <div className="p-12">
            <EmptyState
              icon={
                <svg className="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
              }
              title="No Recommendations Found"
              description={searchTerm || priorityFilter !== 'all' || statusFilter !== 'all' ? 'No recommendations match your filters' : 'All optimizations have been completed'}
              action={
                (searchTerm || priorityFilter !== 'all' || statusFilter !== 'all') && (
                  <Button variant="secondary" onClick={() => {
                    setSearchTerm('');
                    setPriorityFilter('all');
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
          {filteredRecommendations.map((item) => (
            <Card key={item.id} className="p-6">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center space-x-3 mb-3">
                    <h3 className="text-lg font-semibold text-neutral-900">{item.title}</h3>
                    <Badge variant={getPriorityColor(item.priority) as any}>
                      {item.priority}
                    </Badge>
                    <Badge variant={getStatusColor(item.status) as any}>
                      {item.status}
                    </Badge>
                  </div>
                  
                  <p className="text-neutral-600 mb-4">{item.description}</p>
                  
                  <div className="flex items-center space-x-6 text-sm mb-4">
                    <div className="flex items-center space-x-2">
                      <span className="text-neutral-500">Resource:</span>
                      <span className="font-medium text-neutral-900">{item.resource_name || item.resource_id}</span>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="text-neutral-500">Type:</span>
                      <Badge variant="info">{item.resource_type}</Badge>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="text-neutral-500">Risk:</span>
                      <span className="font-medium text-neutral-900">{item.risk_level}</span>
                    </div>
                  </div>
                </div>
                
                <div className="ml-6 text-right">
                  <div className="text-2xl font-bold text-green-600">
                    {formatCurrency(item.estimated_savings_usd)}
                  </div>
                  <div className="text-xs text-neutral-500 mt-1">Est. Savings</div>
                  {item.status === 'active' && (
                    <Button 
                      size="sm" 
                      className="mt-3"
                      onClick={() => handleExecuteRecommendations([item.id])}
                      loading={executing}
                    >
                      Execute
                    </Button>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
