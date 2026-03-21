import { formatDistanceToNow, format } from 'date-fns';
import { Calendar, Building2, Link, TrendingUp, Target } from 'lucide-react';

export function TenderCard({ tender }) {
  if (!tender) return null;
  
  const getStatusBadge = () => {
    switch (tender.status) {
      case 'new':
        return <span className="badge badge-new">🆕 New</span>;
      case 'updated':
        return <span className="badge badge-updated">📝 Updated</span>;
      default:
        return null;
    }
  };
  
  const deadlineDate = tender.deadline ? new Date(tender.deadline) : new Date();
  const isDeadlineSoon = deadlineDate - new Date() < 7 * 24 * 60 * 60 * 1000;
  
  return (
    <div className="card p-6 hover:scale-[1.02] transition-transform">
      <div className="flex justify-between items-start mb-3">
        <h3 className="text-xl font-semibold text-gray-900 flex-1">{tender.title || 'Untitled'}</h3>
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
            Deadline: {tender.deadline ? format(deadlineDate, 'PPP') : 'Not specified'}
            {isDeadlineSoon && tender.status !== 'closed' && ' ⚠️ Soon!'}
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
          <div>Reference: {tender.reference_number || 'N/A'}</div>
          <div>Version: {tender.version || 1}</div>
          <div>Updated: {tender.last_updated ? formatDistanceToNow(new Date(tender.last_updated)) : 'Unknown'} ago</div>
        </div>
      </div>
    </div>
  );
}