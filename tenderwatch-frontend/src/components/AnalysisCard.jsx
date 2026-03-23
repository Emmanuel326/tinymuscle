import { useState } from 'react';
import { CheckCircle, XCircle, Copy, Mail, Calendar, DollarSign } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import toast from 'react-hot-toast';

export function AnalysisCard({ analysis, tender, onClose }) {
  const [copied, setCopied] = useState(false);
  
  const copyToClipboard = () => {
    navigator.clipboard.writeText(analysis.draft_response);
    setCopied(true);
    toast.success('Draft response copied to clipboard!');
    setTimeout(() => setCopied(false), 2000);
  };
  
  return (
    <div className="bg-white rounded-lg shadow-lg border border-purple-200 overflow-hidden">
      <div className="bg-purple-50 px-6 py-4 border-b border-purple-200">
        <div className="flex justify-between items-center">
          <h3 className="text-lg font-semibold text-purple-900">🧠 AI Analysis</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            ×
          </button>
        </div>
        <p className="text-xs text-purple-600 mt-1">
          Analyzed {formatDistanceToNow(new Date(analysis.analyzed_at))} ago
        </p>
      </div>
      
      <div className="p-6 space-y-6">
        {/* Qualifies Badge */}
        <div className="flex items-center gap-3 p-4 rounded-lg bg-gray-50">
          {analysis.qualifies ? (
            <>
              <CheckCircle className="w-8 h-8 text-green-600" />
              <div>
                <p className="font-semibold text-green-700">✓ Your business qualifies!</p>
                <p className="text-sm text-gray-600">This tender matches your profile</p>
              </div>
            </>
          ) : (
            <>
              <XCircle className="w-8 h-8 text-red-600" />
              <div>
                <p className="font-semibold text-red-700">✗ Low qualification match</p>
                <p className="text-sm text-gray-600">Review reasons below</p>
              </div>
            </>
          )}
        </div>
        
        {/* Qualify Reasons */}
        {analysis.qualify_reasons && analysis.qualify_reasons.length > 0 && (
          <div>
            <h4 className="font-semibold text-gray-900 mb-2">Why we think this:</h4>
            <ul className="list-disc list-inside space-y-1 text-sm text-gray-600">
              {analysis.qualify_reasons.map((reason, i) => (
                <li key={i}>{reason}</li>
              ))}
            </ul>
          </div>
        )}
        
        {/* Summary */}
        <div>
          <h4 className="font-semibold text-gray-900 mb-2">Summary</h4>
          <p className="text-gray-700">{analysis.summary}</p>
        </div>
        
        {/* Key Info */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 p-4 bg-gray-50 rounded-lg">
          {analysis.deadline && analysis.deadline !== '0001-01-01T00:00:00Z' && (
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-gray-500" />
              <div>
                <p className="text-xs text-gray-500">Deadline</p>
                <p className="text-sm font-medium">{new Date(analysis.deadline).toLocaleDateString()}</p>
              </div>
            </div>
          )}
          {analysis.estimated_value && (
            <div className="flex items-center gap-2">
              <DollarSign className="w-4 h-4 text-gray-500" />
              <div>
                <p className="text-xs text-gray-500">Estimated Value</p>
                <p className="text-sm font-medium">{analysis.estimated_value}</p>
              </div>
            </div>
          )}
          {analysis.contact_person && analysis.contact_person.includes('@') && (
            <div className="flex items-center gap-2">
              <Mail className="w-4 h-4 text-gray-500" />
              <div>
                <p className="text-xs text-gray-500">Contact</p>
                <a href={`mailto:${analysis.contact_person}`} className="text-sm text-blue-600 hover:underline">
                  {analysis.contact_person}
                </a>
              </div>
            </div>
          )}
        </div>
        
        {/* Eligibility Criteria */}
        {analysis.eligibility_criteria && analysis.eligibility_criteria.length > 0 && (
          <div>
            <h4 className="font-semibold text-gray-900 mb-2">Eligibility Criteria</h4>
            <ul className="list-disc list-inside space-y-1 text-sm text-gray-600">
              {analysis.eligibility_criteria.map((criterion, i) => (
                <li key={i}>{criterion}</li>
              ))}
            </ul>
          </div>
        )}
        
        {/* Required Documents */}
        {analysis.required_documents && analysis.required_documents.length > 0 && (
          <div>
            <h4 className="font-semibold text-gray-900 mb-2">Required Documents</h4>
            <ul className="list-disc list-inside space-y-1 text-sm text-gray-600">
              {analysis.required_documents.map((doc, i) => (
                <li key={i}>{doc}</li>
              ))}
            </ul>
          </div>
        )}
        
        {/* Evaluation Criteria */}
        {analysis.evaluation_criteria && analysis.evaluation_criteria.length > 0 && (
          <div>
            <h4 className="font-semibold text-gray-900 mb-2">Evaluation Criteria</h4>
            <ul className="list-decimal list-inside space-y-1 text-sm text-gray-600">
              {analysis.evaluation_criteria.map((criterion, i) => (
                <li key={i}>{criterion}</li>
              ))}
            </ul>
          </div>
        )}
        
        {/* Draft Response */}
        {analysis.draft_response && (
          <div>
            <div className="flex justify-between items-center mb-2">
              <h4 className="font-semibold text-gray-900">📝 Draft Response</h4>
              <button
                onClick={copyToClipboard}
                className="flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800"
              >
                <Copy className="w-4 h-4" />
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            <textarea
              value={analysis.draft_response}
              readOnly
              rows={8}
              className="w-full p-3 border rounded-lg bg-gray-50 font-mono text-sm"
            />
            <p className="text-xs text-gray-500 mt-2">
              ⚠️ Review before sending — TinyMuscle never auto-submits responses
            </p>
          </div>
        )}
      </div>
    </div>
  );
}