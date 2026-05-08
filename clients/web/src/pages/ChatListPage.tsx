import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { chatApi } from '@services/api';
import { useChatStore, Chat } from '@store/chatStore';
import { wsService } from '@services/websocket';
import { formatDistanceToNow } from 'date-fns';
import { ru } from 'date-fns/locale';

export function ChatListPage() {
  const navigate = useNavigate();
  const { chats, setChats, setActiveChat } = useChatStore();
  const [loading, setLoading] = useState(true);
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'reconnecting'>('disconnected');

  useEffect(() => {
    loadChats();
    
    // Подписка на обновления WebSocket
    const unsubscribeMessage = wsService.onMessage((data) => {
      if (data.type === 'message.new') {
        // Обновляем список чатов при новом сообщении
        loadChats();
      }
    });

    const unsubscribeStatus = wsService.onStatusChange(setConnectionStatus);

    return () => {
      unsubscribeMessage();
      unsubscribeStatus();
    };
  }, []);

  const loadChats = async () => {
    try {
      const data = await chatApi.getChats();
      setChats(data.chats || []);
    } catch (error) {
      console.error('Failed to load chats', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChatClick = (chat: Chat) => {
    setActiveChat(chat.id);
    navigate(`/chat/${chat.id}`);
  };

  const getStatusColor = (chat: Chat) => {
    if (chat.typing) return 'bg-green-500 animate-pulse';
    if (chat.isOnline) return 'bg-green-500';
    return 'bg-gray-400';
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow-sm sticky top-0 z-10">
        <div className="max-w-3xl mx-auto px-4 py-4 flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Чаты</h1>
          <div className="flex items-center space-x-2">
            <span className={`w-2 h-2 rounded-full ${getStatusColor({ typing: false, isOnline: connectionStatus === 'connected' } as Chat)}`} />
            <button 
              className="touch-target text-blue-500 hover:text-blue-600"
              aria-label="Новый чат"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            </button>
          </div>
        </div>
      </header>

      {/* Connection status banner */}
      {connectionStatus === 'reconnecting' && (
        <div className="bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 text-center py-2 text-sm">
          Переподключение...
        </div>
      )}
      {connectionStatus === 'disconnected' && (
        <div className="bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200 text-center py-2 text-sm">
          Нет соединения
        </div>
      )}

      {/* Chat list */}
      <main className="max-w-3xl mx-auto">
        {chats.length === 0 ? (
          <div className="text-center py-16 px-4">
            <svg className="mx-auto h-16 w-16 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
            </svg>
            <h3 className="mt-4 text-lg font-medium text-gray-900 dark:text-white">Нет чатов</h3>
            <p className="mt-2 text-gray-600 dark:text-gray-400">
              Начните новый диалог или создайте группу
            </p>
          </div>
        ) : (
          <ul className="divide-y divide-gray-200 dark:divide-gray-700">
            {chats.map((chat) => (
              <li key={chat.id}>
                <button
                  onClick={() => handleChatClick(chat)}
                  className="w-full hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors p-4 flex items-start space-x-3 touch-target"
                >
                  {/* Avatar */}
                  <div className="relative flex-shrink-0">
                    {chat.avatarUrl ? (
                      <img
                        src={chat.avatarUrl}
                        alt={chat.name || 'Avatar'}
                        className="w-12 h-12 rounded-full object-cover"
                      />
                    ) : (
                      <div className="w-12 h-12 rounded-full bg-blue-500 flex items-center justify-center text-white font-medium">
                        {(chat.name || '?').charAt(0).toUpperCase()}
                      </div>
                    )}
                    {chat.type === 'private' && (
                      <span className={`absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-white dark:border-gray-800 ${getStatusColor(chat)}`} />
                    )}
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0 text-left">
                    <div className="flex items-center justify-between mb-1">
                      <h3 className="text-base font-medium text-gray-900 dark:text-white truncate">
                        {chat.name || 'Без имени'}
                      </h3>
                      {chat.lastMessage && (
                        <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0 ml-2">
                          {formatDistanceToNow(chat.lastMessage.timestamp, { addSuffix: true, locale: ru })}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center justify-between">
                      <p className="text-sm text-gray-600 dark:text-gray-400 truncate">
                        {chat.typing ? (
                          <span className="text-green-500 italic">Печатает...</span>
                        ) : (
                          chat.lastMessage?.content || 'Нет сообщений'
                        )}
                      </p>
                      {chat.unreadCount > 0 && (
                        <span className="ml-2 inline-flex items-center justify-center w-5 h-5 rounded-full bg-blue-500 text-white text-xs font-medium flex-shrink-0">
                          {chat.unreadCount}
                        </span>
                      )}
                    </div>
                  </div>
                </button>
              </li>
            ))}
          </ul>
        )}
      </main>
    </div>
  );
}
