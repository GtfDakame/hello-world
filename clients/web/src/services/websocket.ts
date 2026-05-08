export class WebSocketService {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private messageHandlers: Set<(data: any) => void> = new Set();
  private statusHandlers: Set<(status: 'connected' | 'disconnected' | 'reconnecting') => void> = new Set();
  private pingInterval: number | null = null;

  connect(token: string) {
    const wsUrl = (import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws') + `?token=${token}`;
    
    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.notifyStatus('connected');
        this.startPing();
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket closed', event.code, event.reason);
        this.stopPing();
        this.notifyStatus('disconnected');
        this.attemptReconnect(token);
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error', error);
      };

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.notifyMessage(data);
        } catch (e) {
          console.error('Failed to parse WebSocket message', e);
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket', error);
    }
  }

  private attemptReconnect(token: string) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    
    console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    this.notifyStatus('reconnecting');

    setTimeout(() => this.connect(token), delay);
  }

  disconnect() {
    this.stopPing();
    if (this.ws) {
      this.ws.close(1000, 'User disconnected');
      this.ws = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts; // Prevent auto-reconnect
  }

  send(message: any) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected, message not sent');
    }
  }

  sendMessage(chatId: string, content: string, type: string = 'text') {
    this.send({
      type: 'message.send',
      payload: { chatId, content, type },
    });
  }

  sendTypingIndicator(chatId: string) {
    this.send({
      type: 'chat.typing',
      payload: { chatId },
    });
  }

  onMessage(handler: (data: any) => void) {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onStatusChange(handler: (status: 'connected' | 'disconnected' | 'reconnecting') => void) {
    this.statusHandlers.add(handler);
    return () => this.statusHandlers.delete(handler);
  }

  private notifyMessage(data: any) {
    this.messageHandlers.forEach(handler => handler(data));
  }

  private notifyStatus(status: 'connected' | 'disconnected' | 'reconnecting') {
    this.statusHandlers.forEach(handler => handler(status));
  }

  private startPing() {
    this.pingInterval = window.setInterval(() => {
      this.send({ type: 'ping', timestamp: Date.now() });
    }, 30000); // Ping every 30 seconds
  }

  private stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

export const wsService = new WebSocketService();
