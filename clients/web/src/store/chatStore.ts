import { create } from 'zustand';

export interface Message {
  id: string;
  chatId: string;
  senderId: string;
  content: string;
  type: 'text' | 'image' | 'video' | 'audio' | 'file';
  timestamp: number;
  isRead: boolean;
  reactions?: Record<string, string[]>;
}

export interface Chat {
  id: string;
  type: 'private' | 'group' | 'channel';
  name?: string;
  avatarUrl?: string;
  lastMessage?: Message;
  unreadCount: number;
  participants: string[];
  isOnline?: boolean;
  typing?: boolean;
}

interface ChatState {
  chats: Chat[];
  messages: Record<string, Message[]>; // chatId -> messages
  activeChatId: string | null;
  setChats: (chats: Chat[]) => void;
  addMessage: (chatId: string, message: Message) => void;
  setActiveChat: (chatId: string | null) => void;
  markAsRead: (chatId: string) => void;
  updateTypingStatus: (chatId: string, isTyping: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  chats: [],
  messages: {},
  activeChatId: null,
  
  setChats: (chats) => set({ chats }),
  
  addMessage: (chatId, message) => set((state) => {
    const existingMessages = state.messages[chatId] || [];
    return {
      messages: {
        ...state.messages,
        [chatId]: [...existingMessages, message],
      },
      chats: state.chats.map(chat => 
        chat.id === chatId 
          ? { ...chat, lastMessage: message }
          : chat
      ).sort((a, b) => {
        const timeA = a.lastMessage?.timestamp || 0;
        const timeB = b.lastMessage?.timestamp || 0;
        return timeB - timeA;
      }),
    };
  }),
  
  setActiveChat: (chatId) => set({ activeChatId: chatId }),
  
  markAsRead: (chatId) => set((state) => ({
    chats: state.chats.map(chat =>
      chat.id === chatId ? { ...chat, unreadCount: 0 } : chat
    ),
  })),
  
  updateTypingStatus: (chatId, isTyping) => set((state) => ({
    chats: state.chats.map(chat =>
      chat.id === chatId ? { ...chat, typing: isTyping } : chat
    ),
  })),
}));
