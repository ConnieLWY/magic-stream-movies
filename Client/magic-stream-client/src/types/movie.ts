// Mirrors Server/MagicStreamMoviesServer/models/movie_model.go

export interface Genre {
  genre_id: number
  genre_name: string
}

export interface Ranking {
  ranking_value: number
  ranking_name: string
}

export interface MovieModel {
  id: string
  imdb_id: string
  title: string
  poster_path: string
  youtube_id?: string
  genre?: Genre[]
  admin_review?: string
  ranking?: Ranking
}
