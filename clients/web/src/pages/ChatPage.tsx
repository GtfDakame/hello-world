import { useEffect, useState, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { messageApi, chatApi } from '@services/api';
import { useChatStore, Message } from '@store/chatStore';
import { wsService } from '@services/websocket';
import { format } from 'date-fns';
import { ru } from 'date-fns/locale';

export function ChatPage() {
  const { id: chatId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const messagesEndRef = useRef<HTMLDivElement>(null);
  
  const { activeChatId, setActiveChat, messages, addMessage, markAsRead, updateTypingStatus } = useChatStore();
  const [inputValue, setInputValue] = useState('');
  const [loading, setLoading] = useState(true);
  const [chat, setChat] = useState<any>(null);
  const [isTyping, setIsTyping] = useState(false);
  const typingTimeoutRef = useRef<number | null>(null);

  const chatMessages = chatId ? messages[chatId] || [] : [];

  useEffect(() => {
    if (chatId) {
      loadChat();
      loadMessages();
      setActiveChat(chatId);
      markAsRead(chatId);
    }

    const unsubscribeMessage = wsService.onMessage((data) => {
      if (data.type === 'message.new' && data.payload.chatId === chatId) {
        addMessage(chatId!, data.payload.message);
        markAsRead(chatId!);
      }
      if (data.type === 'chat.typing' && data.payload.chatId === chatId) {
        updateTypingStatus(chatId!, data.payload.isTyping);
      }
    });

    return () => {
      unsubscribeMessage();
      if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    };
  }, [chatId]);

  useEffect(() => {
    scrollToBottom();
  }, [chatMessages]);

  const loadChat = async () => {
    if (!chatId) return;
    try {
      const data = await chatApi.getChat(chatId);
      setChat(data);
    } catch (error) {
      console.error('Failed to load chat', error);
    }
  };

  const loadMessages = async () => {
    if (!chatId) return;
    try {
      const data = await messageApi.getMessages(chatId);
      data.messages.forEach((msg: Message) => addMessage(chatId, msg));
    } catch (error) {
      console.error('Failed to load messages', error);
    } finally {
      setLoading(false);
    }
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const handleSendMessage = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || !chatId) return;

    const content = inputValue.trim();
    setInputValue('');

    try {
      await messageApi.sendMessage(chatId, content);
      // Сообщение придет через WebSocket
    } catch (error) {
      console.error('Failed to send message', error);
    }
  };

  const handleTyping = () => {
    if (!chatId) return;

    if (!isTyping) {
      setIsTyping(true);
      wsService.sendTypingIndicator(chatId);
    }

    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    
    typingTimeoutRef.current = window.setTimeout(() => {
      setIsTyping(false);
    }, 2000);
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow-sm sticky top-0 z-10">
        <div className="max-w-3xl mx-auto px-4 py-3 flex items-center space-x-3">
          <button
            onClick={() => navigate('/chats')}
            className="touch-target text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
            aria-label="Назад"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          
          <div className="flex-1 min-w-0">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white truncate">
              {chat?.name || 'Чат'}
            </h2>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {chat?.typing ? (
                <span className="text-green-500">Печатает...</span>
              ) : chat?.isOnline ? (
                'В сети'
              ) : (
                'Был(а) недавно'
              )}
            </p>
          </div>
        </div>
      </header>

      {/* Messages */}
      <main className="flex-1 overflow-y-auto p-4 space-y-3">
        {chatMessages.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-gray-500 dark:text-gray-400">
              Начните диалог, отправив первое сообщение
            </p>
          </div>
        ) : (
          chatMessages.map((message) => (
            <MessageBubble key={message.id} message={message} />
          ))
        )}
        <div ref={messagesEndRef} />
      </main>

      {/* Input */}
      <footer className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4">
        <form onSubmit={handleSendMessage} className="max-w-3xl mx-auto flex items-end space-x-2">
          <textarea
            value={inputValue}
            onChange={(e) => {
              setInputValue(e.target.value);
              handleTyping();
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSendMessage(e);
              }
            }}
            placeholder="Сообщение..."
            rows={1}
            className="flex-1 resize-none px-4 py-3 rounded-xl border border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all touch-target max-h-32"
            style={{ minHeight: '44px' }}
          />
          <button
            type="submit"
            disabled={!inputValue.trim()}
            className="touch-target bg-blue-500 hover:bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-gray-600 text-white rounded-xl p-3 transition-colors"
            aria-label="Отправить"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          </button>
        </form>
      </footer>
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const isOwn = message.senderId === 'me'; // TODO: заменить на реальный ID пользователя
  
  return (
    <div className={`flex ${isOwn ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`message-bubble ${isOwn ? 'sent' : 'received'} animate-slide-up`}
      >
        <p className="text-sm whitespace-pre-wrap break-words">{message.content}</p>
        <div className={`text-xs mt-1 ${isOwn ? 'text-blue-100' : 'text-gray-500'}`}>
          {format(message.timestamp, 'HH:mm', { locale: ru })}
        </div>
      </div>
    </div>
  );
}
