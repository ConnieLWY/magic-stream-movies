import type { MovieModel } from "../../types/movie";
import { Link } from "react-router-dom";
import { PenIcon, PlayIcon } from "../brand/Icons";
import useAuth from "../../hook/useAuth";

interface MovieProps {
  movie: MovieModel;
  updateMovieReview?: (imdb_id: string) => void;
}

function Movie({ movie, updateMovieReview }: MovieProps) {
  const { auth } = useAuth();
  const isAdmin = auth?.role === "ADMIN";
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
          {/* Movie details are for signed-in users; only admins can edit the review */}
          {updateMovieReview && auth && (
            <button
              type="button"
              className="btn btn-link-subtle"
              onClick={() => updateMovieReview(movie.imdb_id)}
              aria-label={`${isAdmin ? "Edit" : "Read"} review: ${movie.title}`}
            >
              {isAdmin && <PenIcon size={13} />}
              {isAdmin ? "Edit review" : "Read review"}
            </button>
          )}
        </div>
      </div>
    </article>
  );
}

export default Movie;
