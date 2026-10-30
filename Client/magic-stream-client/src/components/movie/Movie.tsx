import Card from 'react-bootstrap/Card'
import Badge from 'react-bootstrap/Badge'
import Button from 'react-bootstrap/Button'
import type { MovieModel } from '../../types/movie'

interface MovieProps {
  movie: MovieModel
  updateMovieReview?: (imdb_id: string) => void
}

function Movie({ movie, updateMovieReview }: MovieProps) {
  return (
    <Card className="h-100 shadow-sm">
      <Card.Img
        variant="top"
        src={movie.poster_path}
        alt={movie.title}
        style={{ aspectRatio: '2 / 3', objectFit: 'cover' }}
      />
      <Card.Body className="d-flex flex-column">
        <Card.Title className="fs-6">{movie.title}</Card.Title>
        <Card.Text className="text-muted small mb-2">{movie.imdb_id}</Card.Text>
        <div className="mt-auto d-flex align-items-center justify-content-between gap-2">
          {movie.ranking?.ranking_name && (
            <Badge bg="warning" text="dark">
              {movie.ranking.ranking_name}
            </Badge>
          )}
          {updateMovieReview && (
            <Button
              variant="outline-primary"
              size="sm"
              className="ms-auto"
              onClick={() => updateMovieReview(movie.imdb_id)}
            >
              Review
            </Button>
          )}
        </div>
      </Card.Body>
    </Card>
  )
}

export default Movie
