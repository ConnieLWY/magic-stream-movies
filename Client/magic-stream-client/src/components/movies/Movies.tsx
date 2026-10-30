import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'
import Alert from 'react-bootstrap/Alert'
import Movie from '../movie/Movie'
import type { MovieModel } from '../../types/movie'

interface MoviesProps {
  movies: MovieModel[]
  message?: string
  updateMovieReview?: (imdb_id: string) => void
}

function Movies({ movies, message, updateMovieReview }: MoviesProps) {
  if (!movies || movies.length === 0) {
    return (
      <Alert variant="secondary" className="text-center my-4">
        {message || 'No movies found.'}
      </Alert>
    )
  }

  return (
    <Row xs={2} sm={3} md={4} lg={5} className="g-4">
      {movies.map((movie) => (
        <Col key={movie.imdb_id}>
          <Movie movie={movie} updateMovieReview={updateMovieReview} />
        </Col>
      ))}
    </Row>
  )
}

export default Movies
