import React, { createContext, useContext, ReactNode } from 'react';

// Create a generic context for authentication
interface AuthContextType {
  isAuthenticated: boolean;
  user: { name: string };
  getAccessTokenSilently: () => Promise<string>;
  login: () => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | null>(null);

// A simple provider that provides a fake authentication state
export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const isAuthenticated = true; // Always authenticated for development
  const user = { name: 'Dev User' };

  const getAccessTokenSilently = async () => {
    // Return a dummy JWT token
    return 'your-dummy-jwt-token';
  };

  const login = () => {
    console.log('Login called (mock)');
  };

  const logout = () => {
    console.log('Logout called (mock)');
  };

  const authValue = {
    isAuthenticated,
    user,
    getAccessTokenSilently,
    login,
    logout,
  };

  return (
    <AuthContext.Provider value={authValue}>
      {children}
    </AuthContext.Provider>
  );
};

// Generic hook to access the authentication context
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === null) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};