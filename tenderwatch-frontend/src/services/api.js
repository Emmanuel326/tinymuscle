const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const api = {
  async getPortals() {
    try {
      const response = await fetch(`${API_BASE}/portals`);
      if (!response.ok) throw new Error('Failed to fetch portals');
      const data = await response.json();
      return data || []; // Ensure we always return an array
    } catch (error) {
      console.error('Error fetching portals:', error);
      return []; // Return empty array on error
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
  
  async getTenders(portalId = null) {
    try {
      const url = portalId ? `${API_BASE}/tenders/${portalId}` : `${API_BASE}/tenders`;
      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed to fetch tenders');
      const data = await response.json();
      return data || []; // Ensure we always return an array
    } catch (error) {
      console.error('Error fetching tenders:', error);
      return []; // Return empty array on error
    }
  },
  
  async getTenderStats() {
    const tenders = await this.getTenders();
    return {
      total: tenders.length,
      new: tenders.filter(t => t.status === 'new').length,
      updated: tenders.filter(t => t.status === 'updated').length
    };
  }
};


