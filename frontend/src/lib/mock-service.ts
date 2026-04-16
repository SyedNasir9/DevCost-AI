// Mock Data Service - Production-grade SaaS simulation
import { DashboardSummary, WasteResource, Recommendation, Action } from './api';

// Simulate API delays
const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

// Random delay between 300-800ms
const randomDelay = () => delay(Math.floor(Math.random() * 500) + 300);

// Mock data generators
const generateMockWasteResources = (): WasteResource[] => [
  {
    resource_id: 'i-0abcd1234efgh5678',
    resource_type: 'EC2 Instance',
    resource_name: 'prod-web-server-01',
    type: 'idle_resource',
    reason: 'Instance has been running with <1% CPU utilization for 7 days',
    severity: 'high',
    estimated_savings_usd: 127.50,
    confidence: 0.95,
  },
  {
    resource_id: 'vol-0123456789abcdef0',
    resource_type: 'EBS Volume',
    resource_name: 'unattached-data-volume',
    type: 'unattached_volume',
    reason: 'EBS volume not attached to any instance for 30+ days',
    severity: 'medium',
    estimated_savings_usd: 45.30,
    confidence: 0.88,
  },
  {
    resource_id: 'arn:aws:s3:::unused-logs-bucket',
    resource_type: 'S3 Bucket',
    resource_name: 'unused-logs-bucket',
    type: 'underutilized_storage',
    reason: 'S3 bucket with minimal activity but high storage costs',
    severity: 'low',
    estimated_savings_usd: 12.75,
    confidence: 0.92,
  },
  {
    resource_id: 'i-0fghijklmnop12345',
    resource_type: 'EC2 Instance',
    resource_name: 'staging-db-server',
    type: 'overprovisioned',
    reason: 'Instance size exceeds actual usage patterns',
    severity: 'medium',
    estimated_savings_usd: 89.60,
    confidence: 0.85,
  },
];

const generateMockRecommendations = (): Recommendation[] => [
  {
    id: 'rec-001',
    resource_id: 'i-0abcd1234efgh5678',
    resource_type: 'EC2 Instance',
    resource_name: 'prod-web-server-01',
    type: 'rightsize_instance',
    title: 'Downsize EC2 instance from t3.large to t3.medium',
    description: 'Based on CPU and memory usage patterns, this instance can be safely downsized to reduce costs while maintaining performance.',
    priority: 'high',
    status: 'active',
    estimated_savings_usd: 127.50,
    risk_level: 'low',
  },
  {
    id: 'rec-002',
    resource_id: 'vol-0123456789abcdef0',
    resource_type: 'EBS Volume',
    resource_name: 'unattached-data-volume',
    type: 'delete_volume',
    title: 'Delete unattached EBS volume',
    description: 'This volume is not attached to any instance and can be safely deleted to eliminate storage costs.',
    priority: 'medium',
    status: 'active',
    estimated_savings_usd: 45.30,
    risk_level: 'low',
  },
  {
    id: 'rec-003',
    resource_id: 'i-0fghijklmnop12345',
    resource_type: 'EC2 Instance',
    resource_name: 'staging-db-server',
    type: 'rightsize_instance',
    title: 'Rightsize staging database server',
    description: 'Consider switching to a smaller instance type or using reserved instances for better cost efficiency.',
    priority: 'medium',
    status: 'pending',
    estimated_savings_usd: 89.60,
    risk_level: 'medium',
  },
  {
    id: 'rec-004',
    resource_id: 'arn:aws:s3:::unused-logs-bucket',
    resource_type: 'S3 Bucket',
    resource_name: 'unused-logs-bucket',
    type: 'optimize_storage',
    title: 'Implement S3 lifecycle policies',
    description: 'Add lifecycle policies to transition old logs to cheaper storage tiers or delete them automatically.',
    priority: 'low',
    status: 'active',
    estimated_savings_usd: 12.75,
    risk_level: 'low',
  },
];

const generateMockActions = (): Action[] => [
  {
    id: 'action-001',
    resource_id: 'i-0abcd1234efgh5678',
    resource_type: 'EC2 Instance',
    action_type: 'stop',
    status: 'completed',
    executed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    duration_ms: 4500,
    created_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: 'action-002',
    resource_id: 'vol-0123456789abcdef0',
    resource_type: 'EBS Volume',
    action_type: 'delete',
    status: 'failed',
    executed_at: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
    duration_ms: 2300,
    error_message: 'Volume is still attached to an instance',
    created_at: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: 'action-003',
    resource_id: 'i-0fghijklmnop12345',
    resource_type: 'EC2 Instance',
    action_type: 'resize',
    status: 'running',
    executed_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
    duration_ms: null,
    created_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  },
];

