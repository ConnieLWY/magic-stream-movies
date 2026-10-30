import { useEffect, useState } from "react";
import Container from "react-bootstrap/Container";
import Spinner from "react-bootstrap/Spinner";
import useAxiosPrivate from "../../hook/useAxiosPrivate";
import Movies from "../movies/Movies";
import type { MovieModel } from "../../types/movie";

function Recommended() {
  const [movies, setMovies] = useState<MovieModel[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const axiosPrivate = useAxiosPrivate();

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

  return (
    <Container className="py-4">
      <h2 className="mb-1">Recommended for you</h2>
      <p className="text-muted mb-4">Picked from your favourite genres</p>

      {loading ? (
        <div className="d-flex justify-content-center my-5">
          <Spinner animation="border" role="status" />
        </div>
      ) : (
        <Movies movies={movies} message={message} />
      )}
    </Container>
  );
}

export default Recommended;
