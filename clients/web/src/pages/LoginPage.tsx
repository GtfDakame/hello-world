import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { authApi } from '@services/api';
import { useAuthStore } from '@store/authStore';
import { wsService } from '@services/websocket';

export function LoginPage() {
  const [step, setStep] = useState<'phone' | 'otp'>('phone');
  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const login = useAuthStore(state => state.login);
  const navigate = useNavigate();

  const handleRequestOtp = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    
    try {
      await authApi.requestOtp(phone);
      setStep('otp');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Ошибка отправки кода');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyOtp = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);
    
    try {
      const response = await authApi.verifyOtp(phone, otp);
      const { token, refreshToken, user } = response;
      
      login(token, refreshToken, user);
      
      // Подключаем WebSocket
      wsService.connect(token);
      
      navigate('/chats');
    } catch (err: any) {
      setError(err.response?.data?.message || 'Неверный код');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="max-w-md w-full space-y-8 animate-fade-in">
        <div className="text-center">
          <h1 className="text-4xl font-bold text-gray-900 dark:text-white mb-2">
            Messenger
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Мессенджер нового поколения
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-lg p-8">
          {step === 'phone' ? (
            <form onSubmit={handleRequestOtp} className="space-y-6">
              <div>
                <label htmlFor="phone" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Номер телефона
                </label>
                <input
                  id="phone"
                  type="tel"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                  placeholder="+7 (999) 000-00-00"
                  className="w-full px-4 py-3 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all touch-target"
                  required
                  pattern="^\+?[0-9\s\-()]+$"
                />
              </div>

              {error && (
                <div className="text-red-500 text-sm text-center" role="alert">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full bg-blue-500 hover:bg-blue-600 disabled:bg-blue-300 text-white font-medium py-3 rounded-xl transition-colors touch-target"
              >
                {loading ? 'Отправка...' : 'Получить код'}
              </button>
            </form>
          ) : (
            <form onSubmit={handleVerifyOtp} className="space-y-6">
              <div>
                <label htmlFor="otp" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Код подтверждения
                </label>
                <input
                  id="otp"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  value={otp}
                  onChange={(e) => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="000000"
                  className="w-full px-4 py-3 text-center text-2xl tracking-widest rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all touch-target"
                  required
                  maxLength={6}
                />
              </div>

              {error && (
                <div className="text-red-500 text-sm text-center" role="alert">
                  {error}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="w-full bg-blue-500 hover:bg-blue-600 disabled:bg-blue-300 text-white font-medium py-3 rounded-xl transition-colors touch-target"
              >
                {loading ? 'Проверка...' : 'Войти'}
              </button>

              <button
                type="button"
                onClick={() => setStep('phone')}
                className="w-full text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white text-sm transition-colors touch-target"
              >
                Изменить номер
              </button>
            </form>
          )}
        </div>

        <p className="text-center text-xs text-gray-500 dark:text-gray-500">
          Продолжая, вы соглашаетесь с условиями использования и политикой конфиденциальности
        </p>
      </div>
    </div>
  );
}
