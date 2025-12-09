import { useState, useEffect } from 'react';
import axios from 'axios';

interface MetricData {
  [key: string]: unknown;
}

interface ComponentMetrics {
  component: string;
  duration: string;
  metrics: MetricData[];
}

interface SystemMetrics {
  duration: string;
  metrics: MetricData[];
}

interface PerformanceAnalysis {
  duration: string;
  analyzed_at: string;
  system_metrics: MetricData[];
  http_metrics: MetricData[];
  issues: string[];
  recommendations: string[];
}

interface FlameGraphResult {
  success: boolean;
  svg_url: string;
  profile_url: string;
  duration?: string;
  timestamp: string;
  download_svg: string;
}

const API_BASE = '/v1/monitoring';

export function useMonitoring() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const getComponentMetrics = async (
    component: string,
    duration: string = '1h'
  ): Promise<ComponentMetrics | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get(
        `${API_BASE}/metrics/component/${component}`,
        {
          params: { duration },
        }
      );
      return response.data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      return null;
    } finally {
      setLoading(false);
    }
  };

  const getSystemMetrics = async (
    duration: string = '1h'
  ): Promise<SystemMetrics | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get(`${API_BASE}/metrics/system`, {
        params: { duration },
      });
      return response.data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      return null;
    } finally {
      setLoading(false);
    }
  };

  const getHTTPMetrics = async (
    duration: string = '1h'
  ): Promise<{ duration: string; metrics: MetricData[] } | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get(`${API_BASE}/metrics/http`, {
        params: { duration },
      });
      return response.data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      return null;
    } finally {
      setLoading(false);
    }
  };

  const getErrorMetrics = async (
    component: string,
    duration: string = '1h'
  ): Promise<ComponentMetrics | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get(
        `${API_BASE}/metrics/errors/${component}`,
        {
          params: { duration },
        }
      );
      return response.data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch metrics');
      return null;
    } finally {
      setLoading(false);
    }
  };

  const generateCPUFlameGraph = async (
    duration: string = '30s'
  ): Promise<FlameGraphResult | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.post(`${API_BASE}/flamegraph/cpu`, null, {
        params: { duration },
      });
      return response.data;
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to generate flamegraph'
      );
      return null;
    } finally {
      setLoading(false);
    }
  };

  const generateHeapFlameGraph = async (): Promise<FlameGraphResult | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.post(`${API_BASE}/flamegraph/heap`);
      return response.data;
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to generate flamegraph'
      );
      return null;
    } finally {
      setLoading(false);
    }
  };

  const generateGoroutineFlameGraph =
    async (): Promise<FlameGraphResult | null> => {
      setLoading(true);
      setError(null);
      try {
        const response = await axios.post(`${API_BASE}/flamegraph/goroutine`);
        return response.data;
      } catch (err) {
        setError(
          err instanceof Error ? err.message : 'Failed to generate flamegraph'
        );
        return null;
      } finally {
        setLoading(false);
      }
    };

  const analyzePerformance = async (
    duration: string = '5m'
  ): Promise<PerformanceAnalysis | null> => {
    setLoading(true);
    setError(null);
    try {
      const response = await axios.get(`${API_BASE}/profile/analyze`, {
        params: { duration },
      });
      return response.data;
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to analyze performance'
      );
      return null;
    } finally {
      setLoading(false);
    }
  };

  return {
    loading,
    error,
    getComponentMetrics,
    getSystemMetrics,
    getHTTPMetrics,
    getErrorMetrics,
    generateCPUFlameGraph,
    generateHeapFlameGraph,
    generateGoroutineFlameGraph,
    analyzePerformance,
  };
}

export function useAutoRefreshMetrics<T>(
  fetchFn: () => Promise<T | null>,
  interval: number = 10000
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      setError(null);
      try {
        const result = await fetchFn();
        setData(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch data');
      } finally {
        setLoading(false);
      }
    };

    fetch();
    const intervalId = setInterval(fetch, interval);

    return () => clearInterval(intervalId);
  }, [fetchFn, interval]);

  return { data, loading, error };
}
