import { contextBridge, ipcRenderer } from 'electron';

export interface AppConfig {
  hotkey: string;
  autoStart: boolean;
  theme: 'light' | 'dark' | 'system';
  autoPaste: boolean;
  maxHistory: number;
}

interface PasteAPI {
  getPort: () => Promise<number>;
  getConfig: () => Promise<AppConfig>;
  saveConfig: (config: AppConfig) => Promise<boolean>;
  hideWindow: () => Promise<void>;
  toggleWindow: () => Promise<void>;
  shouldUseDarkColors: () => Promise<boolean>;
  quit: () => Promise<void>;
  openExternal: (url: string) => Promise<void>;
  onThemeChanged: (callback: (isDark: boolean) => void) => void;
}

const api: PasteAPI = {
  getPort: () => ipcRenderer.invoke('api:get-port'),
  getConfig: () => ipcRenderer.invoke('config:get'),
  saveConfig: (config: AppConfig) => ipcRenderer.invoke('config:save', config),
  hideWindow: () => ipcRenderer.invoke('window:hide'),
  toggleWindow: () => ipcRenderer.invoke('window:toggle'),
  shouldUseDarkColors: () => ipcRenderer.invoke('theme:should-use-dark-colors'),
  quit: () => ipcRenderer.invoke('app:quit'),
  openExternal: (url: string) => ipcRenderer.invoke('shell:open-external', url),
  onThemeChanged: (callback: (isDark: boolean) => void) => {
    ipcRenderer.on('theme:changed', (_event, isDark: boolean) => {
      callback(isDark);
    });
  },
};

contextBridge.exposeInMainWorld('pasteAPI', api);
