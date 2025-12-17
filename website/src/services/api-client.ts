const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export class ApiError extends Error {
  status: number;
  statusText: string;

  constructor(message: string, status: number, statusText: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.statusText = statusText;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const text = await response.text();
    throw new ApiError(
      text || response.statusText,
      response.status,
      response.statusText
    );
  }

  const contentType = response.headers.get('content-type');
  if (contentType?.includes('application/json')) {
    return response.json();
  }

  return {} as T;
}

export async function get<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`);
  return handleResponse<T>(response);
}

export async function post<T>(endpoint: string, data?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: data ? JSON.stringify(data) : undefined,
  });
  return handleResponse<T>(response);
}

export async function put<T>(endpoint: string, data?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: data ? JSON.stringify(data) : undefined,
  });
  return handleResponse<T>(response);
}

export function getWebSocketUrl(endpoint: string): string {
  const wsProtocol = API_BASE_URL.startsWith('https') ? 'wss' : 'ws';
  const url = API_BASE_URL.replace(/^https?:\/\//, '');
  return `${wsProtocol}://${url}${endpoint}`;
}
