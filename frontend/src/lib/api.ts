const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

// Types matching backend responses
export interface Resource {
  id: string;
  resource_id: string;
  resource_type: string;
  name: string;
  region: string;
  state: string;
  tags: Record<string, string>;
  created_at: string;
}

export interface DashboardSummary {
  totalCost: number;
  wastePercentage: number;
  totalResources: number;
  monthlySavings: number;
  annualSavings: number;
}

export interface WasteResource {
  resource_id: string;
  resource_type: string;
  resource_name?: string;
  type: string;
  reason: string;
  severity: string;
  estimated_savings_usd: number;
  confidence: number;
}

export interface Recommendation {
  id: string;
  resource_id: string;
  resource_type: string;
  resource_name?: string;
  type: string;
  title: string;
  description: string;
  priority: string;
  status: string;
  estimated_savings_usd: number;
  risk_level: string;
}

export interface Action {
  id: string;
  resource_id: string;
  resource_type: string;
  action_type: string;
  status: string;
  executed_at?: string;
  duration_ms?: number;
  error_message?: string;
  created_at: string;
}

// API Response types
interface WasteListResponse {
  success: boolean;
  count: number;
  waste: WasteResource[];
  total_estimated_savings_usd: number;
  high_priority_count: number;
  medium_priority_count: number;
  low_priority_count: number;
}

interface RecommendationsListResponse {
  success: boolean;
  count: number;
  recommendations: Recommendation[];
  total_estimated_savings_usd: number;
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
}

interface ActionsListResponse {
  success: boolean;
  count: number;
  actions: Action[];
}

interface ResourcesListResponse {
  success: boolean;
  resources: Resource[];
  total: number;
  page: number;
  page_size: number;
  has_next: boolean;
}

// API Client
class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }

  // Resources
  async getResources(): Promise<ResourcesListResponse> {
    return this.fetch('/resources');
  }

  // Waste
  async getWaste(): Promise<WasteListResponse> {
    return this.fetch('/waste');
  }

  // Recommendations
  async getRecommendations(): Promise<RecommendationsListResponse> {
    return this.fetch('/recommendations');
  }

  // Actions
  async getActions(): Promise<ActionsListResponse> {
    return this.fetch('/actions');
  }

  // Dashboard Summary (computed from waste + recommendations)
  async getDashboardSummary(): Promise<DashboardSummary> {
    const [wasteRes, recsRes, resourcesRes] = await Promise.all([
      this.getWaste().catch(() => ({ waste: [], total_estimated_savings_usd: 0, count: 0 })),
      this.getRecommendations().catch(() => ({ recommendations: [], total_estimated_savings_usd: 0 })),
      this.getResources().catch(() => ({ resources: [], total: 0 })),
    ]);

    const totalSavings = (wasteRes as WasteListResponse).total_estimated_savings_usd || 0;
    const wasteCount = (wasteRes as WasteListResponse).count || 0;
    const totalResources = (resourcesRes as ResourcesListResponse).total || 0;

    return {
      totalCost: 0, // Would need cost API endpoint
      wastePercentage: totalResources > 0 ? (wasteCount / totalResources) * 100 : 0,
      totalResources: totalResources,
      monthlySavings: totalSavings,
      annualSavings: totalSavings * 12,
    };
  }

  // Execute Recommendations
  async executeRecommendations(recommendationIds?: string[]): Promise<{ success: boolean; message: string }> {
    return this.fetch('/actions/execute', {
      method: 'POST',
      body: JSON.stringify(recommendationIds ? { recommendation_ids: recommendationIds } : { all_active: true }),
    });
  }
}

export const api = new ApiClient(API_BASE_URL);
