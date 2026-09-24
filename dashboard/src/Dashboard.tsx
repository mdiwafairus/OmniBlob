import React, { useEffect, useState } from 'react';
import { Activity, HardDrive, FileCheck, Clock, Settings, RefreshCw, BarChart2, PieChart as PieChartIcon, Server, AlertTriangle, Network, Zap, History, Archive, Folder, ChevronDown } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, Cell, PieChart, Pie } from 'recharts';
import { Logo } from './components/Logo';

interface MigrationJob {
  id: string;
  name: string;
  type: string;
  volume_bytes: number;
  date: string;
  status: 'Success' | 'Dry-Run' | 'Failed' | 'Running';
}

interface DashboardSummary {
  total_data_migrated_bytes: number;
  total_files_migrated: number;
  total_pending_files: number;
  overall_progress_percent: number;
  stale_files_over_1_year: number;
  migration_enabled: boolean;
  vm_total_bytes: number;
  vm_used_bytes: number;
  vm_free_bytes: number;
  vm_used_percent: number;
  root_path: string;
  legacy_path: string;
  sharding_type: string;
  clients: string[];
  // New metrics for distribution and history
  sync_rate_mbps?: number;
  connected_endpoints?: number;
  delta_changes_today?: number;
  last_synced_time?: string;
  total_lifetime_migrations?: number;
  migration_history?: MigrationJob[];
}

interface ExtensionStat {
  extension: string;
  count: number;
  size_bytes: number;
}

interface ModuleStat {
  module: string;
  count: number;
  size_bytes: number;
}

interface LargeFile {
  bin_id: number;
  file_name: string;
  path: string;
  size_bytes: number;
  extension: string;
}

interface YearStat {
  year: number;
  count: number;
  size_bytes: number;
}

interface MonthlyStat {
  year: number;
  month: number;
  count: number;
  size_bytes: number;
}

interface StorageAnalytics {
  extension_stats: ExtensionStat[];
  module_stats: ModuleStat[];
  top_large_files: LargeFile[];
  yearly_stats: YearStat[];
  monthly_stats: MonthlyStat[];
}

interface DuplicateGroup {
  checksum: string;
  count: number;
  size_bytes: number;
  wasted_bytes: number;
  example_name: string;
}

interface DataQualityStats {
  total_duplicate_files: number;
  total_wasted_bytes: number;
  duplicate_groups: DuplicateGroup[];
}