const generateMockDashboardSummary = (): DashboardSummary => {
  const waste = generateMockWasteResources();
  const recommendations = generateMockRecommendations();
  
  const totalSavings = waste.reduce((sum, item) => sum + item.estimated_savings_usd, 0);
  const totalResources = 156; // Mock total resource count
  
  return {
    totalCost: 2457.80,
    wastePercentage: (waste.length / totalResources) * 100,
    totalResources: totalResources,
    monthlySavings: totalSavings,
    annualSavings: totalSavings * 12,
  };
};

// Mock API Service
export class MockApiService {
  private wasteCache: WasteResource[] | null = null;
  private recommendationsCache: Recommendation[] | null = null;
  private actionsCache: Action[] | null = null;
  private summaryCache: DashboardSummary | null = null;

  async getWaste() {
    await randomDelay();
    
    if (!this.wasteCache) {
      this.wasteCache = generateMockWasteResources();
    }
    
    return {
      success: true,
      count: this.wasteCache.length,
      waste: this.wasteCache,
      total_estimated_savings_usd: this.wasteCache.reduce((sum, item) => sum + item.estimated_savings_usd, 0),
      high_priority_count: this.wasteCache.filter(item => item.severity === 'high').length,
      medium_priority_count: this.wasteCache.filter(item => item.severity === 'medium').length,
      low_priority_count: this.wasteCache.filter(item => item.severity === 'low').length,
    };
  }

  async getRecommendations() {
    await randomDelay();
    
    if (!this.recommendationsCache) {
      this.recommendationsCache = generateMockRecommendations();
    }
    
    return {
      success: true,
      count: this.recommendationsCache.length,
      recommendations: this.recommendationsCache,
      total_estimated_savings_usd: this.recommendationsCache.reduce((sum, item) => sum + item.estimated_savings_usd, 0),
      critical_count: this.recommendationsCache.filter(item => item.priority === 'critical').length,
      high_count: this.recommendationsCache.filter(item => item.priority === 'high').length,
      medium_count: this.recommendationsCache.filter(item => item.priority === 'medium').length,
      low_count: this.recommendationsCache.filter(item => item.priority === 'low').length,
    };
  }

  async getActions() {
    await randomDelay();
    
    if (!this.actionsCache) {
      this.actionsCache = generateMockActions();
    }
    
    return {
      success: true,
      count: this.actionsCache.length,
      actions: this.actionsCache,
    };
  }

  async getResources() {
    await randomDelay();
    
    // Mock resources data
    const resources = [
      {
        id: '1',
        resource_id: 'i-0abcd1234efgh5678',
        resource_type: 'EC2 Instance',
        name: 'prod-web-server-01',
        region: 'us-east-1',
        state: 'running',
        tags: { Environment: 'production', Service: 'web' },
        created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
      },
      // Add more mock resources as needed
    ];
    
    return {
      success: true,
      resources: resources,
      total: 156,
      page: 1,
      page_size: 50,
      has_next: true,
    };
  }

  async getDashboardSummary(): Promise<DashboardSummary> {
    await randomDelay();
    
    if (!this.summaryCache) {
      this.summaryCache = generateMockDashboardSummary();
    }
    
    return this.summaryCache;
  }

  async executeRecommendations(recommendationIds?: string[]) {
    await randomDelay();
    
    // Simulate execution
    const targetIds = recommendationIds || ['rec-001', 'rec-002'];
    
    // Update action status
    if (this.actionsCache) {
      const newAction: Action = {
        id: `action-${Date.now()}`,
        resource_id: 'i-0abcd1234efgh5678',
        resource_type: 'EC2 Instance',
        action_type: 'stop',
        status: 'completed',
        executed_at: new Date().toISOString(),
        duration_ms: Math.floor(Math.random() * 5000) + 1000,
        created_at: new Date().toISOString(),
      };
      this.actionsCache.unshift(newAction);
    }
    
    // Update recommendation status
    if (this.recommendationsCache) {
      targetIds.forEach(id => {
        const rec = this.recommendationsCache!.find(r => r.id === id);
        if (rec) {
          rec.status = 'completed';
        }
      });
    }
    
    return {
      success: true,
      message: `Successfully executed ${targetIds.length} recommendation(s)`,
    };
  }

  // Clear cache for testing
  clearCache() {
    this.wasteCache = null;
    this.recommendationsCache = null;
    this.actionsCache = null;
    this.summaryCache = null;
  }
}

export const mockApi = new MockApiService();
