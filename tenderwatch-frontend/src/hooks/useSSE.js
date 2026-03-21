import { useEffect, useState } from 'react';

export function useSSE(url) {
  const [events, setEvents] = useState([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState(null);
  
  useEffect(() => {
    let eventSource = null;
    
    const connect = () => {
      try {
        eventSource = new EventSource(url);
        
        eventSource.onopen = () => {
          setIsConnected(true);
          setError(null);
          console.log('SSE connection established');
        };
        
        eventSource.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data);
            setEvents(prev => [...prev, data]);
          } catch (err) {
            console.error('Failed to parse SSE event:', err);
          }
        };
        
        eventSource.onerror = (err) => {
          setIsConnected(false);
          setError('Connection lost. Reconnecting...');
          console.error('SSE error:', err);
          
          // Close and reconnect after delay
          if (eventSource) {
            eventSource.close();
          }
          setTimeout(connect, 5000);
        };
      } catch (err) {
        setError('Failed to connect to SSE');
        console.error('SSE connection error:', err);
      }
    };
    
    connect();
    
    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [url]);
  
  const clearEvents = () => {
    setEvents([]);
  };
  
  return { events, isConnected, error, clearEvents };
}