import axios, { AxiosInstance } from 'axios';
import type {
  ClipboardItem,
  SearchResult,
  AppConfig,
  APIResponse,
  Stats,
} from '../types';

declare global {
  interface Window {
    pasteAPI: {
      getPort: () => Promise<number>;
      getConfig: () => Promise<AppConfig>;
      saveConfig: (config: AppConfig) => Promise<boolean>;
      hideWindow: () => Promise<void>;
      toggleWindow: () => Promise<void>;
      shouldUseDarkColors: () => Promise<boolean>;
      quit: () => Promise<void>;
      openExternal: (url: string) => Promise<void>;
      onThemeChanged: (callback: (isDark: boolean) => void) => void;
    };
  }
}

let apiClient: AxiosInstance | null = null;
let baseURL: string | null = null;

async function ensureClient(): Promise<AxiosInstance> {
  if (apiClient && baseURL) {
    return apiClient;
  }

  let port = 48175;
  try {
    if (window.pasteAPI) {
      port = await window.pasteAPI.getPort();
    }
  } catch {
    console.warn('Failed to get API port, using default');
  }

  baseURL = `http://127.0.0.1:${port}/api/v1`;
  apiClient = axios.create({
    baseURL,
    timeout: 5000,
  });

  apiClient.interceptors.response.use(
    (response) => response,
    (error) => {
      console.error('API Error:', error.message);
      return Promise.reject(error);
    }
  );

  return apiClient;
}

async function request<T>(url: string, config: Record<string, unknown> = {}): Promise<T> {
  const client = await ensureClient();
  const response = await client.request<APIResponse<T>>({ url, ...config });
  if (!response.data.success) {
    throw new Error(response.data.error || 'Request failed');
  }
  return response.data.data as T;
}

export const api = {
  async healthCheck(): Promise<{ status: string; version: string; pid: number }> {
    return request('/health', { method: 'GET' });
  },

  async listItems(params: {
    offset?: number;
    limit?: number;
    favorites?: boolean;
    type?: string;
  } = {}): Promise<SearchResult> {
    const client = await ensureClient();
    const response = await client.get<APIResponse<SearchResult>>('/items', { params });
    if (!response.data.success) {
      throw new Error(response.data.error || 'Failed to list items');
    }
    return response.data.data!;
  },

  async getItem(id: string): Promise<ClipboardItem> {
    return request(`/items/${id}`, { method: 'GET' });
  },

  async deleteItem(id: string): Promise<void> {
    return request(`/items/${id}`, { method: 'DELETE' });
  },

  async toggleFavorite(id: string, favorite: boolean): Promise<ClipboardItem> {
    return request(`/items/${id}/favorite`, {
      method: 'PUT',
      data: { favorite },
    });
  },

  async copyItem(id: string): Promise<void> {
    return request(`/items/${id}/copy`, { method: 'POST' });
  },

  async pasteItem(id: string): Promise<void> {
    return request(`/items/${id}/paste`, { method: 'POST' });
  },

  async search(query: string, params: { offset?: number; limit?: number; favorites?: boolean } = {}): Promise<SearchResult> {
    const client = await ensureClient();
    const response = await client.get<APIResponse<SearchResult>>('/search', {
      params: { q: query, ...params },
    });
    if (!response.data.success) {
      throw new Error(response.data.error || 'Search failed');
    }
    return response.data.data!;
  },

  async getStats(): Promise<Stats> {
    return request('/stats', { method: 'GET' });
  },

  async clearAll(keepFavorites = true): Promise<void> {
    const client = await ensureClient();
    await client.delete<APIResponse<void>>('/items', { params: { keepFavorites } });
  },

  async getConfig(): Promise<AppConfig> {
    return request('/config', { method: 'GET' });
  },

  async updateConfig(config: AppConfig): Promise<AppConfig> {
    return request('/config', {
      method: 'PUT',
      data: config,
    });
  },

  async getAutostart(): Promise<{ enabled: boolean }> {
    return request('/autostart', { method: 'GET' });
  },

  async setAutostart(enabled: boolean): Promise<{ enabled: boolean }> {
    return request('/autostart', {
      method: 'PUT',
      data: { enabled },
    });
  },

  async getSensitiveApps(): Promise<{ apps: string[] }> {
    return request('/sensitive-apps', { method: 'GET' });
  },

  getImageUrl(id: string): string {
    return `${baseURL}/images/${id}`;
  },
};
