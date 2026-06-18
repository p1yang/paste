export type ClipboardType = 'text' | 'image';

export interface ClipboardItem {
  id: string;
  type: ClipboardType;
  content?: string;
  imageUrl?: string;
  isFavorite: boolean;
  appName?: string;
  createdAt: string;
  updatedAt: string;
  pasteCount: number;
  sizeBytes: number;
}

export interface SearchResult {
  items: ClipboardItem[];
  total: number;
}

export interface AppConfig {
  hotkey: string;
  autoStart: boolean;
  theme: 'light' | 'dark' | 'system';
  autoPaste: boolean;
  maxHistory: number;
  sensitiveApps: string[];
  blacklistPatterns: string[];
  enableSensitive: boolean;
  ignoredApps: string[];
}

export interface APIResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Stats {
  totalItems: number;
  textItems: number;
  imageItems: number;
  favoriteItems: number;
  totalSize: number;
  todayCount: number;
}

export type ViewTab = 'all' | 'favorites' | 'text' | 'images' | 'settings';
