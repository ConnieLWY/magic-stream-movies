import Movie from "../movie/Movie";
import { SparkIcon } from "../brand/Icons";
import type { MovieModel } from "../../types/movie";

interface MoviesProps {
  movies: MovieModel[];
  message?: string;
  updateMovieReview?: (imdb_id: string) => void;
}

function Movies({ movies, message, updateMovieReview }: MoviesProps) {
  if (!movies || movies.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-icon">
          <SparkIcon size={28} />
        </div>
        <p className="mb-0">{message || "No movies found."}</p>
      </div>
    );
  }

  return (
    <div className="movie-grid">
      {movies.map((movie) => (
        <Movie
          key={movie.id}
          movie={movie}
          updateMovieReview={updateMovieReview}
        />
      ))}
    </div>
  );
}

export function MovieGridSkeleton({ count = 10 }: { count?: number }) {
  return (
    <div className="movie-grid" aria-busy="true" aria-label="Loading movies">
      {Array.from({ length: count }, (_, i) => (
        <div key={i}>
          <div className="skeleton skeleton-poster" />
          <div className="skeleton skeleton-line" style={{ width: "80%" }} />
          <div className="skeleton skeleton-line" style={{ width: "50%" }} />
        </div>
      ))}
    </div>
  );
}

export default Movies;
