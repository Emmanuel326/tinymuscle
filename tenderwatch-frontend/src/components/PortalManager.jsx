import { useState, useEffect } from 'react';
import { PlusIcon, TrashIcon, TargetIcon, ZapIcon } from 'lucide-react';
import { api } from '../services/api';
import toast from 'react-hot-toast';

export function PortalManager({ onPortalChange }) {
  const [portals, setPortals] = useState([]);
  const [showForm, setShowForm] = useState(false);
  const [loading, setLoading] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [formData, setFormData] = useState({
    id: '',
    name: '',
    url: '',
    goal: 'Navigate to the procurement notices page, extract all visible open tenders. For each tender extract: title, reference_number, issuing_entity, deadline, estimated_value, source_url. Return as JSON array.',
    interval_min: 60,
    business_profile: '',
    relevance_threshold: 60
  });
  
  const fetchPortals = async () => {
    try {
      const data = await api.getPortals();
      setPortals(data);
      if (onPortalChange) onPortalChange(data);
    } catch (error) {
      toast.error('Failed to fetch portals');
    }
  };
  
  useEffect(() => {
    fetchPortals();
  }, []);
  
  // Auto-fill ID from name
  const handleNameChange = (name) => {
    const id = name.toLowerCase().replace(/[^a-z0-9]/g, '_');
    setFormData({ ...formData, name, id });
  };
  
  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const payload = { ...formData };
      if (!payload.business_profile) {
        delete payload.business_profile;
        delete payload.relevance_threshold;
      }
      
      await api.createPortal(payload);
      toast.success(formData.business_profile ? 
        'Portal added with AI relevance filtering! 🎯' : 
        'Portal added successfully!');
      setShowForm(false);
      fetchPortals();
      setFormData({
        id: '',
        name: '',
        url: '',
        goal: 'Navigate to the procurement notices page, extract all visible open tenders. For each tender extract: title, reference_number, issuing_entity, deadline, estimated_value, source_url. Return as JSON array.',
        interval_min: 60,
        business_profile: '',
        relevance_threshold: 60
      });
    } catch (error) {
      toast.error('Failed to add portal');
    } finally {
      setLoading(false);
    }
  };
  
  const handleDelete = async (portalId) => {
    if (!confirm('Are you sure you want to delete this portal?')) return;
    try {
      await api.deletePortal(portalId);
      toast.success('Portal deleted');
      fetchPortals();
    } catch (error) {
      toast.error('Failed to delete portal');
    }
  };
  
  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <div className="flex justify-between items-center mb-4">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">📋 Active Portals</h2>
          <p className="text-sm text-gray-500 mt-1">
            Each portal runs on its own schedule with AI-powered relevance filtering
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="btn-primary flex items-center gap-2"
        >
          <PlusIcon className="w-4 h-4" />
          Add Portal
        </button>
      </div>
      
      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 p-4 bg-gray-50 rounded-lg border border-gray-200">
          <h3 className="font-semibold mb-3 flex items-center gap-2">
            <ZapIcon className="w-5 h-5 text-blue-600" />
            Configure New Portal
          </h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <input
              type="text"
              placeholder="Portal Name"
              className="px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500"
              value={formData.name}
              onChange={(e) => handleNameChange(e.target.value)}
              required
            />
            <input
              type="text"
              placeholder="Portal ID (auto-filled)"
              className="px-3 py-2 border rounded-md bg-gray-100"
              value={formData.id}
              readOnly
            />
            <input
              type="url"
              placeholder="URL"
              className="px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500"
              value={formData.url}
              onChange={(e) => setFormData({...formData, url: e.target.value})}
              required
            />
            <input
              type="number"
              placeholder="Crawl Interval (minutes)"
              className="px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500"
              value={formData.interval_min}
              onChange={(e) => setFormData({...formData, interval_min: parseInt(e.target.value)})}
              required
            />
            <textarea
              placeholder="Crawl Goal - Natural language instructions for TinyFish"
              className="px-3 py-2 border rounded-md md:col-span-2 focus:ring-2 focus:ring-blue-500"
              rows="3"
              value={formData.goal}
              onChange={(e) => setFormData({...formData, goal: e.target.value})}
              required
            />
          </div>
          
          <div className="mt-4">
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-2 text-sm text-blue-600 hover:text-blue-800"
            >
              <TargetIcon className="w-4 h-4" />
              {showAdvanced ? 'Hide' : 'Show'} AI Relevance Filtering
            </button>
            
            {showAdvanced && (
              <div className="mt-3 space-y-3 border-t border-gray-200 pt-3">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Business Profile (for AI relevance scoring)
                  </label>
                  <textarea
                    placeholder="Describe your business to get only relevant tenders..."
                    className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500"
                    rows="3"
                    value={formData.business_profile}
                    onChange={(e) => setFormData({...formData, business_profile: e.target.value})}
                  />
                </div>
                
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Relevance Threshold: {formData.relevance_threshold}%
                  </label>
                  <input
                    type="range"
                    min="0"
                    max="100"
                    step="5"
                    className="w-full"
                    value={formData.relevance_threshold}
                    onChange={(e) => setFormData({...formData, relevance_threshold: parseInt(e.target.value)})}
                    disabled={!formData.business_profile}
                  />
                </div>
              </div>
            )}
          </div>
          
          <div className="mt-4 flex gap-2">
            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? 'Adding...' : 'Add Portal'}
            </button>
            <button type="button" onClick={() => setShowForm(false)} className="btn-secondary">
              Cancel
            </button>
          </div>
        </form>
      )}
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {portals.map(portal => (
          <div key={portal.id} className="border rounded-lg p-4 hover:shadow-md transition-shadow">
            <div className="flex justify-between items-start">
              <div className="flex-1">
                <h3 className="font-semibold text-gray-900">{portal.name}</h3>
                <p className="text-sm text-gray-600 mt-1 font-mono">{portal.id}</p>
                <div className="flex items-center gap-2 mt-2">
                  <ZapIcon className="w-3 h-3 text-gray-400" />
                  <p className="text-xs text-gray-500">
                    Every {portal.interval_min} minutes
                  </p>
                </div>
                {portal.business_profile && (
                  <div className="mt-2 flex items-center gap-1">
                    <TargetIcon className="w-3 h-3 text-green-600" />
                    <p className="text-xs text-green-600">
                      AI Filtered (threshold: {portal.relevance_threshold || 60}%)
                    </p>
                  </div>
                )}
              </div>
              <button
                onClick={() => handleDelete(portal.id)}
                className="text-red-500 hover:text-red-700 p-1"
                title="Delete portal"
              >
                <TrashIcon className="w-4 h-4" />
              </button>
            </div>
            <a 
              href={portal.url} 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-sm text-blue-600 hover:underline block mt-2 truncate"
            >
              {portal.url}
            </a>
          </div>
        ))}
      </div>
    </div>
  );
}