import { useEffect, useState, type SubmitEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom';
import axios from 'axios';
import Form from 'react-bootstrap/Form'
import Row from 'react-bootstrap/Row'
import Col from 'react-bootstrap/Col'
import Button from 'react-bootstrap/Button'
import Alert from 'react-bootstrap/Alert'
import Spinner from 'react-bootstrap/Spinner'
import { axiosClient } from '../../api/axiosConfig';
import type { Genre } from '../../types/movie';
import type { User } from '../../types/user';
import useAuth from '../../hook/useAuth';
import AuthShell from '../auth/AuthShell';

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
    const { setAuth } = useAuth();

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

        // Mirror the server-side validation in models.UserRegister
        if (firstName.trim().length < 2 || lastName.trim().length < 2) {
            setError("First and last name must be at least 2 characters");
            return;
        }
        if (password.length < 8) {
            setError("Password must be at least 8 characters");
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
            // Registering also signs the user in: the response matches /auth/login
            const response = await axiosClient.post<User>('/auth/register', {
                first_name: firstName.trim(),
                last_name: lastName.trim(),
                email: email.trim(),
                password,
                favourite_genre_ids: favouriteGenres.map((g) => g.genre_id),
            });
            setAuth(response.data);
            navigate('/', { replace: true });
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
    <AuthShell wide>
        <h1>Create your account</h1>
        <p className="lede">Tell us what you love and we'll fill your screen with it.</p>

        {error && <Alert variant="danger">{error}</Alert>}

        <Form onSubmit={handleSubmit}>
            <Row className="g-3 mb-3">
                <Col sm={6}>
                    <Form.Group controlId="firstName">
                        <Form.Label>First name</Form.Label>
                        <Form.Control
                            value={firstName}
                            onChange={(e) => setFirstName(e.target.value)}
                            autoComplete="given-name"
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
                            autoComplete="family-name"
                            required
                        />
                    </Form.Group>
                </Col>
            </Row>

            <Form.Group className="mb-3" controlId="email">
                <Form.Label>Email</Form.Label>
                <Form.Control
                    type="email"
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    autoComplete="email"
                    required
                />
            </Form.Group>

            <Row className="g-3 mb-4">
                <Col sm={6}>
                    <Form.Group controlId="password">
                        <Form.Label>Password</Form.Label>
                        <Form.Control
                            type="password"
                            placeholder="At least 8 characters"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            autoComplete="new-password"
                            minLength={8}
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

            <fieldset className="mb-4">
                <legend className="form-label">
                    Favourite genres{' '}
                    <span className="text-muted fw-normal">({favouriteGenres.length} selected)</span>
                </legend>
                <div className="chip-row">
                    {genres.map((genre) => (
                        <button
                            key={genre.genre_id}
                            type="button"
                            className="chip"
                            aria-pressed={favouriteGenres.some((g) => g.genre_id === genre.genre_id)}
                            onClick={() => toggleGenre(genre)}
                        >
                            {genre.genre_name}
                        </button>
                    ))}
                </div>
            </fieldset>

            <Button type="submit" variant="brand" size="lg" className="w-100" disabled={loading}>
                {loading ? <Spinner animation="border" size="sm" /> : 'Create account'}
            </Button>
        </Form>

        <p className="auth-switch">
            Already a member? <Link to="/login">Sign in</Link>
        </p>
    </AuthShell>
  )
}

export default Register
