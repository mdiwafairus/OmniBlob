import React, { useEffect, useState } from 'react';
import { Activity, HardDrive, FileCheck, Clock, Settings, RefreshCw, BarChart2, PieChart as PieChartIcon, Server, AlertTriangle } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, Cell, PieChart, Pie } from 'recharts';
import { Logo } from './components/Logo';

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
  const [analytics, setAnalytics] = useState<StorageAnalytics | null>(null);
  const [quality, setQuality] = useState<DataQualityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedYear, setSelectedYear] = useState<number>(new Date().getFullYear());

  const [chartData] = useState<{time: string, speed: number}[]>([
    { time: '10:00', speed: 10 },
    { time: '10:05', speed: 45 },
    { time: '10:10', speed: 80 },
    { time: '10:15', speed: 150 },
    { time: '10:20', speed: 145 },
    { time: '10:25', speed: 155 },
  ]);

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
      setAnalytics(anaData);
      setQuality(qualData);
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

  const data = summary || {
    total_data_migrated_bytes: 0,
    total_files_migrated: 0,
    total_pending_files: 0,
    overall_progress_percent: 100,
    stale_files_over_1_year: 0,
    migration_enabled: false,
    vm_total_bytes: 0,
    vm_used_bytes: 0,
    vm_free_bytes: 0,
    vm_used_percent: 0,
    root_path: '...',
    legacy_path: '...',
    sharding_type: '...',
    clients: [],
  };

  const COLORS = ['#5bc8dc', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#14b8a6', '#f43f5e'];

  return (
    <div className="flex-1 w-full font-sans">
      <header className="bg-panel border-b border-line px-6 py-4 flex items-center justify-between sticky top-0 z-10">
        <h2 className="text-lg font-semibold text-ink">Overview</h2>
      </header>
      <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
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
        {/* Migration Progress (only if enabled) */}
        {data.migration_enabled && (
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 flex flex-col md:flex-row items-center justify-between">
             <div className="flex-1">
                <h3 className="text-lg font-semibold text-ink mb-2">Migration Progress</h3>
                <p className="text-sm text-dim mb-4">Background migration from legacy NAS is currently active.</p>
                <div className="w-full bg-panel2 rounded-full h-4 mb-2 overflow-hidden border border-line">
                  <div className="bg-cy h-4 rounded-full transition-all duration-1000" style={{ width: (data.overall_progress_percent) + '%' }}></div>
                </div>
                <div className="flex justify-between text-xs text-faint font-medium">
                  <span>{data.total_files_migrated.toLocaleString()} Migrated</span>
                  <span>{data.total_pending_files.toLocaleString()} Pending</span>
                </div>
             </div>
             <div className="mt-6 md:mt-0 md:ml-10 text-center">
                <span className="text-4xl font-bold text-ink">{data.overall_progress_percent.toFixed(1)}%</span>
             </div>
          </div>
        )}

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
                <select 
                  className="bg-panel2 text-ink text-sm rounded border border-line px-2 py-1 outline-none"
                  value={selectedYear}
                  onChange={(e) => setSelectedYear(Number(e.target.value))}
                >
                  {analytics.yearly_stats.map(y => (
                    <option key={y.year} value={y.year}>{y.year}</option>
                  ))}
                </select>
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
