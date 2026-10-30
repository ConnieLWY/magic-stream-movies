import type { MovieModel } from "../../types/movie";
import { Link } from "react-router-dom";
import { PenIcon, PlayIcon } from "../brand/Icons";

interface MovieProps {
  movie: MovieModel;
  updateMovieReview?: (imdb_id: string) => void;
}

function Movie({ movie, updateMovieReview }: MovieProps) {
  const genres = movie.genre?.map((g) => g.genre_name).join(" · ");

  const poster = (
    <>
      <img src={movie.poster_path} alt="" loading="lazy" />
      {movie.ranking?.ranking_name && (
        <span className="pill pill-rank">{movie.ranking.ranking_name}</span>
      )}
      {movie.youtube_id && (
        <span className="movie-poster-overlay">
          <span className="play-orb">
            <PlayIcon size={22} />
          </span>
        </span>
      )}
    </>
  );

  return (
    <article className="movie-card">
      {movie.youtube_id ? (
        <Link
          to={`/stream/${movie.youtube_id}`}
          className="movie-poster"
          aria-label={`Play trailer: ${movie.title}`}
        >
          {poster}
        </Link>
      ) : (
        <div className="movie-poster">{poster}</div>
      )}

      <div className="movie-info">
        <h3 className="movie-title">{movie.title}</h3>
        <div className="movie-sub">
          <span>{genres || movie.imdb_id}</span>
          {updateMovieReview && (
            <button
              type="button"
              className="btn btn-link-subtle"
              onClick={() => updateMovieReview(movie.imdb_id)}
              aria-label={`Review ${movie.title}`}
            >
              <PenIcon size={13} />
              Review
            </button>
          )}
        </div>
      </div>
    </article>
  );
}

export default Movie;
