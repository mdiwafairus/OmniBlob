import React from 'react';
import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { LayoutDashboard, FolderTree } from 'lucide-react';
import { Dashboard } from './Dashboard';
import { Explorer } from './Explorer';
import { Logo } from './components/Logo';

function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex bg-bg text-ink font-sans">
      {/* Sidebar */}
      <aside className="w-64 bg-panel border-r border-line flex flex-col">
        <div className="p-6 flex items-center gap-3 border-b border-line">
          <Logo className="w-8 h-8" />
          <h1 className="text-xl font-bold text-ink">OmniBlob</h1>
        </div>
        
        <nav className="flex-1 p-4 space-y-2">
          <NavLink
            to="/"
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                isActive 
                  ? 'bg-cy/10 text-cy font-medium' 
                  : 'text-dim hover:text-ink hover:bg-panel2'
              }`
            }
          >
            <LayoutDashboard className="w-5 h-5" />
            Dashboard
          </NavLink>
          
          <NavLink
            to="/explorer"
            className={({ isActive }) =>
              `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                isActive 
                  ? 'bg-cy/10 text-cy font-medium' 
                  : 'text-dim hover:text-ink hover:bg-panel2'
              }`
            }
          >
            <FolderTree className="w-5 h-5" />
            Storage Explorer
          </NavLink>
        </nav>
        
        <div className="p-4 border-t border-line">
          <div className="flex items-center gap-2 text-xs text-dim">
            <span className="relative flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
            </span>
            Node: Active
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col h-screen overflow-auto">
        {children}
      </main>
    </div>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/explorer" element={<Explorer />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}
