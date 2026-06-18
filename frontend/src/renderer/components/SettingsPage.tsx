import React, { useState, useEffect } from 'react';
import { useAppStore } from '../stores/appStore';
import type { AppConfig } from '../types';
import { ShieldIcon, MoonIcon, SunIcon, MonitorIcon, TrashIcon, PlusIcon, XIcon } from './Icons';
import { formatBytes } from '../utils/format';

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

const Toggle: React.FC<{ value: boolean; onChange: (v: boolean) => void }> = ({ value, onChange }) => (
  <div
    className={`toggle-switch ${value ? 'on' : ''}`}
    onClick={() => onChange(!value)}
  />
);

export const SettingsPage: React.FC = () => {
  const { config, saveConfig, stats, clearAll, loadStats, loadItems } = useAppStore();
  const [localConfig, setLocalConfig] = useState<AppConfig>(defaultConfig);
  const [newApp, setNewApp] = useState('');
  const [newPattern, setNewPattern] = useState('');
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (config) {
      setLocalConfig(config);
    }
  }, [config]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  const handleSave = async () => {
    try {
      await saveConfig(localConfig);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (err) {
      console.error('Failed to save config:', err);
    }
  };

  const handleAddSensitiveApp = () => {
    if (!newApp.trim()) return;
    setLocalConfig({
      ...localConfig,
      ignoredApps: [...localConfig.ignoredApps, newApp.trim()],
    });
    setNewApp('');
  };

  const handleRemoveSensitiveApp = (app: string) => {
    setLocalConfig({
      ...localConfig,
      ignoredApps: localConfig.ignoredApps.filter((a) => a !== app),
    });
  };

  const handleAddPattern = () => {
    if (!newPattern.trim()) return;
    setLocalConfig({
      ...localConfig,
      blacklistPatterns: [...localConfig.blacklistPatterns, newPattern.trim()],
    });
    setNewPattern('');
  };

  const handleRemovePattern = (pattern: string) => {
    setLocalConfig({
      ...localConfig,
      blacklistPatterns: localConfig.blacklistPatterns.filter((p) => p !== pattern),
    });
  };

  const handleClearHistory = async () => {
    if (window.confirm('确定要清除所有非收藏的历史记录吗？此操作无法撤销。')) {
      await clearAll();
      loadItems();
      loadStats();
    }
  };

  const handleQuit = () => {
    if (window.pasteAPI) {
      window.pasteAPI.quit();
    }
  };

  return (
    <div className="h-full overflow-y-auto p-5">
      <div className="max-w-xl mx-auto space-y-6">
        <div>
          <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">通用设置</h2>
          <div className="glass-content rounded-xl p-4 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-gray-800 dark:text-gray-100">开机自启动</div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">登录时自动启动 Paste</div>
              </div>
              <Toggle
                value={localConfig.autoStart}
                onChange={(v) => setLocalConfig({ ...localConfig, autoStart: v })}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-gray-800 dark:text-gray-100">双击自动粘贴</div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">双击条目时自动粘贴到当前应用</div>
              </div>
              <Toggle
                value={localConfig.autoPaste}
                onChange={(v) => setLocalConfig({ ...localConfig, autoPaste: v })}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-gray-800 dark:text-gray-100">最大历史记录</div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">超出数量的旧记录将被自动清理</div>
              </div>
              <select
                value={localConfig.maxHistory}
                onChange={(e) =>
                  setLocalConfig({ ...localConfig, maxHistory: parseInt(e.target.value) })
                }
                className="px-3 py-1.5 rounded-lg text-sm bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-100 outline-none border-0"
              >
                <option value={1000}>1,000 条</option>
                <option value={2500}>2,500 条</option>
                <option value={5000}>5,000 条</option>
                <option value={10000}>10,000 条</option>
                <option value={20000}>20,000 条</option>
              </select>
            </div>
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">外观</h2>
          <div className="glass-content rounded-xl p-4">
            <div className="flex items-center gap-3">
              {(['light', 'dark', 'system'] as const).map((theme) => (
                <button
                  key={theme}
                  onClick={() => setLocalConfig({ ...localConfig, theme })}
                  className={`flex-1 flex flex-col items-center gap-2 py-3 rounded-xl border transition-all ${
                    localConfig.theme === theme
                      ? 'border-blue-500 bg-blue-500/10'
                      : 'border-transparent bg-gray-100/50 dark:bg-gray-700/50 hover:bg-gray-200/50 dark:hover:bg-gray-600/50'
                  }`}
                >
                  {theme === 'light' && <SunIcon size={20} />}
                  {theme === 'dark' && <MoonIcon size={20} />}
                  {theme === 'system' && <MonitorIcon size={20} />}
                  <span className="text-xs font-medium text-gray-700 dark:text-gray-300">
                    {theme === 'light' ? '浅色' : theme === 'dark' ? '深色' : '跟随系统'}
                  </span>
                </button>
              ))}
            </div>
          </div>
        </div>

        <div>
          <div className="flex items-center gap-2 mb-4">
            <ShieldIcon size={18} />
            <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100">隐私与安全</h2>
          </div>

          <div className="glass-content rounded-xl p-4 space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-gray-800 dark:text-gray-100">启用敏感内容保护</div>
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                  自动忽略密码管理器、银行应用等敏感场景
                </div>
              </div>
              <Toggle
                value={localConfig.enableSensitive}
                onChange={(v) => setLocalConfig({ ...localConfig, enableSensitive: v })}
              />
            </div>

            <div>
              <div className="text-sm font-medium text-gray-800 dark:text-gray-100 mb-2">忽略的应用</div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                这些应用中的复制内容不会被记录
              </div>
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newApp}
                  onChange={(e) => setNewApp(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleAddSensitiveApp()}
                  placeholder="输入应用名称"
                  className="flex-1 px-3 py-1.5 rounded-lg text-sm bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-100 outline-none border-0 placeholder-gray-400"
                />
                <button onClick={handleAddSensitiveApp} className="action-btn">
                  <PlusIcon size={16} />
                </button>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {localConfig.ignoredApps.map((app) => (
                  <span
                    key={app}
                    className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
                  >
                    {app}
                    <button onClick={() => handleRemoveSensitiveApp(app)} className="hover:text-red-500">
                      <XIcon size={12} />
                    </button>
                  </span>
                ))}
                {localConfig.ignoredApps.length === 0 && (
                  <span className="text-xs text-gray-400">暂无自定义忽略应用</span>
                )}
              </div>
            </div>

            <div>
              <div className="text-sm font-medium text-gray-800 dark:text-gray-100 mb-2">内容黑名单</div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                匹配以下正则表达式的内容将被忽略
              </div>
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newPattern}
                  onChange={(e) => setNewPattern(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleAddPattern()}
                  placeholder="输入正则表达式"
                  className="flex-1 px-3 py-1.5 rounded-lg text-sm font-mono bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-100 outline-none border-0 placeholder-gray-400"
                />
                <button onClick={handleAddPattern} className="action-btn">
                  <PlusIcon size={16} />
                </button>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {localConfig.blacklistPatterns.map((pattern) => (
                  <span
                    key={pattern}
                    className="inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-mono bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 max-w-full"
                  >
                    <span className="truncate max-w-[200px]">{pattern}</span>
                    <button onClick={() => handleRemovePattern(pattern)} className="hover:text-red-500 flex-shrink-0">
                      <XIcon size={12} />
                    </button>
                  </span>
                ))}
                {localConfig.blacklistPatterns.length === 0 && (
                  <span className="text-xs text-gray-400">暂无自定义黑名单规则</span>
                )}
              </div>
            </div>
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-4">数据管理</h2>
          <div className="glass-content rounded-xl p-4 space-y-4">
            {stats && (
              <div className="grid grid-cols-2 gap-3">
                <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-gray-800/50">
                  <div className="text-2xl font-semibold text-gray-800 dark:text-gray-100">{stats.totalItems}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">总记录数</div>
                </div>
                <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-gray-800/50">
                  <div className="text-2xl font-semibold text-gray-800 dark:text-gray-100">{formatBytes(stats.totalSize)}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">占用空间</div>
                </div>
                <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-gray-800/50">
                  <div className="text-2xl font-semibold text-gray-800 dark:text-gray-100">{stats.todayCount}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">今日新增</div>
                </div>
                <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-gray-800/50">
                  <div className="text-2xl font-semibold text-paste-warning">{stats.favoriteItems}</div>
                  <div className="text-xs text-gray-500 dark:text-gray-400">收藏数</div>
                </div>
              </div>
            )}

            <button
              onClick={handleClearHistory}
              className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg text-sm font-medium text-red-500 bg-red-50 dark:bg-red-500/10 hover:bg-red-100 dark:hover:bg-red-500/20 transition-colors"
            >
              <TrashIcon size={16} />
              清除历史记录（保留收藏）
            </button>
          </div>
        </div>

        <div className="flex items-center justify-between pt-2 pb-4">
          <button
            onClick={handleQuit}
            className="btn-secondary text-gray-600 dark:text-gray-300"
          >
            退出应用
          </button>

          <div className="flex items-center gap-3">
            {saved && <span className="text-sm text-green-500">已保存 ✓</span>}
            <button onClick={handleSave} className="btn-primary">
              保存设置
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
