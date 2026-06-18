import { app, BrowserWindow, ipcMain, globalShortcut, screen, nativeTheme, shell } from 'electron';
import * as path from 'path';
import * as fs from 'fs';
import { spawn, ChildProcess } from 'child_process';
import Store = require('electron-store');

const store = new Store();

let mainWindow: BrowserWindow | null = null;
let backendProcess: ChildProcess | null = null;
const API_PORT = 48175;

interface AppConfig {
  hotkey: string;
  autoStart: boolean;
  theme: 'light' | 'dark' | 'system';
  autoPaste: boolean;
  maxHistory: number;
}

const defaultConfig: AppConfig = {
  hotkey: 'Command+Shift+V',
  autoStart: false,
  theme: 'system',
  autoPaste: true,
  maxHistory: 5000,
};

function getConfig(): AppConfig {
  return { ...defaultConfig, ...(store.get('config') as Partial<AppConfig>) };
}

function saveConfig(config: AppConfig): void {
  store.set('config', config);
}

function startBackend(): void {
  const isDev = !app.isPackaged;
  let backendPath: string;

  if (isDev) {
    backendPath = path.join(__dirname, '..', '..', '..', 'backend', 'bin', 'paste-backend');
  } else {
    backendPath = path.join(process.resourcesPath, 'bin', 'paste-backend');
  }

  const dataDir = path.join(app.getPath('userData'), 'data');
  if (!fs.existsSync(dataDir)) {
    fs.mkdirSync(dataDir, { recursive: true });
  }

  try {
    if (fs.existsSync(backendPath)) {
      backendProcess = spawn(backendPath, [
        '-port', String(API_PORT),
        '-data-dir', dataDir,
        '-max-history', String(getConfig().maxHistory),
      ], {
        stdio: isDev ? 'inherit' : 'ignore',
      });

      backendProcess.on('error', (err) => {
        console.error('Failed to start backend:', err);
      });

      backendProcess.on('exit', (code, signal) => {
        console.log(`Backend exited: code=${code}, signal=${signal}`);
      });
    } else {
      console.warn('Backend binary not found at:', backendPath);
    }
  } catch (err) {
    console.error('Error starting backend:', err);
  }
}

function stopBackend(): void {
  if (backendProcess) {
    backendProcess.kill('SIGTERM');
    backendProcess = null;
  }
}

function getWindowPosition(): { x: number; y: number } {
  const primaryDisplay = screen.getPrimaryDisplay();
  const { width: screenWidth } = primaryDisplay.workAreaSize;
  const windowWidth = 680;

  const x = Math.floor((screenWidth - windowWidth) / 2);
  const y = 120;

  return { x, y };
}

function createWindow(): void {
  const isDev = !app.isPackaged;
  const config = getConfig();
  const { x, y } = getWindowPosition();

  mainWindow = new BrowserWindow({
    width: 680,
    height: 560,
    x,
    y,
    frame: false,
    transparent: true,
    resizable: true,
    minimizable: false,
    maximizable: false,
    fullscreenable: false,
    alwaysOnTop: true,
    skipTaskbar: true,
    hasShadow: true,
    vibrancy: 'under-window',
    visualEffectState: 'active',
    backgroundColor: '#00000000',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  mainWindow.setWindowButtonVisibility(false);

  if (isDev) {
    mainWindow.loadURL('http://localhost:5173');
  } else {
    mainWindow.loadFile(path.join(__dirname, 'renderer', 'index.html'));
  }

  mainWindow.on('blur', () => {
    if (mainWindow && !mainWindow.webContents.isDevToolsOpened()) {
      mainWindow.hide();
    }
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  if (config.theme === 'system') {
    nativeTheme.themeSource = 'system';
  } else {
    nativeTheme.themeSource = config.theme;
  }
}

function toggleWindow(): void {
  if (!mainWindow) {
    createWindow();
    return;
  }

  if (mainWindow.isVisible()) {
    mainWindow.hide();
  } else {
    const { x, y } = getWindowPosition();
    mainWindow.setPosition(x, y);
    mainWindow.show();
    mainWindow.focus();
  }
}

function registerHotkey(): void {
  const config = getConfig();
  globalShortcut.unregisterAll();

  try {
    const ret = globalShortcut.register(config.hotkey, () => {
      toggleWindow();
    });

    if (!ret) {
      console.error('Hotkey registration failed');
    }
  } catch (err) {
    console.error('Error registering hotkey:', err);
  }
}

function setupIpcHandlers(): void {
  ipcMain.handle('api:get-port', () => API_PORT);

  ipcMain.handle('config:get', () => getConfig());

  ipcMain.handle('config:save', (_event, config: AppConfig) => {
    saveConfig(config);
    if (config.theme === 'system') {
      nativeTheme.themeSource = 'system';
    } else {
      nativeTheme.themeSource = config.theme;
    }
    registerHotkey();
    return true;
  });

  ipcMain.handle('window:hide', () => {
    if (mainWindow) {
      mainWindow.hide();
    }
  });

  ipcMain.handle('window:toggle', () => {
    toggleWindow();
  });

  ipcMain.handle('theme:should-use-dark-colors', () => {
    return nativeTheme.shouldUseDarkColors;
  });

  ipcMain.handle('app:quit', () => {
    app.quit();
  });

  ipcMain.handle('shell:open-external', (_event, url: string) => {
    shell.openExternal(url);
  });
}

app.whenReady().then(() => {
  startBackend();
  setupIpcHandlers();
  registerHotkey();

  nativeTheme.on('updated', () => {
    if (mainWindow) {
      mainWindow.webContents.send('theme:changed', nativeTheme.shouldUseDarkColors);
    }
  });

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('will-quit', () => {
  globalShortcut.unregisterAll();
  stopBackend();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});
