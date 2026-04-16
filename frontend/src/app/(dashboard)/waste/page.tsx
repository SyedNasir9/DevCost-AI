'use client';

import { useEffect, useState } from 'react';
import { mockApi, WasteResource } from '@/lib/mock-service';
import { Card, Badge, Button, SearchInput, EmptyState, Skeleton } from '@/components/ui';

export default function WastePage() {
  const [waste, setWaste] = useState<WasteResource[]>([]);
  const [filteredWaste, setFilteredWaste] = useState<WasteResource[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [refreshing, setRefreshing] = useState(false);

  const loadWasteData = async () => {
    try {
      setError(null);
      const data = await mockApi.getWaste();
      setWaste(data.waste);
      setFilteredWaste(data.waste);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load waste data');
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    mockApi.clearCache();
    await loadWasteData();
  };

  useEffect(() => {
    loadWasteData();
  }, []);

  useEffect(() => {
    let filtered = waste;

    // Filter by search term
    if (searchTerm) {
      filtered = filtered.filter(item => 
        item.resource_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.resource_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        item.reason.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }

    // Filter by severity
    if (severityFilter !== 'all') {
      filtered = filtered.filter(item => item.severity === severityFilter);
    }

    setFilteredWaste(filtered);
  }, [waste, searchTerm, severityFilter]);

  const getSeverityColor = (severity: string) => {
    switch (severity.toLowerCase()) {
      case 'high': return 'error';
      case 'medium': return 'warning';
      case 'low': return 'success';
      default: return 'default';
    }
  };

  const formatCurrency = (value: number) =>
    new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(value);

  const totalSavings = filteredWaste.reduce((sum, item) => sum + item.estimated_savings_usd, 0);

  if (loading) {
    return (
      <div className="space-y-6">
        {/* Header Skeleton */}
        <div className="flex items-center justify-between">
          <div>
            <div className="h-8 bg-neutral-200 rounded w-48 mb-2"></div>
            <div className="h-4 bg-neutral-200 rounded w-64"></div>
          </div>
          <div className="h-10 bg-neutral-200 rounded w-24"></div>
        </div>

        {/* Filters Skeleton */}
        <div className="flex items-center space-x-4">
          <div className="h-10 bg-neutral-200 rounded w-64"></div>
          <div className="h-10 bg-neutral-200 rounded w-32"></div>
        </div>

        {/* Table Skeleton */}
        <Card>
          <div className="p-6 space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse">
                <div className="h-16 bg-neutral-200 rounded"></div>
              </div>
            ))}
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <Card className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-red-800">Error Loading Waste Data</h3>
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
          <h1 className="text-2xl font-bold text-neutral-900">Resource Waste Analysis</h1>
          <p className="text-neutral-600 mt-1">Identify and eliminate wasted cloud resources</p>
        </div>
        <div className="flex items-center space-x-3">
          <div className="text-right">
            <div className="text-sm text-neutral-500">Total Resources</div>
            <div className="text-lg font-semibold text-neutral-900">{filteredWaste.length}</div>
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
              <p className="text-sm text-neutral-600">Total Waste</p>
              <p className="text-2xl font-bold text-neutral-900">{filteredWaste.length}</p>
            </div>
            <div className="p-2 bg-red-50 rounded-lg">
              <svg className="w-5 h-5 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
              </svg>
            </div>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">High Severity</p>
              <p className="text-2xl font-bold text-red-600">
                {filteredWaste.filter(item => item.severity === 'high').length}
              </p>
            </div>
            <Badge variant="error">High</Badge>
          </div>
        </Card>

        <Card className="p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-neutral-600">Medium Severity</p>
              <p className="text-2xl font-bold text-yellow-600">
                {filteredWaste.filter(item => item.severity === 'medium').length}
              </p>
            </div>
            <Badge variant="warning">Medium</Badge>
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
            placeholder="Search resources..."
            value={searchTerm}
            onSearch={setSearchTerm}
          />
        </div>
        
        <select
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className="px-3 py-2 border border-neutral-300 rounded-lg bg-white text-neutral-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="all">All Severities</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      </div>

      {/* Waste List */}
      <Card>
        {filteredWaste.length === 0 ? (
          <div className="p-12">
            <EmptyState
              icon={
                <svg className="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              }
              title="No Waste Detected"
              description={searchTerm || severityFilter !== 'all' ? 'No resources match your filters' : 'Your resources are optimized efficiently'}
              action={
                (searchTerm || severityFilter !== 'all') && (
                  <Button variant="secondary" onClick={() => {
                    setSearchTerm('');
                    setSeverityFilter('all');
                  }}>
                    Clear Filters
                  </Button>
                )
              }
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-neutral-200">
                  <th className="text-left py-3 px-6 text-xs font-medium text-neutral-500 uppercase tracking-wider">
                    Resource
                  </th>
                  <th className="text-left py-3 px-6 text-xs font-medium text-neutral-500 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="text-left py-3 px-6 text-xs font-medium text-neutral-500 uppercase tracking-wider">
                    Reason
                  </th>
                  <th className="text-left py-3 px-6 text-xs font-medium text-neutral-500 uppercase tracking-wider">
                    Severity
                  </th>
                  <th className="text-left py-3 px-6 text-xs font-medium text-neutral-500 uppercase tracking-wider">
                    Est. Savings
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {filteredWaste.map((item, index) => (
                  <tr key={item.resource_id || index} className="hover:bg-neutral-50 transition-colors">
                    <td className="py-4 px-6">
                      <div>
                        <div className="text-sm font-medium text-neutral-900">
                          {item.resource_name || item.resource_id}
                        </div>
                        <div className="text-xs text-neutral-500 mt-1">{item.resource_id}</div>
                      </div>
                    </td>
                    <td className="py-4 px-6">
                      <Badge variant="info">{item.resource_type}</Badge>
                    </td>
                    <td className="py-4 px-6 text-sm text-neutral-600 max-w-xs truncate">
                      {item.reason}
                    </td>
                    <td className="py-4 px-6">
                      <Badge variant={getSeverityColor(item.severity) as any}>
                        {item.severity}
                      </Badge>
                    </td>
                    <td className="py-4 px-6">
                      <div className="text-sm font-semibold text-green-600">
                        {formatCurrency(item.estimated_savings_usd)}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}
