import { useState, useEffect, useRef, useCallback } from 'react';
import { getWebSocketUrl, startAudioStream, stopAudioStream } from '../services';
import type { AudioStreamData, AudioTuningConfig } from '../types/api';

export interface ChartDataPoint {
  time: number;
  bassRaw: number;
  midLowRaw: number;
  midHighRaw: number;
  trebleRaw: number;
  bassProcessed: number;
  midLowProcessed: number;
  midHighProcessed: number;
  trebleProcessed: number;
}

const MAX_DATA_POINTS = 1200; // 30 seconds at 40 Hz

export interface UseAudioStreamOptions {
  onConfigReceived?: (config: AudioTuningConfig) => void;
}

export interface UseAudioStreamResult {
  chartData: ChartDataPoint[];
  isStreaming: boolean;
  isConnected: boolean;
  error: string | null;
  startStreaming: () => Promise<void>;
  stopStreaming: () => Promise<void>;
  clearData: () => void;
}

export function useAudioStream(options: UseAudioStreamOptions = {}): UseAudioStreamResult {
  const [chartData, setChartData] = useState<ChartDataPoint[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const startTimeRef = useRef<number>(Date.now());
  const isStreamingRef = useRef<boolean>(false);
  const configReceivedRef = useRef<boolean>(false);
  const onConfigReceivedRef = useRef(options.onConfigReceived);

  // Keep refs in sync
  useEffect(() => {
    isStreamingRef.current = isStreaming;
  }, [isStreaming]);

  useEffect(() => {
    onConfigReceivedRef.current = options.onConfigReceived;
  }, [options.onConfigReceived]);

  const connectWebSocket = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const wsUrl = getWebSocketUrl('/api/audio/tuning/stream');
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('Audio WebSocket connected');
      setIsConnected(true);
      startTimeRef.current = Date.now();
    };

    ws.onmessage = (event) => {
      try {
        const data: AudioStreamData = JSON.parse(event.data);

        // Notify about config on first message
        if (data.config && !configReceivedRef.current && onConfigReceivedRef.current) {
          onConfigReceivedRef.current(data.config);
          configReceivedRef.current = true;
        }

        // Only add data point if streaming is active
        if (data.raw && data.processed && isStreamingRef.current) {
          const timeSeconds = (Date.now() - startTimeRef.current) / 1000;

          setChartData((prev) => {
            const newPoint: ChartDataPoint = {
              time: timeSeconds,
              bassRaw: data.raw.bass,
              midLowRaw: data.raw.mid_low,
              midHighRaw: data.raw.mid_high,
              trebleRaw: data.raw.treble,
              bassProcessed: data.processed.bass,
              midLowProcessed: data.processed.mid_low,
              midHighProcessed: data.processed.mid_high,
              trebleProcessed: data.processed.treble,
            };

            const newData = [...prev, newPoint];
            return newData.slice(-MAX_DATA_POINTS);
          });
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onerror = () => {
      setError('WebSocket connection error');
    };

    ws.onclose = () => {
      console.log('Audio WebSocket disconnected');
      setIsConnected(false);
    };

    wsRef.current = ws;
  }, []);

  const disconnectWebSocket = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
      setIsConnected(false);
    }
  }, []);

  useEffect(() => {
    connectWebSocket();
    return () => {
      disconnectWebSocket();
    };
  }, [connectWebSocket, disconnectWebSocket]);

  const startStreaming = async () => {
    try {
      setError(null);
      await startAudioStream();
      setIsStreaming(true);
      setChartData([]);
      startTimeRef.current = Date.now();
    } catch (err) {
      setError('Failed to start streaming: ' + (err as Error).message);
      throw err;
    }
  };

  const stopStreaming = async () => {
    try {
      setError(null);
      await stopAudioStream();
      setIsStreaming(false);
    } catch (err) {
      setError('Failed to stop streaming: ' + (err as Error).message);
      throw err;
    }
  };

  const clearData = () => {
    setChartData([]);
    startTimeRef.current = Date.now();
  };

  return {
    chartData,
    isStreaming,
    isConnected,
    error,
    startStreaming,
    stopStreaming,
    clearData,
  };
}
