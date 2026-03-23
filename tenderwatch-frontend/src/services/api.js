const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = {
  // Portals
  async getPortals() {
    try {
      const response = await fetch(`${API_BASE}/portals`);
      if (!response.ok) throw new Error('Failed to fetch portals');
      const data = await response.json();
      return data || [];
    } catch (error) {
      console.error('Error fetching portals:', error);
      return [];
    }
  },
  
  async createPortal(portalData) {
    const response = await fetch(`${API_BASE}/portals`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(portalData)
    });
    if (!response.ok) throw new Error('Failed to create portal');
    return response.json();
  },
  
  async deletePortal(portalId) {
    const response = await fetch(`${API_BASE}/portals/${portalId}`, {
      method: 'DELETE'
    });
    if (!response.ok) throw new Error('Failed to delete portal');
    return response.json();
  },
  
  // Tenders
  async getTenders(portalId = null) {
    try {
      const url = portalId ? `${API_BASE}/tenders/${portalId}` : `${API_BASE}/tenders`;
      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed to fetch tenders');
      const data = await response.json();
      return data || [];
    } catch (error) {
      console.error('Error fetching tenders:', error);
      return [];
    }
  },
  
  async getTenderStats(tenders) {
    return {
      total: tenders.length,
      new: tenders.filter(t => t.status === 'new').length,
      updated: tenders.filter(t => t.status === 'updated').length
    };
  },
  
  // Analysis
  async triggerAnalysis(portalId, referenceNumber) {
    const encodedRef = encodeURIComponent(referenceNumber);
    const response = await fetch(`${API_BASE}/tenders/${portalId}/${encodedRef}/analyze`, {
      method: 'POST'
    });
    if (!response.ok) throw new Error('Failed to trigger analysis');
    return response.json();
  },
  
  async getAnalysis(portalId, referenceNumber) {
    try {
      const encodedRef = encodeURIComponent(referenceNumber);
      const response = await fetch(`${API_BASE}/tenders/${portalId}/${encodedRef}/analysis`);
      if (response.status === 404) return null;
      if (!response.ok) throw new Error('Failed to fetch analysis');
      return response.json();
    } catch (error) {
      console.error('Error fetching analysis:', error);
      return null;
    }
  }
};