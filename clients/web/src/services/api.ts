import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api';

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 30000,
});

// Интерцептор для добавления токена
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth-token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Интерцептор для обработки ошибок
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Токен истек, пробуем обновить
      const refreshToken = localStorage.getItem('auth-refresh-token');
      if (refreshToken) {
        try {
          const response = await axios.post(`${API_BASE_URL}/auth/refresh`, {
            refreshToken,
          });
          const newToken = response.data.token;
          localStorage.setItem('auth-token', newToken);
          error.config.headers.Authorization = `Bearer ${newToken}`;
          return apiClient.request(error.config);
        } catch (refreshError) {
          // Не удалось обновить токен, выходим
          localStorage.removeItem('auth-token');
          localStorage.removeItem('auth-refresh-token');
          window.location.href = '/login';
          return Promise.reject(refreshError);
        }
      }
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authApi = {
  requestOtp: async (phone: string) => {
    const response = await apiClient.post('/auth/otp/request', { phone });
    return response.data;
  },
  
  verifyOtp: async (phone: string, code: string) => {
    const response = await apiClient.post('/auth/otp/verify', { phone, code });
    return response.data;
  },
  
  enable2FA: async () => {
    const response = await apiClient.post('/auth/2fa/enable');
    return response.data;
  },
  
  verify2FA: async (code: string) => {
    const response = await apiClient.post('/auth/2fa/verify', { code });
    return response.data;
  },
  
  logout: async () => {
    const response = await apiClient.post('/auth/logout');
    return response.data;
  },
};

// Chat API
export const chatApi = {
  getChats: async () => {
    const response = await apiClient.get('/chats');
    return response.data;
  },
  
  createChat: async (participantIds: string[], type: 'private' | 'group' | 'channel', name?: string) => {
    const response = await apiClient.post('/chats', { participantIds, type, name });
    return response.data;
  },
  
  getChat: async (chatId: string) => {
    const response = await apiClient.get(`/chats/${chatId}`);
    return response.data;
  },
  
  deleteChat: async (chatId: string) => {
    const response = await apiClient.delete(`/chats/${chatId}`);
    return response.data;
  },
};

// Message API
export const messageApi = {
  sendMessage: async (chatId: string, content: string, type: 'text' | 'image' | 'video' | 'audio' | 'file' = 'text') => {
    const response = await apiClient.post('/messages', { chatId, content, type });
    return response.data;
  },
  
  editMessage: async (messageId: string, content: string) => {
    const response = await apiClient.put(`/messages/${messageId}`, { content });
    return response.data;
  },
  
  deleteMessage: async (messageId: string) => {
    const response = await apiClient.delete(`/messages/${messageId}`);
    return response.data;
  },
  
  addReaction: async (messageId: string, emoji: string) => {
    const response = await apiClient.post(`/messages/${messageId}/reactions`, { emoji });
    return response.data;
  },
  
  getMessages: async (chatId: string, limit: number = 50, offset: number = 0) => {
    const response = await apiClient.get(`/messages`, {
      params: { chatId, limit, offset },
    });
    return response.data;
  },
};

// Media API
export const mediaApi = {
  upload: async (file: File, chatId: string) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('chatId', chatId);
    
    const response = await apiClient.post('/media/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        const percentCompleted = Math.round((progressEvent.loaded * 100) / (progressEvent.total || 1));
        console.log(`Upload progress: ${percentCompleted}%`);
      },
    });
    return response.data;
  },
  
  getMedia: async (mediaId: string) => {
    const response = await apiClient.get(`/media/${mediaId}`, {
      responseType: 'blob',
    });
    return response.data;
  },
};

// Search API
export const searchApi = {
  search: async (query: string, options?: { chatId?: string; senderId?: string; dateFrom?: string; dateTo?: string }) => {
    const response = await apiClient.get('/search', {
      params: { q: query, ...options },
    });
    return response.data;
  },
};
