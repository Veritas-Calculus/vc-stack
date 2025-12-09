import { useState, useCallback } from 'react';
import { useMonitoring } from '../../hooks/useMonitoring';

export function FlameGraphViewer() {
  const {
    loading,
    error,
    generateCPUFlameGraph,
    generateHeapFlameGraph,
    generateGoroutineFlameGraph,
  } = useMonitoring();

  const [svgUrl, setSvgUrl] = useState<string | null>(null);
  const [profileType, setProfileType] = useState<string>('');
  const [duration, setDuration] = useState('30s');

  const generateFlameGraph = useCallback(
    async (type: 'cpu' | 'heap' | 'goroutine') => {
      setSvgUrl(null);
      setProfileType(type);

      let result;
      if (type === 'cpu') {
        result = await generateCPUFlameGraph(duration);
      } else if (type === 'heap') {
        result = await generateHeapFlameGraph();
      } else {
        result = await generateGoroutineFlameGraph();
      }

      if (result && result.success) {
        setSvgUrl(result.download_svg);
      }
    },
    [
      duration,
      generateCPUFlameGraph,
      generateHeapFlameGraph,
      generateGoroutineFlameGraph,
    ]
  );

  return (
    <div className="p-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold">Flamegraph Generator</h1>
      </div>

      <div className="bg-white p-6 rounded-lg shadow">
        <h2 className="text-xl font-semibold mb-4">Generate Flamegraph</h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div>
            <label className="block text-sm font-medium mb-2">
              CPU Profile Duration
            </label>
            <select
              value={duration}
              onChange={(e) => setDuration(e.target.value)}
              className="w-full px-4 py-2 border rounded"
            >
              <option value="10s">10 seconds</option>
              <option value="30s">30 seconds</option>
              <option value="1m">1 minute</option>
              <option value="2m">2 minutes</option>
              <option value="5m">5 minutes</option>
            </select>
          </div>
        </div>

        <div className="flex gap-4">
          <button
            onClick={() => generateFlameGraph('cpu')}
            disabled={loading}
            className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 disabled:bg-gray-300"
          >
            {loading && profileType === 'cpu'
              ? 'Generating...'
              : 'Generate CPU Flamegraph'}
          </button>
          <button
            onClick={() => generateFlameGraph('heap')}
            disabled={loading}
            className="px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 disabled:bg-gray-300"
          >
            {loading && profileType === 'heap'
              ? 'Generating...'
              : 'Generate Heap Flamegraph'}
          </button>
          <button
            onClick={() => generateFlameGraph('goroutine')}
            disabled={loading}
            className="px-4 py-2 bg-purple-500 text-white rounded hover:bg-purple-600 disabled:bg-gray-300"
          >
            {loading && profileType === 'goroutine'
              ? 'Generating...'
              : 'Generate Goroutine Flamegraph'}
          </button>
        </div>

        {error && (
          <div className="mt-4 p-4 bg-red-100 text-red-700 rounded">
            {error}
          </div>
        )}
      </div>

      {svgUrl && (
        <div className="bg-white p-6 rounded-lg shadow">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-semibold">
              {profileType.charAt(0).toUpperCase() + profileType.slice(1)}{' '}
              Flamegraph
            </h2>
            <a
              href={svgUrl}
              download
              className="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
            >
              Download SVG
            </a>
          </div>
          <div className="border rounded p-4 overflow-auto">
            <iframe
              src={svgUrl}
              className="w-full h-96 border-0"
              title="Flamegraph"
            />
          </div>
        </div>
      )}

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold mb-2">About Flamegraphs</h3>
        <ul className="list-disc list-inside space-y-1 text-sm">
          <li>
            <strong>CPU Flamegraph:</strong> Shows where CPU time is spent
          </li>
          <li>
            <strong>Heap Flamegraph:</strong> Shows memory allocation patterns
          </li>
          <li>
            <strong>Goroutine Flamegraph:</strong> Shows goroutine stack traces
          </li>
        </ul>
      </div>
    </div>
  );
}
