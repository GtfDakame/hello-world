import { Routes, Route, Navigate } from 'react-router-dom';
import { LoginPage } from '@pages/LoginPage';
import { ChatListPage } from '@pages/ChatListPage';
import { ChatPage } from '@pages/ChatPage';
import { useAuthStore } from '@store/authStore';

export default function App() {
  const isAuthenticated = useAuthStore(state => state.isAuthenticated);

  return (
    <Routes>
      <Route 
        path="/login" 
        element={!isAuthenticated ? <LoginPage /> : <Navigate to="/chats" />} 
      />
      <Route 
        path="/chats" 
        element={isAuthenticated ? <ChatListPage /> : <Navigate to="/login" />} 
      />
      <Route 
        path="/chat/:id" 
        element={isAuthenticated ? <ChatPage /> : <Navigate to="/login" />} 
      />
      <Route 
        path="/" 
        element={<Navigate to={isAuthenticated ? "/chats" : "/login"} />} 
      />
    </Routes>
  );
}
