import { useEffect, useRef, useState, type SubmitEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom';
import axios from 'axios';
import Container from 'react-bootstrap/Container'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'
import Form from 'react-bootstrap/Form'
import Button from 'react-bootstrap/Button'
import Badge from 'react-bootstrap/Badge'
import Spinner from 'react-bootstrap/Spinner'
import Alert from 'react-bootstrap/Alert'
import useAuth from '../../hook/useAuth';
import useAxiosPrivate from '../../hook/useAxiosPrivate';
import type { MovieModel } from '../../types/movie';

function Review() {
    const [movie, setMovie] = useState<MovieModel | null>(null);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const revText = useRef<HTMLTextAreaElement>(null);
    const { imdb_id } = useParams();
    const {auth,setAuth} = useAuth();
    const axiosPrivate = useAxiosPrivate();
    const navigate = useNavigate();

    // Only admins may update reviews (server enforces role === "ADMIN")
    const isAdmin = auth?.role === 'ADMIN';

    useEffect(() => {
        const fetchMovie = async () => {
            setLoading(true);
            try {
                const response = await axiosPrivate.get<MovieModel>(`/movie/${imdb_id}`);
                setMovie(response.data);
            } catch (error) {
                console.error('Error fetching movie:', error);
                setError('Error fetching movie');
            } finally {
                setLoading(false);
            }
        };

        fetchMovie();
        // axiosPrivate is recreated every render
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [imdb_id]);

    const handleSubmit = async (e: SubmitEvent<HTMLFormElement>) => {
        e.preventDefault();
        if (!movie || !revText.current) return;

        setSaving(true);
        setError(null);
        try {

            const response = await axiosPrivate.patch(`/updatereview/${imdb_id}`, { admin_review: revText.current.value });

            setMovie((prev) => prev && ({
                ...prev,
                admin_review: response.data?.admin_review ?? prev.admin_review,
                ranking: {
                    ranking_value: prev.ranking?.ranking_value ?? 0,
                    ranking_name: response.data?.ranking_name ?? prev.ranking?.ranking_name ?? ''
                }
            }));

        } catch (err) {
            console.error(err);
            if (axios.isAxiosError(err) && err.response?.status === 401) {
                 console.error('Unauthorized access - redirecting to login');
                 localStorage.removeItem('user');
                 setAuth(null);
                 navigate('/login');
            } else {
                setError('Error updating review');
            }

        } finally {
            setSaving(false);
        }
    };

  if (loading) {
    return (
        <div className="d-flex justify-content-center my-5">
            <Spinner animation="border" role="status" />
        </div>
    )
  }

  if (!movie) {
    return (
        <Container className="py-4">
            <Alert variant="secondary" className="text-center">{error ?? 'Movie not found'}</Alert>
        </Container>
    )
  }

  return (
    <Container className="py-4">
        <Row className="g-4">
            <Col md={4}>
                <img
                    src={movie.poster_path}
                    alt={movie.title}
                    className="img-fluid rounded shadow-sm w-100"
                    style={{ aspectRatio: '2 / 3', objectFit: 'cover' }}
                />
            </Col>

            <Col md={8}>
                <h2 className="mb-1">{movie.title}</h2>
                <p className="text-muted mb-3">{movie.imdb_id}</p>

                <div className="d-flex flex-wrap gap-2 mb-4">
                    {movie.ranking?.ranking_name && (
                        <Badge bg="warning" text="dark">{movie.ranking.ranking_name}</Badge>
                    )}
                    {movie.genre?.map((g) => (
                        <Badge key={g.genre_id} bg="secondary">{g.genre_name}</Badge>
                    ))}
                </div>

                <h5>Admin review</h5>

                {error && <Alert variant="danger" dismissible onClose={() => setError(null)}>{error}</Alert>}

                {isAdmin ? (
                    <Form onSubmit={handleSubmit}>
                        <Form.Group className="mb-3" controlId="adminReview">
                            <Form.Control
                                as="textarea"
                                rows={6}
                                ref={revText}
                                defaultValue={movie.admin_review}
                                placeholder="Write your review..."
                                required
                            />
                        </Form.Group>
                        <Button type="submit" disabled={saving}>
                            {saving ? (
                                <>
                                    <Spinner animation="border" size="sm" className="me-2" />
                                    Saving...
                                </>
                            ) : (
                                'Submit review'
                            )}
                        </Button>
                    </Form>
                ) : (
                    <p className="mb-0">{movie.admin_review || <span className="text-muted">No review yet.</span>}</p>
                )}
            </Col>
        </Row>
    </Container>
  )
}

export default Review
