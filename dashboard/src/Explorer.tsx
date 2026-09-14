import React, { useState, useEffect } from 'react';
import { Folder, FolderOpen, File, ChevronRight, ChevronDown, RefreshCw, AlertTriangle } from 'lucide-react';

interface FileNode {
  name: string;
  path: string;
  is_directory: boolean;
  size: number;
  modified_time: string;
}

interface DashboardSummary {
  migration_enabled: boolean;
  legacy_path: string;
  clients: string[];
}

export function Explorer() {
  const [activeRoot, setActiveRoot] = useState<'legacy' | 'omniblob' | null>(null);
  const [currentPath, setCurrentPath] = useState('/');
  
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  
  const [files, setFiles] = useState<FileNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Stores fetched children for any path: key is `${target}:${path}`
  const [treeData, setTreeData] = useState<Record<string, FileNode[]>>({});
  const [expandedFolders, setExpandedFolders] = useState<Record<string, boolean>>({});

  useEffect(() => {
    fetch('/api/v1/dashboard/summary')
      .then(res => res.json())
      .then(data => {
        setSummary(data);
        // Default to omniblob if legacy is disabled
        if (!activeRoot) {
           setActiveRoot(data.migration_enabled ? 'legacy' : 'omniblob');
        }
      })
      .catch(console.error);
  }, []);

  const fetchFiles = async (target: string, path: string) => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`/api/v1/explorer/list?target=${target}&path=${encodeURIComponent(path)}`);
      if (!res.ok) throw new Error('Failed to load directory');
      const data = await res.json();
      setFiles(data || []);
    } catch (err: any) {
      setError(err.message);
      setFiles([]);
    } finally {
      setLoading(false);
    }
  };

  // Fetch files for right panel when active path changes
  useEffect(() => {
    if (activeRoot && currentPath) {
      // Special case: if we are at root of omniblob and want to just show clients
      // wait, the API will list actual folders. We will just list actual folders.
      fetchFiles(activeRoot, currentPath);
    }
  }, [activeRoot, currentPath]);

  // Fetch children for tree view when a folder is expanded
  const toggleFolder = async (target: 'legacy' | 'omniblob', path: string) => {
    const key = `${target}:${path}`;
    const isExpanded = expandedFolders[key];
    
    // Toggle state
    setExpandedFolders(prev => ({ ...prev, [key]: !isExpanded }));

    // If we are expanding and don't have data yet, fetch it
    if (!isExpanded && !treeData[key]) {
      try {
        const res = await fetch(`/api/v1/explorer/list?target=${target}&path=${encodeURIComponent(path)}`);
        if (res.ok) {
          const data: FileNode[] = await res.json();
          const dirsOnly = (data || []).filter(f => f.is_directory);
          setTreeData(prev => ({ ...prev, [key]: dirsOnly }));
        }
      } catch (err) {
        console.error("Failed to load tree nodes", err);
      }
    }
  };

  const selectFolder = (target: 'legacy' | 'omniblob', path: string) => {
    setActiveRoot(target);
    setCurrentPath(path);
  };

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const handleRowClick = (node: FileNode) => {
    if (node.is_directory) {
      setCurrentPath(node.path);
    }
  };

  const getLegacyFolderName = () => {
    if (!summary?.legacy_path) return 'Legacy storage';
    const parts = summary.legacy_path.replace(/\\/g, '/').split('/').filter(Boolean);
    return parts.length > 0 ? parts[parts.length - 1] : 'Legacy storage';
  };

  // Recursive Tree Component
  const TreeNode = ({ target, node, level }: { target: 'legacy'|'omniblob', node: FileNode, level: number }) => {
    const key = `${target}:${node.path}`;
    const isExpanded = expandedFolders[key];
    const isActive = activeRoot === target && currentPath === node.path;
    const children = treeData[key] || [];

    return (
      <div className="w-full">
        <div 
          className={`flex items-center p-1.5 rounded cursor-pointer text-sm ${isActive ? 'bg-panel2 text-ink font-medium' : 'text-dim hover:text-ink hover:bg-panel2/50'}`}
          style={{ paddingLeft: `${level * 12 + 8}px` }}
          onClick={() => selectFolder(target, node.path)}
        >
          <div onClick={(e) => { e.stopPropagation(); toggleFolder(target, node.path); }} className="w-5 flex items-center justify-center text-faint hover:text-ink">
            {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </div>
          <Folder className={`w-4 h-4 mr-2 shrink-0 ${isActive ? 'text-cy' : 'text-faint'}`} />
          <span className="flex-1 truncate">{node.name}</span>
          {node.size > 0 && <span className="text-[10px] text-faint ml-2">{formatBytes(node.size)}</span>}
        </div>
        
        {isExpanded && (
          <div className="w-full">
            {children.length === 0 ? (
              <div className="text-xs text-faint italic py-1" style={{ paddingLeft: `${(level+1) * 12 + 28}px` }}>empty</div>
            ) : (
              children.map((child, idx) => (
                <TreeNode key={idx} target={target} node={child} level={level + 1} />
              ))
            )}
          </div>
        )}
      </div>
    );
  };

  if (!summary) return null;

  const breadcrumbs = currentPath.split('/').filter(Boolean);

  return (
    <div className="flex-1 w-full flex flex-col h-full bg-bg font-sans">
      <header className="bg-panel border-b border-line px-6 py-4 flex items-center justify-between sticky top-0 z-10 shrink-0">
        <h2 className="text-lg font-semibold text-ink">Storage Explorer</h2>
      </header>
      
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar - Tree View */}
        <div className="w-72 bg-panel border-r border-line p-2 overflow-y-auto shrink-0 select-none">
          
          {/* Legacy Root (Only if Migration is Enabled) */}
          {summary.migration_enabled && (
            <div className="mb-4">
              <div 
                className={`flex items-center p-2 rounded cursor-pointer ${activeRoot === 'legacy' && currentPath === '/' ? 'bg-panel2' : 'hover:bg-panel2/50'}`}
                onClick={() => selectFolder('legacy', '/')}
              >
                <div onClick={(e) => { e.stopPropagation(); toggleFolder('legacy', '/'); }} className="mr-1 text-dim hover:text-ink">
                  {expandedFolders['legacy:/'] ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                </div>
                <Folder className="w-4 h-4 mr-2 text-faint" />
                <span className="text-sm font-medium text-ink flex-1 truncate pr-2" title={summary.legacy_path}>{getLegacyFolderName()}</span>
                <span className="text-[10px] px-2 py-0.5 rounded-full bg-amber/10 text-amber font-semibold">legacy</span>
              </div>
              
              {expandedFolders['legacy:/'] && (
                <div className="w-full">
                  {(treeData['legacy:/'] || []).map((child, idx) => (
                    <TreeNode key={idx} target="legacy" node={child} level={1} />
                  ))}
                  {treeData['legacy:/']?.length === 0 && (
                    <div className="text-xs text-faint italic py-1 pl-10">no folders found</div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* OmniBlob Root */}
          <div>
            <div 
              className={`flex items-center p-2 rounded cursor-pointer ${activeRoot === 'omniblob' && currentPath === '/' ? 'bg-panel2' : 'hover:bg-panel2/50'}`}
              onClick={() => selectFolder('omniblob', '/')}
            >
              <div onClick={(e) => { e.stopPropagation(); toggleFolder('omniblob', '/'); }} className="mr-1 text-dim hover:text-ink">
                {expandedFolders['omniblob:/'] ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
              </div>
              <Folder className="w-4 h-4 mr-2 text-faint" />
              <span className="text-sm font-medium text-ink flex-1">omniBlob storage</span>
              <span className="text-[10px] px-2 py-0.5 rounded-full bg-cy/10 text-cy font-semibold">omniBlob</span>
            </div>
            
            {expandedFolders['omniblob:/'] && (
              <div className="w-full">
                {/* Dynamically list actual folders. We will visually highlight client folders if they match */}
                {(treeData['omniblob:/'] || []).map((child, idx) => (
                  <TreeNode key={idx} target="omniblob" node={child} level={1} />
                ))}
                
                {/* Fallback to show registered clients even if their folders aren't created yet */}
                {summary.clients.filter(clientName => !(treeData['omniblob:/'] || []).find(d => d.name === clientName)).map((clientName, idx) => (
                  <div key={`client-${idx}`} className="w-full">
                    <div 
                      className={`flex items-center p-1.5 rounded cursor-pointer text-sm text-dim hover:text-ink hover:bg-panel2/50 opacity-60`}
                      style={{ paddingLeft: `20px` }}
                      onClick={() => selectFolder('omniblob', '/' + clientName)}
                    >
                      <div className="w-5"></div>
                      <Folder className="w-4 h-4 mr-2 text-faint" />
                      <span className="flex-1 truncate">{clientName}</span>
                    </div>
                  </div>
                ))}

              </div>
            )}
          </div>

        </div>

        {/* Right Content - File List */}
        <div className="flex-1 flex flex-col bg-bg overflow-hidden relative">
          
          {/* Breadcrumbs */}
          <div className="px-6 py-4 border-b border-linesoft bg-bg flex items-center gap-2 text-sm shrink-0">
            <span className="text-ink font-medium">
              {activeRoot === 'legacy' ? getLegacyFolderName() : 'omniBlob storage'}
            </span>
            {breadcrumbs.map((crumb, idx) => (
              <React.Fragment key={idx}>
                <span className="text-faint">/</span>
                <span 
                  className="text-ink cursor-pointer hover:underline"
                  onClick={() => setCurrentPath('/' + breadcrumbs.slice(0, idx+1).join('/'))}
                >{crumb}</span>
              </React.Fragment>
            ))}
          </div>

          {/* File List */}
          <div className="flex-1 overflow-y-auto p-6">
            {error ? (
              <div className="flex flex-col items-center justify-center h-full text-dim">
                <AlertTriangle className="w-12 h-12 text-alert mb-4 opacity-50" />
                <p>{error}</p>
                <p className="text-xs mt-2 text-faint">Ensure path is configured in config.yaml</p>
              </div>
            ) : loading ? (
              <div className="flex items-center justify-center h-full">
                <RefreshCw className="w-8 h-8 text-dim animate-spin" />
              </div>
            ) : files.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-dim">
                <FolderOpen className="w-16 h-16 mb-4 opacity-20" />
                <p>Folder is empty</p>
              </div>
            ) : (
              <table className="w-full text-left text-sm relative">
                <thead className="text-xs text-dim uppercase sticky top-0 bg-bg z-10 shadow-[0_1px_0_0_var(--color-linesoft)]">
                  <tr>
                    <th className="py-3 font-medium bg-bg">Name</th>
                    <th className="py-3 font-medium text-right bg-bg">Size</th>
                    <th className="py-3 font-medium text-right bg-bg">Modified</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-linesoft">
                  {currentPath !== '/' && (
                    <tr 
                      className="hover:bg-panel2 cursor-pointer transition-colors"
                      onClick={() => {
                        const parts = currentPath.split('/').filter(Boolean);
                        parts.pop();
                        setCurrentPath('/' + parts.join('/'));
                      }}
                    >
                      <td className="py-3 flex items-center gap-3 text-ink">
                        <Folder className="w-5 h-5 text-faint" />
                        ..
                      </td>
                      <td className="py-3 text-right text-dim">-</td>
                      <td className="py-3 text-right text-dim">-</td>
                    </tr>
                  )}
                  {files.map((f, i) => (
                    <tr 
                      key={i} 
                      className="hover:bg-panel2 cursor-pointer transition-colors"
                      onClick={() => handleRowClick(f)}
                    >
                      <td className="py-3 flex items-center gap-3 text-ink font-medium">
                        {f.is_directory ? (
                          <Folder className="w-5 h-5 text-cy" />
                        ) : (
                          <File className="w-5 h-5 text-dim" />
                        )}
                        {f.name}
                      </td>
                      <td className="py-3 text-right text-dim">{f.size > 0 ? formatBytes(f.size) : (f.is_directory ? '-' : '0 B')}</td>
                      <td className="py-3 text-right text-dim">{new Date(f.modified_time).toLocaleString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
