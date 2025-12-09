import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { Play, Pause, SkipBack, SkipForward, ArrowLeft, Download } from 'lucide-react';
import { resolveApiBase } from '@/lib/api';
import '@xterm/xterm/css/xterm.css';

const api = resolveApiBase();

interface WebShellEvent {
  id: number;
  session_id: string;
  event_type: string;
  event_time: string;
  data?: string;
  data_size: number;
  cols?: number;
  rows?: number;
  time_offset: number;
  created_at: string;
}

interface SessionInfo {
  id: number;
  session_id: string;
  username: string;
  remote_host: string;
  remote_port: number;
  remote_user: string;
  auth_method: string;
  status: string;
  started_at: string;
  ended_at?: string;
  duration_seconds?: number;
  bytes_sent: number;
  bytes_received: number;
  client_ip?: string;
}

interface SessionEventsResponse {
  session: SessionInfo;
  events: WebShellEvent[];
}

export function SessionReplay() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [events, setEvents] = useState<WebShellEvent[]>([]);
  const [loading, setLoading] = useState(true);
  
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentEventIndex, setCurrentEventIndex] = useState(0);
  const [currentTime, setCurrentTime] = useState(0);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  
  const playbackTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Load session data
  useEffect(() => {
    const loadSession = async () => {
      if (!sessionId) return;
      
      try {
        const response = await fetch(`${api}/v1/webshell/sessions/${sessionId}/events`, {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
          },
        });

        if (response.ok) {
          const data: SessionEventsResponse = await response.json();
          setSession(data.session);
          setEvents(data.events);
        }
      } catch {
        // Error loading session
      } finally {
        setLoading(false);
      }
    };

    loadSession();
  }, [sessionId]);

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current || !session) return;

    const term = new XTerm({
      cursorBlink: false,
      disableStdin: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    const handleResize = () => {
      fitAddon.fit();
    };
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      term.dispose();
    };
  }, [session]);

  // Playback logic
  useEffect(() => {
    if (!isPlaying || events.length === 0) {
      if (playbackTimerRef.current) {
        clearTimeout(playbackTimerRef.current);
        playbackTimerRef.current = null;
      }
      return;
    }

    const playNextEvent = () => {
      if (currentEventIndex >= events.length) {
        setIsPlaying(false);
        return;
      }

      const event = events[currentEventIndex];
      const term = xtermRef.current;
      if (!term) return;

      // Handle different event types
      switch (event.event_type) {
        case 'output':
          if (event.data) {
            term.write(event.data);
          }
          break;
        case 'resize':
          if (event.cols && event.rows) {
            term.resize(event.cols, event.rows);
          }
          break;
        case 'connected':
          if (event.data) {
            term.write(`\r\n${event.data}\r\n`);
          }
          break;
      }

      setCurrentTime(event.time_offset);
      setCurrentEventIndex(currentEventIndex + 1);

      // Schedule next event
      if (currentEventIndex + 1 < events.length) {
        const nextEvent = events[currentEventIndex + 1];
        const delay = (nextEvent.time_offset - event.time_offset) / playbackSpeed;
        playbackTimerRef.current = setTimeout(playNextEvent, delay);
      } else {
        setIsPlaying(false);
      }
    };

    playNextEvent();

    return () => {
      if (playbackTimerRef.current) {
        clearTimeout(playbackTimerRef.current);
        playbackTimerRef.current = null;
      }
    };
  }, [isPlaying, currentEventIndex, events, playbackSpeed]);

  const handlePlayPause = () => {
    if (currentEventIndex >= events.length) {
      // Restart from beginning
      handleRestart();
    } else {
      setIsPlaying(!isPlaying);
    }
  };

  const handleRestart = () => {
    setCurrentEventIndex(0);
    setCurrentTime(0);
    setIsPlaying(false);
    
    const term = xtermRef.current;
    if (term) {
      term.clear();
    }
  };

  const handleSkipBackward = () => {
    const newIndex = Math.max(0, currentEventIndex - 10);
    setCurrentEventIndex(newIndex);
    setIsPlaying(false);
    
    // Replay from start to new position
    const term = xtermRef.current;
    if (term) {
      term.clear();
      for (let i = 0; i < newIndex; i++) {
        const event = events[i];
        if (event.event_type === 'output' && event.data) {
          term.write(event.data);
        }
      }
      if (newIndex > 0) {
        setCurrentTime(events[newIndex - 1].time_offset);
      }
    }
  };

  const handleSkipForward = () => {
    const newIndex = Math.min(events.length, currentEventIndex + 10);
    
    // Play events from current to new position immediately
    const term = xtermRef.current;
    if (term) {
      for (let i = currentEventIndex; i < newIndex; i++) {
        const event = events[i];
        if (event.event_type === 'output' && event.data) {
          term.write(event.data);
        }
      }
    }
    
    setCurrentEventIndex(newIndex);
    if (newIndex > 0 && newIndex <= events.length) {
      setCurrentTime(events[newIndex - 1].time_offset);
    }
  };

  const handleSeek = (timeMs: number) => {
    setIsPlaying(false);
    
    // Find the event closest to the target time
    let targetIndex = 0;
    for (let i = 0; i < events.length; i++) {
      if (events[i].time_offset <= timeMs) {
        targetIndex = i + 1;
      } else {
        break;
      }
    }
    
    // Replay from start to target position
    const term = xtermRef.current;
    if (term) {
      term.clear();
      for (let i = 0; i < targetIndex; i++) {
        const event = events[i];
        if (event.event_type === 'output' && event.data) {
          term.write(event.data);
        }
      }
    }
    
    setCurrentEventIndex(targetIndex);
    setCurrentTime(timeMs);
  };

  const formatTime = (ms: number): string => {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  };

  const totalDuration = events.length > 0 ? events[events.length - 1].time_offset : 0;

  const exportSession = async () => {
    if (!sessionId) return;
    
    try {
      const response = await fetch(`${api}/v1/webshell/sessions/${sessionId}/export`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      });

      if (response.ok) {
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `webshell-${sessionId}-${session?.started_at}.cast`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
      }
    } catch {
      // Error exporting session
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-500 dark:text-gray-400">加载中...</div>
      </div>
    );
  }

  if (!session) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">会话未找到</p>
          <button
            onClick={() => navigate('/webshell/sessions')}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            返回列表
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-gray-100 dark:bg-gray-900">
      {/* Header */}
      <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              onClick={() => navigate('/webshell/sessions')}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
            >
              <ArrowLeft className="w-5 h-5" />
            </button>
            <div>
              <h1 className="text-xl font-bold text-gray-900 dark:text-white">
                会话回放
              </h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">
                {session.username} @ {session.remote_host}:{session.remote_port} · {new Date(session.started_at).toLocaleString('zh-CN')}
              </p>
            </div>
          </div>
          
          <button
            onClick={exportSession}
            className="flex items-center gap-2 px-4 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            <Download className="w-4 h-4" />
            导出
          </button>
        </div>
      </div>

      {/* Terminal */}
      <div className="flex-1 p-6">
        <div className="h-full bg-[#1e1e1e] rounded-lg overflow-hidden shadow-lg">
          <div ref={terminalRef} className="w-full h-full p-4" />
        </div>
      </div>

      {/* Controls */}
      <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
        <div className="space-y-4">
          {/* Progress bar */}
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600 dark:text-gray-400 font-mono w-16">
              {formatTime(currentTime)}
            </span>
            <div className="flex-1">
              <input
                type="range"
                min={0}
                max={totalDuration}
                value={currentTime}
                onChange={(e) => handleSeek(parseInt(e.target.value))}
                className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer"
                style={{
                  background: `linear-gradient(to right, #3b82f6 0%, #3b82f6 ${(currentTime / totalDuration) * 100}%, #e5e7eb ${(currentTime / totalDuration) * 100}%, #e5e7eb 100%)`,
                }}
              />
            </div>
            <span className="text-sm text-gray-600 dark:text-gray-400 font-mono w-16">
              {formatTime(totalDuration)}
            </span>
          </div>

          {/* Control buttons */}
          <div className="flex items-center justify-center gap-4">
            <button
              onClick={handleRestart}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              title="重新开始"
            >
              <SkipBack className="w-5 h-5" />
            </button>
            
            <button
              onClick={handleSkipBackward}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              title="后退"
            >
              <SkipBack className="w-4 h-4" />
            </button>
            
            <button
              onClick={handlePlayPause}
              className="p-3 bg-blue-600 text-white rounded-full hover:bg-blue-700"
            >
              {isPlaying ? (
                <Pause className="w-6 h-6" />
              ) : (
                <Play className="w-6 h-6" />
              )}
            </button>
            
            <button
              onClick={handleSkipForward}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              title="前进"
            >
              <SkipForward className="w-4 h-4" />
            </button>
            
            <div className="flex items-center gap-2 ml-4">
              <span className="text-sm text-gray-600 dark:text-gray-400">速度:</span>
              <select
                value={playbackSpeed}
                onChange={(e) => setPlaybackSpeed(parseFloat(e.target.value))}
                className="px-2 py-1 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-sm"
              >
                <option value={0.5}>0.5x</option>
                <option value={1}>1x</option>
                <option value={1.5}>1.5x</option>
                <option value={2}>2x</option>
                <option value={3}>3x</option>
              </select>
            </div>
          </div>
          
          {/* Stats */}
          <div className="flex items-center justify-center gap-8 text-sm text-gray-600 dark:text-gray-400">
            <span>事件: {currentEventIndex} / {events.length}</span>
            <span>上传: {formatBytes(session.bytes_sent)}</span>
            <span>下载: {formatBytes(session.bytes_received)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}
