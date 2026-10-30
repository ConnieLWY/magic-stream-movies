import {
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { AuthContext } from "./AuthContext";
import type { User } from "../types/user";
import { refreshSession } from "../api/axiosConfig";

interface AuthProviderProps {
  children: ReactNode;
}

function AuthProvider({ children }: AuthProviderProps) {
  // Held in memory only (never localStorage), so the access token is gone on reload
  const [auth, setAuth] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // On page load, restore the session from the HttpOnly refresh_token cookie
  useEffect(() => {
    let active = true;

    refreshSession()
      .then((user) => {
        if (active) setAuth(user);
      })
      .catch(() => {
        if (active) setAuth(null);
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, []);

  return (
    <AuthContext.Provider value={{ auth, setAuth, loading }}>
      {children}
    </AuthContext.Provider>
  );
}

export default AuthProvider;
