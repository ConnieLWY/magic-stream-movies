import { useEffect, useRef, useState, type SubmitEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom';
import axios from 'axios';
import Container from 'react-bootstrap/Container'
import Form from 'react-bootstrap/Form'
import Button from 'react-bootstrap/Button'
import Spinner from 'react-bootstrap/Spinner'
import Alert from 'react-bootstrap/Alert'
import useAuth from '../../hook/useAuth';
import useAxiosPrivate from '../../hook/useAxiosPrivate';
import type { MovieModel, ReviewUpdateResponse } from '../../types/movie';
import { ArrowLeftIcon, PlayIcon } from '../brand/Icons';

function Review() {
    const [movie, setMovie] = useState<MovieModel | null>(null);
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const revText = useRef<HTMLTextAreaElement>(null);
    const { imdb_id } = useParams();
    const { auth } = useAuth();
    const axiosPrivate = useAxiosPrivate();
    const navigate = useNavigate();

    // Only admins may update reviews (server enforces role === "ADMIN")
    const isAdmin = auth?.role === 'ADMIN';

    useEffect(() => {
        const fetchMovie = async () => {
            setLoading(true);
            try {
                const response = await axiosPrivate.get<MovieModel>(`/movies/${imdb_id}`);
                setMovie(response.data);
            } catch (error) {
                console.error('Error fetching movie:', error);
                setError('Error fetching movie');
            } finally {
                setLoading(false);
            }
        };

        fetchMovie();
    }, [axiosPrivate, imdb_id]);

    const handleSubmit = async (e: SubmitEvent<HTMLFormElement>) => {
        e.preventDefault();
        if (!movie || !revText.current) return;

        setSaving(true);
        setError(null);
        try {
            const response = await axiosPrivate.patch<ReviewUpdateResponse>(
                `/movies/${imdb_id}/review`,
                { admin_review: revText.current.value },
            );
            const { admin_review, ranking } = response.data;
            setMovie((prev) => prev && { ...prev, admin_review, ranking });
        } catch (err) {
            // 401 is handled by useAxiosPrivate (refresh, or sign out if that fails)
            if (axios.isAxiosError(err) && err.response?.status === 403) {
                setError('Only admins can update reviews.');
            } else if (axios.isAxiosError(err) && err.response?.status === 502) {
                setError('Could not rank this review automatically. Please try again.');
            } else {
                setError('Error updating review');
            }
        } finally {
            setSaving(false);
        }
    };

  if (loading) {
    return (
        <Container className="detail-grid">
            <div className="skeleton skeleton-poster" />
            <div>
                <div className="skeleton skeleton-line" style={{ width: '30%' }} />
                <div className="skeleton mt-3" style={{ height: '3rem', width: '70%' }} />
                <div className="skeleton mt-4" style={{ height: '10rem' }} />
            </div>
        </Container>
    )
  }

  if (!movie) {
    return (
        <Container className="section">
            <div className="empty-state">{error ?? 'Movie not found'}</div>
        </Container>
    )
  }

  return (
    <section className="detail-hero">
        <div className="hero-backdrop" style={{ backgroundImage: `url(${movie.poster_path})` }} />
        <Container>
            <div className="detail-grid">
                <img src={movie.poster_path} alt={movie.title} className="detail-poster" />

                <div>
                    <button type="button" className="btn btn-link-subtle" onClick={() => navigate(-1)}>
                        <ArrowLeftIcon size={14} /> Back
                    </button>
                    <h1 className="detail-title">{movie.title}</h1>

                    <div className="hero-meta">
                        {movie.ranking?.ranking_name && (
                            <span className="pill pill-rank">{movie.ranking.ranking_name}</span>
                        )}
                        {movie.genre?.map((g) => (
                            <span key={g.genre_id} className="pill">{g.genre_name}</span>
                        ))}
                        <span className="pill text-muted">{movie.imdb_id}</span>
                    </div>

                    {movie.youtube_id && (
                        <Link to={`/stream/${movie.youtube_id}`} className="btn btn-brand">
                            <PlayIcon size={16} /> Watch trailer
                        </Link>
                    )}

                    <div className="surface review-card">
                        <h2>Critic's review</h2>

                        {error && <Alert variant="danger" dismissible onClose={() => setError(null)}>{error}</Alert>}

                        {isAdmin ? (
                            <Form onSubmit={handleSubmit}>
                                <Form.Group className="mb-3" controlId="adminReview">
                                    <Form.Label className="visually-hidden">Review</Form.Label>
                                    <Form.Control
                                        as="textarea"
                                        rows={6}
                                        ref={revText}
                                        defaultValue={movie.admin_review}
                                        placeholder="Write your review..."
                                        required
                                    />
                                </Form.Group>
                                <Button type="submit" variant="brand" disabled={saving}>
                                    {saving ? (
                                        <>
                                            <Spinner animation="border" size="sm" />
                                            Saving...
                                        </>
                                    ) : (
                                        'Publish review'
                                    )}
                                </Button>
                            </Form>
                        ) : movie.admin_review ? (
                            <blockquote className="review-quote">{movie.admin_review}</blockquote>
                        ) : (
                            <p className="text-muted mb-0">No review yet.</p>
                        )}
                    </div>
                </div>
            </div>
        </Container>
    </section>
  )
}

export default Review
