import type { Genre } from './movie'

// Mirrors UserResponse in Server/MagicStreamMoviesServer/models/user_model.go
export interface User {
  user_id: string
  first_name: string
  last_name: string
  email: string
  role: string
  token: string
  refresh_token: string
  favourite_genres: Genre[]
}
