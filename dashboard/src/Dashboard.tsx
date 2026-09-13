import React, { useEffect, useState } from 'react';
import { Activity, HardDrive, FileCheck, Clock, Settings, RefreshCw, BarChart2, PieChart as PieChartIcon } from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, BarChart, Bar, Cell } from 'recharts';

interface DashboardSummary {
  total_data_migrated_bytes: number;
  total_files_migrated: number;
  total_pending_files: number;
  overall_progress_percent: number;
  live_transfer_rate_mbps: number;
  current_latency_ms: number;
  destination_free_space_bytes: number;
}

interface ExtensionStat {
  extension: string;
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

interface StorageAnalytics {
  extension_stats: ExtensionStat[];
  top_large_files: LargeFile[];
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

import { Logo } from './components/Logo';

export function Dashboard() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [analytics, setAnalytics] = useState<StorageAnalytics | null>(null);
  const [quality, setQuality] = useState<DataQualityStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Mock historical data for the chart since the backend doesn't provide it yet
  const [chartData, setChartData] = useState<{time: string, speed: number}[]>([
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
    const interval = setInterval(fetchData, 10000); // Polling every 10 seconds
    return () => clearInterval(interval);
  }, []);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
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

  // Use mock data if API fails in dev mode, so we can still see the UI
  const data = summary || {
    total_data_migrated_bytes: 125000000000,
    total_files_migrated: 15000,
    total_pending_files: 5000,
    overall_progress_percent: 75,
    live_transfer_rate_mbps: 150,
    current_latency_ms: 12,
    destination_free_space_bytes: 1500000000000,
  };