export function Dashboard() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [analyticsResponse, setAnalyticsResponse] = useState<StorageAnalytics | null>(null);
  const [qualityResponse, setQualityResponse] = useState<DataQualityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedYear, setSelectedYear] = useState<number>(new Date().getFullYear());
  
  // Interactive toggle for the client to preview the two states
  const [isDemoMigration, setIsDemoMigration] = useState(false);
  
  // State for global folder filter
  const [selectedFolder, setSelectedFolder] = useState<string>('all');

  const fetchData = async () => {
    try {
      const [sumRes, anaRes, qualRes] = await Promise.all([
        fetch('/api/v1/dashboard/summary'),
        fetch('/api/v1/dashboard/analytics'),
        fetch('/api/v1/dashboard/quality')
      ]);
      
      if (!sumRes.ok || !anaRes.ok || !qualRes.ok) throw new Error('Failed to fetch data');
      
      const sumData = await sumRes.json();
      const anaData = await anaRes.json();
      const qualData = await qualRes.json();
      
      setSummary(sumData);
      setAnalyticsResponse(anaData);
      setQualityResponse(qualData);
      setError('');
    } catch (err: any) {
      console.error(err);
      setError(err.message || 'Error connecting to API');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 10000); 
    return () => clearInterval(interval);
  }, []);

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (loading && !summary) {
    return (
      <div className="min-h-screen bg-panel2 flex items-center justify-center">
        <RefreshCw className="w-8 h-8 text-cy animate-spin" />
      </div>
    );
  }

  const rawData = summary || {
    total_data_migrated_bytes: 125000000000,
    total_files_migrated: 15420,
    total_pending_files: 5310,
    overall_progress_percent: 74.4,
    stale_files_over_1_year: 3412,
    migration_enabled: false,
    vm_total_bytes: 1500000000000,
    vm_used_bytes: 125000000000,
    vm_free_bytes: 1375000000000,
    vm_used_percent: 8.3,
    root_path: '/mnt/storage/omniblob',
    legacy_path: 'nfs://old-nas.internal/export',
    sharding_type: 'hash-based',
    clients: ['192.168.1.10', '192.168.1.11'],
    license_tier: 'ENTERPRISE',
    sync_rate_mbps: 24.5,
    connected_endpoints: 12,
    delta_changes_today: 142,
    last_synced_time: '2 mins ago',
    total_lifetime_migrations: 24,
    migration_history: [
      { id: 'job-0482', name: 'Finance-NFS-to-SMB', type: 'Full Migration', volume_bytes: 909772800000, date: '2026-09-12 22:00', duration: '2h 15m', status: 'Success' },
      { id: 'job-0481', name: 'HR-Archive-Sync', type: 'Delta Sync', volume_bytes: 45097156608, date: '2026-09-10 02:00', duration: '45m 12s', status: 'Success' },
      { id: 'job-0480', name: 'Legacy-SAN-Check', type: 'Dry-Run', volume_bytes: 1209772800000, date: '2026-09-08 14:30', duration: '1h 05m', status: 'Dry-Run' },
      { id: 'job-0479', name: 'Marketing-Media', type: 'Full Migration', volume_bytes: 3509772800000, date: '2026-09-01 00:00', duration: '8h 30m', status: 'Success' },
    ]
  };

  const isClientSelected = selectedFolder !== 'all';
  
  // Apply zeroing out if a specific client is selected and data is empty/not handled by backend yet
  const data = isClientSelected ? {
    ...rawData,
    total_data_migrated_bytes: 0,
    total_files_migrated: 0,
    total_pending_files: 0,
    overall_progress_percent: 0,
    stale_files_over_1_year: 0,
    vm_used_bytes: 0,
    vm_used_percent: 0,
    sync_rate_mbps: 0,
    connected_endpoints: 0,
    delta_changes_today: 0,
    total_lifetime_migrations: 0,
    migration_history: [],
    last_synced_time: 'Never'
  } : rawData;

  const analytics = isClientSelected ? null : analyticsResponse;
  const quality = isClientSelected ? null : qualityResponse;

  const COLORS = ['#5bc8dc', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#14b8a6', '#f43f5e'];

  // Determine active mode based on backend OR local demo toggle
  const showMigrationUI = summary && summary.migration_enabled !== undefined ? summary.migration_enabled : isDemoMigration;

  // Extract available folders/clients dynamically or fallback to mock
  const availableFolders = [...(rawData.clients || [])];
  
  // Include the literal folder 'legacy_path' in the dropdown if migration is active
  if (showMigrationUI && !availableFolders.includes('legacy_path')) {
      availableFolders.push('legacy_path');
  }

  return (
    <div className="flex-1 w-full font-sans">
      <header className="bg-panel border-b border-line px-6 py-4 flex items-center justify-between sticky top-0 z-50">
        <div className="flex items-center gap-4">
         <h2 className="text-lg font-semibold text-ink flex items-center gap-3">
            Overview
            {rawData.license_tier && (
               <span className={`px-2 py-0.5 rounded text-xs font-bold uppercase tracking-wider ${
                  rawData.license_tier.toUpperCase() === 'TRIAL' ? 'bg-amber/20 text-amber border border-amber/30' :
                  rawData.license_tier.toUpperCase() === 'PRO' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' :
                  'bg-cy/20 text-cy border border-cy/30'
               }`}>
                  {rawData.license_tier.toUpperCase() === 'ENTERPRISE' ? '👑 ENTERPRISE' : 
                   rawData.license_tier.toUpperCase() === 'PRO' ? '⭐ PRO' : 
                   '⏳ TRIAL'}
               </span>
            )}
         </h2>
           <div className="h-6 w-px bg-linesoft"></div>
           <div className="relative flex items-center group">
              <Folder className="absolute left-3 w-4 h-4 text-dim group-hover:text-cy transition-colors pointer-events-none" />
              <select 
                 className="appearance-none bg-panel2 text-ink text-sm font-medium outline-none cursor-pointer border border-line hover:border-cy/50 focus:border-cy rounded-lg pl-9 pr-9 py-1.5 transition-all shadow-sm"
                 value={selectedFolder}
                 onChange={(e) => setSelectedFolder(e.target.value)}
              >
                 <option value="all" className="bg-panel text-ink py-2">All Clients (Global View)</option>
                 {availableFolders.map(folder => (
                    <option key={folder} value={folder} className="bg-panel text-ink py-2">{folder}</option>
                 ))}
              </select>
              <ChevronDown className="absolute right-3 w-4 h-4 text-dim pointer-events-none group-hover:text-cy transition-colors" />
           </div>
        </div>
        
        {/* Toggle to let user switch modes for Demo */}
        <div className="flex items-center gap-2 bg-panel2 p-1 rounded-lg border border-line">
           <span className="text-xs font-semibold uppercase tracking-widest text-faint ml-2 mr-2">Demo Mode:</span>
           <button 
              onClick={() => setIsDemoMigration(false)}
              className={`px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${!isDemoMigration ? 'bg-panel shadow-sm text-cy' : 'text-dim hover:text-ink'}`}
           >
              Sync Only
           </button>
           <button 
              onClick={() => setIsDemoMigration(true)}
              className={`px-3 py-1.5 rounded-md text-xs font-semibold transition-all ${isDemoMigration ? 'bg-amber text-panel shadow-sm shadow-amber/20' : 'text-dim hover:text-ink'}`}
           >
              Sync + Migration
           </button>
        </div>
      </header>

      <div className="p-6 md:p-10 max-w-7xl mx-auto space-y-6">
        
        {/* Latest Migration Job (Only show if migration mode is active) */}
        {showMigrationUI && (
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-amber/30 p-6 flex flex-col md:flex-row items-center justify-between mb-6 relative overflow-hidden ring-1 ring-amber/10">
             <div className="absolute top-0 right-0 w-64 h-64 bg-amber/5 rounded-full blur-3xl -mr-10 -mt-10 pointer-events-none"></div>
             <div className="flex-1 relative z-10">
                <div className="flex items-center gap-2 mb-2">
                   <Zap className="w-5 h-5 text-amber" />
                   <h3 className="text-lg font-semibold text-ink">Latest Migration Job</h3>
                </div>
                {data.migration_history && data.migration_history.length > 0 ? (
                    <p className="text-sm text-dim mb-4">
                      Migration job <span className="text-ink font-medium">{data.migration_history[0].name}</span> completed on <span className="text-ink">{data.migration_history[0].date}</span>.
                    </p>
                ) : (
                    <p className="text-sm text-dim mb-4">No recent migration history found.</p>
                )}
                
                {data.migration_history && data.migration_history.length > 0 && (
                <div className="flex flex-wrap gap-6 mt-4">
                   <div>
                      <p className="text-xs text-faint uppercase tracking-wider mb-1">Volume</p>
                      <p className="text-lg font-bold text-ink">{formatBytes(data.migration_history[0].volume_bytes)}</p>
                   </div>
                   <div>
                      <p className="text-xs text-faint uppercase tracking-wider mb-1">Duration</p>
                      <p className="text-lg font-bold text-ink">{data.migration_history[0].duration || 'N/A'}</p>
                   </div>
                   <div>
                      <p className="text-xs text-faint uppercase tracking-wider mb-1">Status</p>
                      <p className={`text-lg font-bold ${data.migration_history[0].status === 'Success' ? 'text-ok' : 'text-amber'}`}>{data.migration_history[0].status}</p>
                   </div>
                </div>
                )}
             </div>
             
             {/* Progress indicator can still be shown if not 100%, otherwise show checkmark */}
             <div className="mt-6 md:mt-0 md:ml-10 text-center relative z-10 border-t md:border-t-0 md:border-l border-linesoft pt-4 md:pt-0 md:pl-8 min-w-[150px]">
                {data.overall_progress_percent < 100 && data.total_pending_files > 0 ? (
                  <>
                    <p className="text-xs text-faint uppercase tracking-wider mb-1">Overall Progress</p>
                    <span className="text-4xl font-bold text-amber">{data.overall_progress_percent.toFixed(1)}%</span>
                    <p className="text-xs text-dim mt-2">{data.total_pending_files.toLocaleString()} files pending</p>
                  </>
                ) : (
                  <div className="flex flex-col items-center justify-center h-full">
                    <div className="w-12 h-12 rounded-full bg-ok/10 flex items-center justify-center mb-2">
                      <FileCheck className="w-6 h-6 text-ok" />
                    </div>
                    <p className="text-sm font-medium text-ok">Migration Complete</p>
                  </div>
                )}
             </div>
          </div>
        )}

        {error && (
          <div className="bg-alert/10 border-l-4 border-alert p-4 rounded-r-md">
            <p className="text-sm text-alert"><strong>API Error:</strong> {error}</p>
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard 
            title="Total Storage Used" 
            value={formatBytes(data.total_data_migrated_bytes)} 
            icon={<HardDrive className="w-6 h-6 text-cy" />} 
          />
          <MetricCard 
            title="Total Files" 
            value={(data.total_files_migrated + data.total_pending_files).toLocaleString()} 
            icon={<FileCheck className="w-6 h-6 text-ok" />} 
          />
          <MetricCard 
            title="Stale Files (>1 Year)" 
            value={data.stale_files_over_1_year.toLocaleString()} 
            icon={<Clock className="w-6 h-6 text-amber" />} 
          />
          <MetricCard 
            title="VM Total Capacity" 
            value={formatBytes(data.vm_total_bytes)} 
            subtitle={data.vm_used_percent > 0 ? (data.vm_used_percent.toFixed(1) + "% Used") : "Loading..."}
            icon={<Server className="w-6 h-6 text-amberdeep" />} 
          />
        </div>
        
        {/* System Settings & Configuration */}
        <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 mt-6">
          <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-4">
            <Settings className="w-5 h-5 text-faint" />
            Storage Configuration
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-panel2 p-4 rounded-lg">
              <p className="text-xs text-dim uppercase tracking-wider mb-1">Root Path</p>
              <p className="text-sm font-mono text-ink break-all">{data.root_path}</p>
            </div>
            <div className="bg-panel2 p-4 rounded-lg">
              <p className="text-xs text-dim uppercase tracking-wider mb-1">Legacy Path (Source)</p>
              <p className="text-sm font-mono text-ink break-all">{data.legacy_path}</p>
            </div>
            <div className="bg-panel2 p-4 rounded-lg">
              <p className="text-xs text-dim uppercase tracking-wider mb-1">Sharding Type</p>
              <p className="text-sm font-medium text-ink capitalize">{data.sharding_type || 'None'}</p>
            </div>
          </div>
        </div>
        
        {/* DYNAMIC BLOCK: Distribution Mode vs Migration Mode */}
        {showMigrationUI && (
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-amber/30 p-6 flex flex-col md:flex-row items-center justify-between relative overflow-hidden">
             <div className="absolute top-0 right-0 w-64 h-64 bg-amber/5 rounded-full blur-3xl -mr-10 -mt-10 pointer-events-none"></div>
             
             <div className="flex-1 relative z-10">
                <div className="flex items-center gap-2 mb-2">
                   <Zap className="w-5 h-5 text-amber" />
                   <h3 className="text-lg font-semibold text-ink">Active Migration Job</h3>
                </div>
                <p className="text-sm text-dim mb-4">Heavy-lifting migration from legacy storage is currently running. ETA: ~2h 15m.</p>
                <div className="w-full bg-panel2 rounded-full h-4 mb-2 overflow-hidden border border-line relative">
                  <div className="bg-amber h-4 rounded-full transition-all duration-1000 relative" style={{ width: (data.overall_progress_percent) + '%' }}></div>
                </div>
                <div className="flex justify-between text-xs text-faint font-medium">
                  <span>{data.total_files_migrated.toLocaleString()} Migrated</span>
                  <span>{data.total_pending_files.toLocaleString()} Pending</span>
                </div>
             </div>
             <div className="mt-6 md:mt-0 md:ml-10 text-center relative z-10 border-t md:border-t-0 md:border-l border-linesoft pt-4 md:pt-0 md:pl-8">
                <p className="text-xs text-faint uppercase tracking-wider mb-1">Overall Progress</p>
                <span className="text-4xl font-bold text-amber">{data.overall_progress_percent.toFixed(1)}%</span>
             </div>
          </div>
        )}

        {/* Active Distribution block is ALWAYS shown */}
        <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-cy/20 p-6 flex flex-col md:flex-row items-center justify-between relative overflow-hidden">
           <div className="absolute top-0 left-0 w-64 h-64 bg-cy/5 rounded-full blur-3xl -ml-10 -mt-10 pointer-events-none"></div>
           
           <div className="flex-1 relative z-10">
              <div className="flex items-center gap-3 mb-2">
                 <div className="relative flex h-3 w-3">
                   <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-cy opacity-75"></span>
                   <span className="relative inline-flex rounded-full h-3 w-3 bg-cy"></span>
                 </div>
                 <h3 className="text-lg font-semibold text-ink">Active Distribution & Sync</h3>
              </div>
              <p className="text-sm text-dim mb-4">OmniBlob is actively monitoring and distributing delta changes to all connected endpoints.</p>
              <div className="flex flex-wrap gap-6 text-sm">
                 <div className="flex items-center gap-2">
                    <Network className="w-5 h-5 text-cy" />
                    <span className="text-ink font-medium">{data.connected_endpoints}</span>
                    <span className="text-faint">Connected Targets</span>
                 </div>
                 <div className="flex items-center gap-2">
                    <Activity className="w-5 h-5 text-amber" />
                    <span className="text-ink font-medium">{data.sync_rate_mbps} MB/s</span>
                    <span className="text-faint">Live Sync Rate</span>
                 </div>
                 <div className="flex items-center gap-2">
                    <RefreshCw className="w-5 h-5 text-ok" />
                    <span className="text-ok font-medium">{data.delta_changes_today} Delta Detected</span>
                 </div>
              </div>
           </div>
           <div className="mt-6 md:mt-0 md:ml-10 flex gap-8 text-center border-t md:border-t-0 md:border-l border-linesoft pt-4 md:pt-0 md:pl-8 relative z-10">
              <div>
                 <p className="text-xs text-faint uppercase tracking-wider mb-1">Last Synced</p>
                 <p className="text-lg font-bold text-ink">{data.last_synced_time}</p>
              </div>
              <div>
                 <p className="text-xs text-faint uppercase tracking-wider mb-1">Total Distributed</p>
                 <p className="text-2xl font-bold text-ok">{formatBytes(data.total_data_migrated_bytes)}</p>
              </div>
           </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
          
          {/* Module / App Breakdowns */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6">
            <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-6">
              <PieChartIcon className="w-5 h-5 text-faint" />
              Storage by Source Application
            </h3>
            <div className="h-64 w-full flex">
              {analytics && analytics.module_stats && analytics.module_stats.length > 0 ? (
                <>
                  <ResponsiveContainer width="60%" height="100%">
                    <PieChart>
                      <Pie
                        data={analytics.module_stats}
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={5}
                        dataKey="size_bytes"
                        nameKey="module"
                      >
                        {analytics.module_stats.map((entry, index) => (
                          <Cell key={index} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip formatter={(val: number) => formatBytes(val)} contentStyle={{ borderRadius: '8px', border: 'none', background: '#1f2d3b', color: '#fff' }} />
                    </PieChart>
                  </ResponsiveContainer>
                  <div className="w-[40%] flex flex-col justify-center gap-3 overflow-auto">
                    {analytics.module_stats.map((entry, index) => (
                       <div key={index} className="flex items-center text-sm">
                         <div className="w-3 h-3 rounded-full mr-2" style={{ backgroundColor: COLORS[index % COLORS.length] }}></div>
                         <span className="text-ink truncate flex-1">{entry.module}</span>
                         <span className="text-dim font-medium ml-2">{formatBytes(entry.size_bytes)}</span>
                       </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="w-full h-full flex items-center justify-center text-faint">No source data available</div>
              )}
            </div>
          </div>

          {/* Extension Breakdowns */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6">
            <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-6">
              <BarChart2 className="w-5 h-5 text-faint" />
              File Distribution by Type
            </h3>
            <div className="h-64 w-full">
              {analytics && analytics.extension_stats && analytics.extension_stats.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={analytics.extension_stats.slice(0, 7)} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#1f2d3b" vertical={false} />
                    <XAxis dataKey="extension" stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} />
                    <YAxis stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => formatBytes(val)} />
                    <Tooltip 
                      contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                      formatter={(value: number) => [formatBytes(value), 'Size']}
                      labelFormatter={(label) => `.${label}`}
                    />
                    <Bar dataKey="size_bytes" radius={[4, 4, 0, 0]}>
                      {analytics.extension_stats.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="w-full h-full flex items-center justify-center text-faint">No extension data available</div>
              )}
            </div>
          </div>

          {/* Storage Size by Year */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6">
            <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-6">
              <Activity className="w-5 h-5 text-faint" />
              Storage Growth by Year
            </h3>
            <div className="h-64 w-full">
              {analytics && analytics.yearly_stats && analytics.yearly_stats.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={analytics.yearly_stats} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#1f2d3b" vertical={false} />
                    <XAxis dataKey="year" stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} />
                    <YAxis stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => formatBytes(val)} />
                    <Tooltip 
                      contentStyle={{ borderRadius: '8px', border: 'none', background: '#1f2d3b', color: '#fff', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                      formatter={(value: number) => [formatBytes(value), 'Total Size']}
                      labelFormatter={(label) => `Year: ${label}`}
                    />
                    <Bar dataKey="size_bytes" radius={[4, 4, 0, 0]} fill="#10b981" />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="w-full h-full flex items-center justify-center text-faint">No yearly data available</div>
              )}
            </div>
          </div>

          {/* Storage Size by Month */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-ink flex items-center gap-2">
                <Activity className="w-5 h-5 text-faint" />
                Monthly Growth
              </h3>
              {analytics && analytics.yearly_stats && analytics.yearly_stats.length > 0 && (
                <div className="relative flex items-center group">
                  <select 
                    className="appearance-none bg-panel2 text-ink text-sm font-medium rounded-lg border border-line hover:border-cy/50 focus:border-cy pl-3 pr-8 py-1 outline-none cursor-pointer transition-all shadow-sm"
                    value={selectedYear}
                    onChange={(e) => setSelectedYear(Number(e.target.value))}
                  >
                    {analytics.yearly_stats.map(y => (
                      <option key={y.year} value={y.year} className="bg-panel text-ink py-1">{y.year}</option>
                    ))}
                  </select>
                  <ChevronDown className="absolute right-2 w-4 h-4 text-dim pointer-events-none group-hover:text-cy transition-colors" />
                </div>
              )}
            </div>
            <div className="h-64 w-full">
              {analytics && analytics.monthly_stats && analytics.monthly_stats.filter(m => m.year === selectedYear).length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={analytics.monthly_stats.filter(m => m.year === selectedYear).map(m => ({
                    ...m,
                    monthName: new Date(2000, m.month - 1, 1).toLocaleString('default', { month: 'short' })
                  }))} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#1f2d3b" vertical={false} />
                    <XAxis dataKey="monthName" stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} />
                    <YAxis stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => formatBytes(val)} />
                    <Tooltip 
                      contentStyle={{ borderRadius: '8px', border: 'none', background: '#1f2d3b', color: '#fff', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                      formatter={(value: number) => [formatBytes(value), 'Total Size']}
                      labelFormatter={(label) => `${label} ${selectedYear}`}
                    />
                    <Bar dataKey="size_bytes" radius={[4, 4, 0, 0]} fill="#3b82f6" />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="w-full h-full flex items-center justify-center text-faint">No monthly data available for {selectedYear}</div>
              )}
            </div>
          </div>
        </div>

        {/* Data Quality Row */}
        {quality && quality.total_duplicate_files > 0 && (
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 mt-6">
            <div className="flex flex-col md:flex-row md:items-center justify-between mb-6 border-b border-linesoft pb-4">
              <div>
                <h3 className="text-lg font-semibold text-ink flex items-center gap-2">
                  <AlertTriangle className="w-5 h-5 text-alert" />
                  Data Quality & Duplicates
                </h3>
                <p className="text-sm text-dim mt-1">We found duplicate files that waste storage space.</p>
              </div>
              <div className="mt-4 md:mt-0 flex items-center gap-6">
                <div className="text-center">
                  <p className="text-xs text-faint font-medium uppercase tracking-wider">Duplicate Files</p>
                  <p className="text-xl font-bold text-ink">{quality.total_duplicate_files.toLocaleString()}</p>
                </div>
                <div className="text-center">
                  <p className="text-xs text-faint font-medium uppercase tracking-wider">Potential Savings</p>
                  <p className="text-xl font-bold text-ok">{formatBytes(quality.total_wasted_bytes)}</p>
                </div>
              </div>
            </div>

            <div className="overflow-auto max-h-80">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-dim uppercase bg-panel2 sticky top-0">
                  <tr>
                    <th className="px-4 py-3 font-medium rounded-tl-lg">Example File Name</th>
                    <th className="px-4 py-3 font-medium text-center">Copies Found</th>
                    <th className="px-4 py-3 font-medium text-right">File Size</th>
                    <th className="px-4 py-3 font-medium rounded-tr-lg text-right text-ok">Wasted Space</th>
                  </tr>
                </thead>
                <tbody>
                  {quality.duplicate_groups.map((group, idx) => (
                    <tr key={idx} className="border-b border-linesoft last:border-0 hover:bg-panel2">
                      <td className="px-4 py-3 font-medium text-ink truncate max-w-[300px]" title={group.example_name}>{group.example_name}</td>
                      <td className="px-4 py-3 text-center">
                        <span className="px-2 py-1 bg-amber/10 text-amberdeep font-semibold rounded-md text-xs">{group.count}x</span>
                      </td>
                      <td className="px-4 py-3 text-right text-dim">{formatBytes(group.size_bytes)}</td>
                      <td className="px-4 py-3 text-right font-medium text-ok">{formatBytes(group.wasted_bytes)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Migration & Job History */}
        <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 mt-6">
          <div className="flex flex-col md:flex-row items-center justify-between mb-6 border-b border-linesoft pb-4">
             <div className="flex items-center gap-3">
                <div className="p-2 bg-panel2 rounded-lg">
                   <History className="w-5 h-5 text-cy" />
                </div>
                <div>
                   <h3 className="text-lg font-semibold text-ink">Migration & Job History</h3>
                   <p className="text-sm text-dim">Audit log of past migrations and synchronization jobs.</p>
                </div>
             </div>
             <div className="mt-4 md:mt-0 text-right">
                <p className="text-xs text-faint uppercase tracking-wider mb-1">Total Lifetime Migrations</p>
                <p className="text-2xl font-bold text-ink">{data.total_lifetime_migrations} <span className="text-sm font-normal text-dim">Jobs</span></p>
             </div>
          </div>
          
          <div className="overflow-x-auto">
             <table className="w-full text-sm text-left">
                <thead className="text-xs text-dim uppercase bg-panel2 sticky top-0">
                   <tr>
                      <th className="px-4 py-3 font-medium rounded-tl-lg">Job Name</th>
                      <th className="px-4 py-3 font-medium">Type</th>
                      <th className="px-4 py-3 font-medium">Date (Finished)</th>
                      <th className="px-4 py-3 font-medium text-right">Volume</th>
                      <th className="px-4 py-3 font-medium rounded-tr-lg text-right">Status</th>
                   </tr>
                </thead>
                <tbody>
                   {data.migration_history && data.migration_history.map((job) => (
                      <tr key={job.id} className="border-b border-linesoft last:border-0 hover:bg-panel2 transition-colors">
                         <td className="px-4 py-3 font-medium text-ink flex items-center gap-2">
                            <Archive className="w-4 h-4 text-faint" />
                            {job.name}
                         </td>
                         <td className="px-4 py-3 text-dim">{job.type}</td>
                         <td className="px-4 py-3 text-dim">{job.date}</td>
                         <td className="px-4 py-3 text-right text-ink font-mono">{formatBytes(job.volume_bytes)}</td>
                         <td className="px-4 py-3 text-right">
                            <span className={`px-2 py-1 text-xs font-semibold rounded-md ${
                               job.status === 'Success' ? 'bg-ok/10 text-ok' : 
                               job.status === 'Dry-Run' ? 'bg-faint/10 text-dim' : 
                               'bg-amber/10 text-amber'
                            }`}>
                               {job.status}
                            </span>
                         </td>
                      </tr>
                   ))}
                </tbody>
             </table>
          </div>
        </div>

      </div>
    </div>
  );
}

function MetricCard({ title, value, subtitle, icon }: { title: string, value: string, subtitle?: string, icon: React.ReactNode }) {
  return (
    <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 flex items-start justify-between">
      <div>
        <p className="text-sm font-medium text-dim mb-1">{title}</p>
        <h4 className="text-2xl font-bold text-ink">{value}</h4>
        {subtitle && <p className="text-xs text-faint mt-1">{subtitle}</p>}
      </div>
      <div className="p-3 bg-panel2 rounded-lg">
        {icon}
      </div>
    </div>
  );
}
