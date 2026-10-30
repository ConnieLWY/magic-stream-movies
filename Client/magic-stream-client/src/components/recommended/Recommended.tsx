import { useEffect, useState } from "react";
import Container from "react-bootstrap/Container";
import useAxiosPrivate from "../../hook/useAxiosPrivate";
import useAuth from "../../hook/useAuth";
import Movies, { MovieGridSkeleton } from "../movies/Movies";
import { SparkIcon } from "../brand/Icons";
import type { MovieModel } from "../../types/movie";

function Recommended() {
  const [movies, setMovies] = useState<MovieModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const axiosPrivate = useAxiosPrivate();
  const { auth } = useAuth();

  useEffect(() => {
    const fetchRecommendedMovies = async () => {
      setLoading(true);
      setMessage("");

      try {
        const response = await axiosPrivate.get<MovieModel[]>("/recommendedmovies");
        setMovies(response.data ?? []);

        if (!response.data || response.data.length === 0) {
          setMessage("No recommendations yet. Try adding more favourite genres.");
        }
      } catch (error) {
        console.error("Error fetching recommended movies: ", error);
        setMessage("Error fetching recommended movies");
      } finally {
        setLoading(false);
      }
    };

    fetchRecommendedMovies();
  }, [axiosPrivate]);

  const favourites = auth?.favourite_genres ?? [];

  return (
    <Container className="section">
      <header className="page-head">
        <span className="eyebrow">
          <SparkIcon size={12} /> Curated for {auth?.first_name ?? "you"}
        </span>
        <h1>Picked for you</h1>
        <p>Conjured from the genres you love.</p>
        {favourites.length > 0 && (
          <div className="chip-row mt-3">
            {favourites.map((g) => (
              <span key={g.genre_id} className="pill">
                {g.genre_name}
              </span>
            ))}
          </div>
        )}
      </header>

      {loading ? <MovieGridSkeleton /> : <Movies movies={movies} message={message} />}
    </Container>
  );
}

export default Recommended;
