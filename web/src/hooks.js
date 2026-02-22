import { useState, useEffect, useRef, useCallback } from 'react';

export function usePulls() {
  const [pulls, setPulls] = useState([]);
  const [connected, setConnected] = useState(false);
  const eventSourceRef = useRef(null);

  // Initial fetch
  useEffect(() => {
    fetch('/api/v1/pulls')
      .then((res) => res.json())
      .then((data) => {
        if (data.pulls) {
          setPulls(data.pulls);
        }
      })
      .catch(() => {});
  }, []);

  // SSE connection
  useEffect(() => {
    function connect() {
      const es = new EventSource('/api/v1/events');
      eventSourceRef.current = es;

      es.onopen = () => setConnected(true);

      es.onmessage = (event) => {
        try {
          const evt = JSON.parse(event.data);
          if (evt.pull) {
            const pull = { ...evt.pull };
            // Prefer top-level nodeName (always set since the fix), fall back to pull.nodeName
            if (evt.nodeName && !pull.nodeName) {
              pull.nodeName = evt.nodeName;
            }
            setPulls((prev) => {
              const idx = prev.findIndex((p) => p.id === pull.id);
              if (idx === -1) return [...prev, pull];
              const next = [...prev];
              next[idx] = pull;
              return next;
            });
          }
        } catch {
          // ignore parse errors
        }
      };

      es.onerror = () => {
        setConnected(false);
        es.close();
        setTimeout(connect, 3000);
      };
    }

    connect();

    return () => {
      if (eventSourceRef.current) eventSourceRef.current.close();
    };
  }, []);

  return { pulls, connected };
}

export function useFilters() {
  const [filters, setFilters] = useState({
    namespace: '',
    node: '',
    pod: '',
    image: '',
  });

  const setFilter = useCallback((key, value) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  }, []);

  const filterPulls = useCallback(
    (pulls) => {
      return pulls.filter((pull) => {
        if (filters.image && !pull.imageRef?.toLowerCase().includes(filters.image.toLowerCase())) {
          return false;
        }
        if (filters.node && !pull.nodeName?.toLowerCase().includes(filters.node.toLowerCase())) {
          return false;
        }
        if (filters.namespace) {
          const hasNs = pull.pods?.some((p) =>
            p.namespace?.toLowerCase().includes(filters.namespace.toLowerCase())
          );
          if (!hasNs) return false;
        }
        if (filters.pod) {
          const hasPod = pull.pods?.some((p) =>
            p.podName?.toLowerCase().includes(filters.pod.toLowerCase())
          );
          if (!hasPod) return false;
        }
        return true;
      });
    },
    [filters]
  );

  return { filters, setFilter, filterPulls };
}