  return (
    <div className="min-h-screen bg-bg text-ink font-sans">
      {/* Top Navigation */}
      <header className="bg-panel border-b border-line px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Logo className="w-8 h-8" />
          <h1 className="text-xl font-bold text-ink">OmniBlob <span className="text-faint font-normal">| Admin Dashboard</span></h1>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-sm text-dim">
            <span className="relative flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
            </span>
            System Online
          </div>
          <button className="p-2 text-faint hover:text-dim rounded-full hover:bg-bg">
            <Settings className="w-5 h-5" />
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 py-8 space-y-6">
        
        {error && (
          <div className="bg-alert/10 border-l-4 border-alert p-4 rounded-r-md">
            <p className="text-sm text-alert"><strong>API Error:</strong> {error}. Using mock data for preview.</p>
          </div>
        )}

        {/* Hero Metrics Row */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard 
            title="Total Migrated Data" 
            value={formatBytes(data.total_data_migrated_bytes)} 
            icon={<HardDrive className="w-6 h-6 text-cy" />} 
          />
          <MetricCard 
            title="Files Processed" 
            value={data.total_files_migrated.toLocaleString()} 
            subtitle={`${data.total_pending_files.toLocaleString()} pending`}
            icon={<FileCheck className="w-6 h-6 text-ok" />} 
          />
          <MetricCard 
            title="Live Transfer Rate" 
            value={`${data.live_transfer_rate_mbps.toFixed(1)} MB/s`} 
            subtitle={`Latency: ${data.current_latency_ms}ms`}
            icon={<Activity className="w-6 h-6 text-amber" />} 
          />
          <MetricCard 
            title="Dest. Free Space" 
            value={formatBytes(data.destination_free_space_bytes)} 
            icon={<HardDrive className="w-6 h-6 text-amberdeep" />} 
          />
        </div>

        {/* Charts & Main Progress Row */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Progress Section */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 lg:col-span-1 flex flex-col items-center justify-center">
            <h3 className="text-lg font-semibold text-ink mb-8 self-start w-full">Overall Progress</h3>
            
            <div className="relative w-48 h-48">
              <svg className="w-full h-full transform -rotate-90">
                <circle cx="96" cy="96" r="80" className="stroke-line" strokeWidth="16" fill="none" />
                <circle 
                  cx="96" cy="96" r="80" 
                  className="stroke-cy transition-all duration-1000 ease-out" 
                  strokeWidth="16" fill="none" 
                  strokeDasharray="502" 
                  strokeDashoffset={502 - (502 * data.overall_progress_percent) / 100} 
                  strokeLinecap="round" 
                />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center">
                <span className="text-4xl font-bold text-ink">{data.overall_progress_percent}%</span>
                <span className="text-sm text-dim mt-1">Complete</span>
              </div>
            </div>

            <div className="w-full mt-8 flex justify-between text-sm text-dim border-t border-linesoft pt-4">
              <span>{data.total_files_migrated.toLocaleString()} done</span>
              <span>{(data.total_files_migrated + data.total_pending_files).toLocaleString()} total</span>
            </div>
          </div>

          {/* Chart Section */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 lg:col-span-2">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-lg font-semibold text-ink flex items-center gap-2">
                <BarChart2 className="w-5 h-5 text-faint" />
                Transfer Speed History
              </h3>
            </div>
            <div className="h-64 w-full">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData} margin={{ top: 5, right: 20, bottom: 5, left: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#1f2d3b" vertical={false} />
                  <XAxis dataKey="time" stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} />
                  <YAxis stroke="#64798c" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(val) => `${val} MB/s`} />
                  <Tooltip 
                    contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                    formatter={(value: number) => [`${value} MB/s`, 'Speed']}
                  />
                  <Line type="monotone" dataKey="speed" stroke="#5bc8dc" strokeWidth={3} dot={{ r: 4, strokeWidth: 2 }} activeDot={{ r: 6 }} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </div>
        </div>

        {/* Storage Analytics Row */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
          
          {/* File Types Distribution */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6">
            <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-6">
              <PieChartIcon className="w-5 h-5 text-faint" />
              File Distribution by Extension
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
                      formatter={(value: number, name: string, props: any) => [formatBytes(value), 'Size']}
                      labelFormatter={(label) => `.${label}`}
                    />
                    <Bar dataKey="size_bytes" radius={[4, 4, 0, 0]}>
                      {analytics.extension_stats.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={['#5bc8dc', '#10b981', '#f59e0b', '#8b5cf6', '#ec4899', '#14b8a6', '#f43f5e'][index % 7]} />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="w-full h-full flex items-center justify-center text-faint">
                  No extension data available
                </div>
              )}
            </div>
          </div>

          {/* Top Largest Files */}
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 flex flex-col h-full">
            <h3 className="text-lg font-semibold text-ink flex items-center gap-2 mb-4">
              <HardDrive className="w-5 h-5 text-faint" />
              Top 100 Largest Files
            </h3>
            
            <div className="overflow-auto flex-grow h-64">
              <table className="w-full text-sm text-left">
                <thead className="text-xs text-dim uppercase bg-panel2 sticky top-0">
                  <tr>
                    <th className="px-4 py-3 font-medium rounded-tl-lg">File Name</th>
                    <th className="px-4 py-3 font-medium">Extension</th>
                    <th className="px-4 py-3 font-medium rounded-tr-lg text-right">Size</th>
                  </tr>
                </thead>
                <tbody>
                  {analytics && analytics.top_large_files ? analytics.top_large_files.slice(0, 20).map((file) => (
                    <tr key={file.bin_id} className="border-b border-linesoft last:border-0 hover:bg-panel2">
                      <td className="px-4 py-3 font-medium text-ink truncate max-w-[200px]" title={file.file_name}>{file.file_name}</td>
                      <td className="px-4 py-3 text-dim">
                        <span className="px-2 py-1 bg-bg rounded-md text-xs">{file.extension || 'unknown'}</span>
                      </td>
                      <td className="px-4 py-3 text-right font-medium text-ink">{formatBytes(file.size_bytes)}</td>
                    </tr>
                  )) : (
                    <tr>
                      <td colSpan={3} className="px-4 py-8 text-center text-faint">No large files found</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>

        </div>

        {/* Data Quality Row */}
        {quality && quality.total_duplicate_files > 0 && (
          <div className="bg-panel rounded-xl shadow-md shadow-black/20 border border-line p-6 mt-6">
            <div className="flex flex-col md:flex-row md:items-center justify-between mb-6 border-b border-linesoft pb-4">
              <div>
                <h3 className="text-lg font-semibold text-ink flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-ok" />
                  Data Quality & Duplicates
                </h3>
                <p className="text-sm text-dim mt-1">We found duplicate files that can be safely removed to save space.</p>
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

      </main>
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
