'use client';

import React, { createContext, useContext, useState, useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { getMe, login as apiLogin, logout as apiLogout } from './api';

const AuthContext = createContext({
  isAuthenticated: false,
  role: null,
  isLoading: true,
  login: async () => {},
  signup: async () => {},
  logout: async () => {},
});

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [role, setRole] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    const checkAuth = async () => {
      try {
        const data = await getMe();
        if (data && data.role) {
          setIsAuthenticated(true);
          setRole(data.role);
          if (pathname === '/login') {
            router.push('/');
          }
        }
      } catch (error) {
        setIsAuthenticated(false);
        setRole(null);
        if (pathname !== '/login') {
          router.push('/login');
        }
      } finally {
        setIsLoading(false);
      }
    };

    checkAuth();
  }, [pathname, router]);

  const login = async (email, password) => {
    setIsLoading(true);
    try {
      const data = await apiLogin(email, password);
      setIsAuthenticated(true);
      setRole(data.role);
      router.push('/');
      return true;
    } catch (error) {
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const signup = async (email, password) => {
    setIsLoading(true);
    try {
      const data = await import('./api').then(m => m.signup(email, password));
      setIsAuthenticated(true);
      setRole(data.role);
      router.push('/');
      return true;
    } catch (error) {
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    setIsLoading(true);
    try {
      await apiLogout();
    } catch (error) {
      console.error('Logout failed:', error);
    } finally {
      setIsAuthenticated(false);
      setRole(null);
      router.push('/login');
      setIsLoading(false);
    }
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, role, isLoading, login, signup, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
