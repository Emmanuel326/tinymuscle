import { useState, useEffect } from 'react';
import { Toaster, toast } from 'react-hot-toast';
import { TenderCard } from './components/TenderCard';
import { PortalManager } from './components/PortalManager';
import { useSSE } from './hooks/useSSE';
import { api } from './services/api';
import { Activity, Zap, Database, Target, AlertCircle, Loader2 } from 'lucide-react';

function App() {
  const [tenders, setTenders] = useState([]);
  const [filteredTenders, setFilteredTenders] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [stats, setStats] = useState({ total: 0, new: 0, updated: 0 });
  const [showLiveOnly, setShowLiveOnly] = useState(false);
  const [loading, setLoading] = useState(true);
  
  const { events, isConnected, error: sseError } = useSSE('http://localhost:8080/events');
  
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const [tendersData, statsData] = await Promise.all([
          api.getTenders(),
          api.getTenderStats()
        ]);
        setTenders(tendersData || []);
        setStats(statsData || { total: 0, new: 0, updated: 0 });
      } catch (error) {
        console.error('Error fetching data:', error);
        toast.error('Failed to load tenders');
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);
  
  useEffect(() => {
    if (events && events.length > 0) {
      events.forEach(event => {
        if (event && (event.type === 'new' || event.type === 'updated') && event.tender) {
          setTenders(prev => {
            const existingIndex = prev.findIndex(t => t?.reference_number === event.tender.reference_number);
            if (existingIndex >= 0) {
              const updated = [...prev];
              updated[existingIndex] = event.tender;
              return updated;
            } else {
              return [event.tender, ...prev];
            }
          });
          
          toast.success(`${event.type === 'new' ? '🆕 New' : '📝 Updated'} tender: ${event.tender.title}`);
          
          setStats(prev => ({
            ...prev,
            total: prev.total + (event.type === 'new' ? 1 : 0),
            [event.type]: prev[event.type] + 1
          }));
        }
      });
    }
  }, [events]);
  
  useEffect(() => {
    let filtered = tenders || [];
    
    if (statusFilter !== 'all') {
      filtered = filtered.filter(t => t?.status === statusFilter);
    }
    
    if (searchTerm) {
      filtered = filtered.filter(t => 
        t?.title?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        t?.reference_number?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        t?.issuing_entity?.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }
    
    if (showLiveOnly) {
      filtered = filtered.slice(0, 20);
    }
    
    setFilteredTenders(filtered);
  }, [tenders, statusFilter, searchTerm, showLiveOnly]);
  
  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="w-12 h-12 text-blue-600 animate-spin mx-auto mb-4" />
          <p className="text-gray-600">Loading TinyMuscle...</p>
        </div>
      </div>
    );
  }
  
  return (
    <div className="min-h-screen bg-gray-50">
      <Toaster position="top-right" />
      
      <header className="bg-gradient-to-r from-blue-600 to-purple-700 text-white shadow-lg">
        <div className="container mx-auto px-4 py-6">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold flex items-center gap-2">
                <Zap className="w-8 h-8" />
                TinyMuscle
              </h1>
              <p className="text-blue-100 mt-1">
                Stateful web intelligence that never misses an opportunity
              </p>
            </div>
            <div className="flex items-center gap-3">
              <div className={`flex items-center gap-2 px-3 py-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}>
                <Activity className="w-4 h-4" />
                <span className="text-sm font-medium">
                  {isConnected ? 'Live Feed' : 'Disconnected'}
                </span>
              </div>
              {sseError && <span className="text-sm text-red-200">{sseError}</span>}
            </div>
          </div>
        </div>
      </header>
      
      <main className="container mx-auto px-4 py-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">Total Tenders</p>
                <p className="text-2xl font-bold text-gray-900">{stats.total || 0}</p>
              </div>
              <Database className="w-8 h-8 text-blue-500" />
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">New Opportunities</p>
                <p className="text-2xl font-bold text-green-600">{stats.new || 0}</p>
              </div>
              <Zap className="w-8 h-8 text-green-500" />
            </div>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-gray-500 text-sm">Updated Tenders</p>
                <p className="text-2xl font-bold text-yellow-600">{stats.updated || 0}</p>
              </div>
              <Target className="w-8 h-8 text-yellow-500" />
            </div>
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex flex-wrap gap-4 items-center justify-between">
            <div className="flex gap-2">
              <button
                onClick={() => setStatusFilter('all')}
                className={`px-4 py-2 rounded-md transition-colors ${statusFilter === 'all' ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'}`}
              >
                All
              </button>
              <button
                onClick={() => setStatusFilter('new')}
                className={`px-4 py-2 rounded-md transition-colors ${statusFilter === 'new' ? 'bg-green-600 text-white' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'}`}
              >
                New
              </button>
              <button
                onClick={() => setStatusFilter('updated')}
                className={`px-4 py-2 rounded-md transition-colors ${statusFilter === 'updated' ? 'bg-yellow-600 text-white' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'}`}
              >
                Updated
              </button>
            </div>
            
            <div className="flex gap-2">
              <input
                type="text"
                placeholder="Search tenders..."
                className="px-4 py-2 border rounded-md w-64 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
              <button
                onClick={() => setShowLiveOnly(!showLiveOnly)}
                className={`px-4 py-2 rounded-md transition-colors ${showLiveOnly ? 'bg-purple-600 text-white' : 'bg-gray-200 text-gray-700 hover:bg-gray-300'}`}
              >
                {showLiveOnly ? '🔴 Live Mode' : '📊 All Tenders'}
              </button>
            </div>
          </div>
        </div>
        
        <PortalManager onPortalChange={() => {}} />
        
        <div className="mt-8">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            {showLiveOnly ? '🔴 Live Tender Feed' : '📑 All Tenders'}
            <span className="text-sm font-normal text-gray-500 ml-2">
              ({filteredTenders.length} of {tenders.length})
            </span>
          </h2>
          
          {filteredTenders.length === 0 ? (
            <div className="text-center py-12 bg-white rounded-lg shadow">
              <AlertCircle className="w-12 h-12 text-gray-400 mx-auto mb-3" />
              <p className="text-gray-500">No tenders found</p>
              <p className="text-sm text-gray-400 mt-1">
                Add portals to start watching for opportunities
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {filteredTenders.map((tender, index) => (
                <TenderCard key={`${tender?.reference_number || index}-${index}`} tender={tender} />
              ))}
            </div>
          )}
        </div>
      </main>
      
      <footer className="bg-gray-800 text-white mt-12 py-6">
        <div className="container mx-auto px-4 text-center">
          <p>Powered by TinyFish — Intelligent web intelligence for Africa</p>
          <p className="text-sm text-gray-400 mt-2">TinyMuscle • Built for the TinyFish Hackathon 2026</p>
        </div>
      </footer>
    </div>
  );
}

export default App;