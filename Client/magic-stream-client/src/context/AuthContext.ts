import { createContext, type Dispatch, type SetStateAction } from "react";
import type { User } from "../types/user";

interface AuthContextValue {
  auth: User | null;
  setAuth: Dispatch<SetStateAction<User | null>>;
  // True while the initial session restore (POST /refresh) is in progress
  loading: boolean;
}

export const AuthContext = createContext<AuthContextValue>({
  auth: null,
  setAuth: () => {},
  loading: true,
});
