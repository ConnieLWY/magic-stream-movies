import type { Genre } from './movie'

// Mirrors UserResponse in Server/MagicStreamMoviesServer/models/user_model.go
export interface User {
  user_id: string
  first_name: string
  last_name: string
  email: string
  role: string
  // Kept in memory only; the refresh token lives in an HttpOnly cookie
  access_token: string
  favourite_genres: Genre[]
}
