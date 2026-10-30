import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import Container from "react-bootstrap/Container";
import { axiosClient } from "../../api/axiosConfig";
import Movies, { MovieGridSkeleton } from "../movies/Movies";
import { PenIcon, PlayIcon, SparkIcon } from "../brand/Icons";
import { BRAND } from "../../brand";
import type { Genre, MovieModel } from "../../types/movie";

interface HomeProps {
  updateMovieReview?: (imdb_id: string) => void;
}

function Home({ updateMovieReview }: HomeProps) {
  const [movies, setMovies] = useState<MovieModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [activeGenre, setActiveGenre] = useState<number | null>(null);

  useEffect(() => {
    const fetchMovies = async () => {
      setLoading(true);
      setMessage("");
      try {
        const response = await axiosClient.get<MovieModel[]>("/movies");
        setMovies(response.data ?? []);

        if (!response.data || response.data.length === 0) {
          setMessage("There are currently no movies available");
        }
      } catch (error) {
        console.error("Error fetching movies: ", error);
        setMessage("Error fetching movies");
      } finally {
        setLoading(false);
      }
    };

    fetchMovies();
  }, []);

  // Prefer a title with a trailer for the hero slot
  const featured = useMemo(
    () => movies.find((m) => m.youtube_id) ?? movies[0],
    [movies],
  );

  const genres = useMemo(() => {
    const byId = new Map<number, Genre>();
    movies.forEach((m) => m.genre?.forEach((g) => byId.set(g.genre_id, g)));
    return [...byId.values()].sort((a, b) =>
      a.genre_name.localeCompare(b.genre_name),
    );
  }, [movies]);

  const visibleMovies =
    activeGenre === null
      ? movies
      : movies.filter((m) => m.genre?.some((g) => g.genre_id === activeGenre));

  return (
    <>
      {featured && !loading && (
        <section className="hero">
          <div
            className="hero-backdrop"
            style={{ backgroundImage: `url(${featured.poster_path})` }}
          />
          <Container className="hero-inner">
            <div>
              <span className="eyebrow">
                <SparkIcon size={12} /> Featured tonight
              </span>
              <h1 className="hero-title">{featured.title}</h1>
              <div className="hero-meta">
                {featured.ranking?.ranking_name && (
                  <span className="pill pill-rank">
                    {featured.ranking.ranking_name}
                  </span>
                )}
                {featured.genre?.map((g) => (
                  <span key={g.genre_id} className="pill">
                    {g.genre_name}
                  </span>
                ))}
              </div>
              <p className="hero-review">
                {featured.admin_review || BRAND.description}
              </p>
              <div className="hero-actions">
                {featured.youtube_id && (
                  <Link
                    to={`/stream/${featured.youtube_id}`}
                    className="btn btn-brand btn-lg"
                  >
                    <PlayIcon size={18} /> Watch trailer
                  </Link>
                )}
                {updateMovieReview && (
                  <button
                    type="button"
                    className="btn btn-ghost btn-lg"
                    onClick={() => updateMovieReview(featured.imdb_id)}
                  >
                    <PenIcon size={16} /> Read review
                  </button>
                )}
              </div>
            </div>
            <img
              className="hero-poster"
              src={featured.poster_path}
              alt={featured.title}
            />
          </Container>
        </section>
      )}

      <Container className="section">
        <div className="section-head">
          <div>
            <span className="eyebrow">Library</span>
            <h2>Discover films</h2>
          </div>
          {genres.length > 0 && (
            <div className="chip-row" role="group" aria-label="Filter by genre">
              <button
                type="button"
                className="chip"
                aria-pressed={activeGenre === null}
                onClick={() => setActiveGenre(null)}
              >
                All
              </button>
              {genres.map((g) => (
                <button
                  key={g.genre_id}
                  type="button"
                  className="chip"
                  aria-pressed={activeGenre === g.genre_id}
                  onClick={() => setActiveGenre(g.genre_id)}
                >
                  {g.genre_name}
                </button>
              ))}
            </div>
          )}
        </div>

        {loading ? (
          <MovieGridSkeleton />
        ) : (
          <Movies
            movies={visibleMovies}
            message={message || "No films in this genre yet."}
            updateMovieReview={updateMovieReview}
          />
        )}
      </Container>
    </>
  );
}

export default Home;
