import { create } from 'zustand';
import type { ClipboardItem, AppConfig, Stats, ViewTab } from '../types';
import { api } from '../api/client';

interface AppState {
  items: ClipboardItem[];
  totalItems: number;
  loading: boolean;
  searchQuery: string;
  selectedIndex: number;
  activeTab: ViewTab;
  config: AppConfig | null;
  stats: Stats | null;
  isDark: boolean;

  setSearchQuery: (query: string) => void;
  setSelectedIndex: (index: number) => void;
  setActiveTab: (tab: ViewTab) => void;
  setIsDark: (dark: boolean) => void;

  loadItems: () => Promise<void>;
  searchItems: (query: string) => Promise<void>;
  loadConfig: () => Promise<void>;
  saveConfig: (config: AppConfig) => Promise<void>;
  loadStats: () => Promise<void>;

  toggleFavorite: (id: string, favorite: boolean) => Promise<void>;
  deleteItem: (id: string) => Promise<void>;
  copyItem: (id: string) => Promise<void>;
  pasteItem: (id: string) => Promise<void>;
  clearAll: () => Promise<void>;
}

const defaultConfig: AppConfig = {
  hotkey: 'Command+Shift+V',
  autoStart: false,
  theme: 'system',
  autoPaste: true,
  maxHistory: 5000,
  sensitiveApps: [],
  blacklistPatterns: [],
  enableSensitive: true,
  ignoredApps: [],
};

export const useAppStore = create<AppState>((set, get) => ({
  items: [],
  totalItems: 0,
  loading: false,
  searchQuery: '',
  selectedIndex: 0,
  activeTab: 'all',
  config: null,
  stats: null,
  isDark: false,

  setSearchQuery: (query) => {
    set({ searchQuery: query, selectedIndex: 0 });
    if (query) {
      get().searchItems(query);
    } else {
      get().loadItems();
    }
  },

  setSelectedIndex: (index) => {
    const total = get().items.length;
    if (index < 0) index = Math.max(0, total - 1);
    if (index >= total) index = 0;
    set({ selectedIndex: index });
  },

  setActiveTab: (tab) => {
    set({ activeTab: tab, selectedIndex: 0 });
    get().loadItems();
  },

  setIsDark: (dark) => set({ isDark: dark }),

  loadItems: async () => {
    set({ loading: true });
    try {
      const { activeTab } = get();
      let params: Record<string, unknown> = { offset: 0, limit: 100 };

      if (activeTab === 'favorites') {
        params.favorites = true;
      } else if (activeTab === 'text') {
        params.type = 'text';
      } else if (activeTab === 'images') {
        params.type = 'image';
      }

      const result = await api.listItems(params);
      set({
        items: result.items,
        totalItems: result.total,
        loading: false,
      });
    } catch (error) {
      console.error('Failed to load items:', error);
      set({ loading: false });
    }
  },

  searchItems: async (query: string) => {
    if (!query) {
      get().loadItems();
      return;
    }
    set({ loading: true });
    try {
      const { activeTab } = get();
      const result = await api.search(query, {
        offset: 0,
        limit: 100,
        favorites: activeTab === 'favorites',
      });
      set({
        items: result.items,
        totalItems: result.total,
        loading: false,
      });
    } catch (error) {
      console.error('Search failed:', error);
      set({ loading: false });
    }
  },

  loadConfig: async () => {
    try {
      let config: AppConfig;
      try {
        if (window.pasteAPI) {
          config = await window.pasteAPI.getConfig();
        } else {
          config = await api.getConfig();
        }
      } catch {
        config = { ...defaultConfig };
      }
      set({ config });
    } catch (error) {
      console.error('Failed to load config:', error);
      set({ config: { ...defaultConfig } });
    }
  },

  saveConfig: async (config: AppConfig) => {
    try {
      if (window.pasteAPI) {
        await window.pasteAPI.saveConfig(config);
      }
      const updated = await api.updateConfig(config);
      set({ config: updated });
    } catch (error) {
      console.error('Failed to save config:', error);
      throw error;
    }
  },

  loadStats: async () => {
    try {
      const stats = await api.getStats();
      set({ stats });
    } catch (error) {
      console.error('Failed to load stats:', error);
    }
  },

  toggleFavorite: async (id: string, favorite: boolean) => {
    try {
      await api.toggleFavorite(id, favorite);
      set((state) => ({
        items: state.items.map((item) =>
          item.id === id ? { ...item, isFavorite: favorite } : item
        ),
      }));
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
      throw error;
    }
  },

  deleteItem: async (id: string) => {
    try {
      await api.deleteItem(id);
      set((state) => ({
        items: state.items.filter((item) => item.id !== id),
        totalItems: state.totalItems - 1,
        selectedIndex: Math.min(state.selectedIndex, state.items.length - 2),
      }));
    } catch (error) {
      console.error('Failed to delete item:', error);
      throw error;
    }
  },

  copyItem: async (id: string) => {
    try {
      await api.copyItem(id);
      if (window.pasteAPI) {
        window.pasteAPI.hideWindow();
      }
    } catch (error) {
      console.error('Failed to copy item:', error);
      throw error;
    }
  },

  pasteItem: async (id: string) => {
    try {
      await api.pasteItem(id);
      if (window.pasteAPI) {
        window.pasteAPI.hideWindow();
      }
    } catch (error) {
      console.error('Failed to paste item:', error);
      throw error;
    }
  },

  clearAll: async () => {
    try {
      await api.clearAll(true);
      get().loadItems();
      get().loadStats();
    } catch (error) {
      console.error('Failed to clear items:', error);
      throw error;
    }
  },
}));
