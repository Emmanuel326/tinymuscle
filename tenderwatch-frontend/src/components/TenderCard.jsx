import { useState, useEffect } from 'react';
import { formatDistanceToNow, format } from 'date-fns';
import { Calendar, Building2, Link, TrendingUp, Target, Brain, Loader2 } from 'lucide-react';
import { api } from '../services/api';
import { AnalysisCard } from './AnalysisCard';
import toast from 'react-hot-toast';

export function TenderCard({ tender }) {
  const [analyzing, setAnalyzing] = useState(false);
  const [analysis, setAnalysis] = useState(null);
  const [showAnalysis, setShowAnalysis] = useState(false);
  const [polling, setPolling] = useState(null);
  
  if (!tender) return null;
  
  const getStatusBadge = () => {
    switch (tender.status) {
      case 'new':
        return <span className="badge badge-new">🆕 New</span>;
      case 'updated':
        return <span className="badge badge-updated">📝 Updated v{tender.version}</span>;
      default:
        return null;
    }
  };
  
  const deadlineDate = tender.deadline && tender.deadline !== '0001-01-01T00:00:00Z' 
    ? new Date(tender.deadline) 
    : null;
  const isDeadlineSoon = deadlineDate && (deadlineDate - new Date() < 7 * 24 * 60 * 60 * 1000);
  
  const handleAnalyze = async () => {
    setAnalyzing(true);
    try {
      await api.triggerAnalysis(tender.portal_id, tender.reference_number);
      toast.success('Analysis started! This may take 2-5 minutes...');
      
      // Start polling
      const pollInterval = setInterval(async () => {
        const result = await api.getAnalysis(tender.portal_id, tender.reference_number);
        if (result) {
          setAnalysis(result);
          setShowAnalysis(true);
          setAnalyzing(false);
          clearInterval(pollInterval);
          setPolling(null);
          toast.success('Analysis complete!');
        }
      }, 30000); // Poll every 30 seconds
      
      setPolling(pollInterval);
    } catch (error) {
      toast.error('Failed to start analysis');
      setAnalyzing(false);
    }
  };
  
  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (polling) clearInterval(polling);
    };
  }, [polling]);
  
  return (
    <>
      <div className="card p-6 hover:scale-[1.02] transition-transform">
        <div className="flex justify-between items-start mb-3">
          <h3 className="text-xl font-semibold text-gray-900 flex-1">{tender.title}</h3>
          {getStatusBadge()}
        </div>
        
        <div className="space-y-2 text-sm text-gray-600">
          <div className="flex items-center gap-2">
            <Building2 className="w-4 h-4" />
            <span>{tender.issuing_entity || 'Unknown'}</span>
          </div>
          
          <div className="flex items-center gap-2">
            <Calendar className="w-4 h-4" />
            <span className={isDeadlineSoon && tender.status !== 'closed' ? 'text-red-600 font-semibold' : ''}>
              Deadline: {deadlineDate ? format(deadlineDate, 'PPP') : '—'}
              {isDeadlineSoon && ' ⚠️ Soon!'}
            </span>
          </div>
          
          {tender.estimated_value && (
            <div className="flex items-center gap-2">
              <TrendingUp className="w-4 h-4" />
              <span>Estimated Value: {tender.estimated_value}</span>
            </div>
          )}
          
          {tender.relevance_score && (
            <div className="flex items-center gap-2">
              <Target className="w-4 h-4" />
              <span>Relevance Score: 
                <span className={`font-semibold ml-1 ${
                  tender.relevance_score >= 70 ? 'text-green-600' :
                  tender.relevance_score >= 40 ? 'text-yellow-600' :
                  'text-gray-600'
                }`}>
                  {tender.relevance_score}%
                </span>
              </span>
            </div>
          )}
          
          <div className="flex items-center gap-2">
            <Link className="w-4 h-4" />
            <a 
              href={tender.source_url || '#'} 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-blue-600 hover:text-blue-800 hover:underline"
            >
              View Tender
            </a>
          </div>
          
          <div className="mt-3 pt-3 border-t border-gray-100 text-xs text-gray-400">
            <div>Reference: <span className="font-mono">{tender.reference_number || 'N/A'}</span></div>
            <div>Portal: {tender.portal_id}</div>
            <div>Updated: {tender.last_updated ? formatDistanceToNow(new Date(tender.last_updated)) : 'Unknown'} ago</div>
          </div>
          
          {/* Analyze Button */}
          <button
            onClick={handleAnalyze}
            disabled={analyzing}
            className="mt-3 w-full btn-secondary flex items-center justify-center gap-2"
          >
            {analyzing ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Analyzing...
              </>
            ) : (
              <>
                <Brain className="w-4 h-4" />
                Analyze with AI
              </>
            )}
          </button>
        </div>
      </div>
      
      {/* Analysis Modal */}
      {showAnalysis && analysis && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4 overflow-y-auto">
          <div className="max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            <AnalysisCard 
              analysis={analysis} 
              tender={tender}
              onClose={() => setShowAnalysis(false)}
            />
          </div>
        </div>
      )}
    </>
  );
}