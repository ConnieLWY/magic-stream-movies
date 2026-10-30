import { useEffect, useState, type SubmitEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom';
import axios from 'axios';
import Container from 'react-bootstrap/Container'
import Card from 'react-bootstrap/Card'
import Form from 'react-bootstrap/Form'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'
import Button from 'react-bootstrap/Button'
import Alert from 'react-bootstrap/Alert'
import Spinner from 'react-bootstrap/Spinner'
import { axiosClient } from '../../api/axiosConfig';
import type { Genre } from '../../types/movie';

function Register() {
    const [firstName, setFirstName] = useState('');
    const [lastName, setLastName] = useState('');
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [favouriteGenres, setFavouriteGenres] = useState<Genre[]>([]);
    const [genres, setGenres] = useState<Genre[]>([]);

    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const navigate = useNavigate();

    useEffect(() => {
        const fetchGenres = async () => {
            try {
                const response = await axiosClient.get<Genre[]>('/genres');
                setGenres(response.data ?? []);
            } catch (err) {
                console.error("Error fetching genres: ", err);
                setError("Unable to load genres");
            }
        }

        fetchGenres();
    }, [])

    const toggleGenre = (genre: Genre) => {
        setFavouriteGenres((prev) =>
            prev.some((g) => g.genre_id === genre.genre_id)
                ? prev.filter((g) => g.genre_id !== genre.genre_id)
                : [...prev, genre]
        );
    }

    const handleSubmit = async (e: SubmitEvent<HTMLFormElement>) => {
        e.preventDefault();
        setError(null);

        // Mirror the server-side validation in models.User
        if (firstName.trim().length < 2 || lastName.trim().length < 2) {
            setError("First and last name must be at least 2 characters");
            return;
        }
        if (password.length < 6) {
            setError("Password must be at least 6 characters");
            return;
        }
        if (password !== confirmPassword) {
            setError("Passwords do not match");
            return;
        }
        if (favouriteGenres.length === 0) {
            setError("Please select at least one favourite genre");
            return;
        }

        setLoading(true);
        try {
            await axiosClient.post('/register', {
                first_name: firstName.trim(),
                last_name: lastName.trim(),
                email: email.trim(),
                password,
                role: 'USER',
                favourite_genres: favouriteGenres,
            });
            navigate('/login', { replace: true });
        } catch (err) {
            if (axios.isAxiosError(err) && err.response?.data?.error) {
                setError(err.response.data.error);
            } else {
                setError("Registration failed. Please try again.");
            }
        } finally {
            setLoading(false);
        }
    }

  return (
    <Container className="py-5" style={{ maxWidth: 560 }}>
        <Card className="shadow-sm">
            <Card.Body className="p-4">
                <h2 className="mb-4 text-center">Create an account</h2>

                {error && <Alert variant="danger">{error}</Alert>}

                <Form onSubmit={handleSubmit}>
                    <Row className="g-3 mb-3">
                        <Col sm={6}>
                            <Form.Group controlId="firstName">
                                <Form.Label>First name</Form.Label>
                                <Form.Control
                                    value={firstName}
                                    onChange={(e) => setFirstName(e.target.value)}
                                    required
                                />
                            </Form.Group>
                        </Col>
                        <Col sm={6}>
                            <Form.Group controlId="lastName">
                                <Form.Label>Last name</Form.Label>
                                <Form.Control
                                    value={lastName}
                                    onChange={(e) => setLastName(e.target.value)}
                                    required
                                />
                            </Form.Group>
                        </Col>
                    </Row>

                    <Form.Group className="mb-3" controlId="email">
                        <Form.Label>Email</Form.Label>
                        <Form.Control
                            type="email"
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            autoComplete="email"
                            required
                        />
                    </Form.Group>

                    <Row className="g-3 mb-3">
                        <Col sm={6}>
                            <Form.Group controlId="password">
                                <Form.Label>Password</Form.Label>
                                <Form.Control
                                    type="password"
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    autoComplete="new-password"
                                    minLength={6}
                                    required
                                />
                            </Form.Group>
                        </Col>
                        <Col sm={6}>
                            <Form.Group controlId="confirmPassword">
                                <Form.Label>Confirm password</Form.Label>
                                <Form.Control
                                    type="password"
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    autoComplete="new-password"
                                    isInvalid={confirmPassword !== '' && password !== confirmPassword}
                                    required
                                />
                                <Form.Control.Feedback type="invalid">
                                    Passwords do not match
                                </Form.Control.Feedback>
                            </Form.Group>
                        </Col>
                    </Row>

                    <Form.Group className="mb-4">
                        <Form.Label>Favourite genres</Form.Label>
                        <div className="d-flex flex-wrap gap-3">
                            {genres.map((genre) => (
                                <Form.Check
                                    key={genre.genre_id}
                                    id={`genre-${genre.genre_id}`}
                                    type="checkbox"
                                    label={genre.genre_name}
                                    checked={favouriteGenres.some((g) => g.genre_id === genre.genre_id)}
                                    onChange={() => toggleGenre(genre)}
                                />
                            ))}
                        </div>
                    </Form.Group>

                    <Button type="submit" className="w-100" disabled={loading}>
                        {loading ? <Spinner animation="border" size="sm" /> : 'Register'}
                    </Button>
                </Form>

                <p className="text-center mt-3 mb-0">
                    Already have an account? <Link to="/login">Login</Link>
                </p>
            </Card.Body>
        </Card>
    </Container>
  )
}

export default Register
