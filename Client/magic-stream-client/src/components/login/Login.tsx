import { Link, useLocation, useNavigate } from "react-router-dom";
import Form from "react-bootstrap/Form";
import Button from "react-bootstrap/Button";
import Alert from "react-bootstrap/Alert";
import Spinner from "react-bootstrap/Spinner";
import { useState, type SubmitEvent } from "react";
import axios from "axios";
import { axiosClient } from "../../api/axiosConfig";
import useAuth from "../../hook/useAuth";
import { BRAND } from "../../brand";
import AuthShell from "../auth/AuthShell";
import type { User } from "../../types/user";

function Login() {
  const { setAuth } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

  const from = location.state?.from?.pathname || "/";

  const handleSubmit = async (e: SubmitEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await axiosClient.post<User>("/auth/login", { email, password });
      // Access token stays in memory (auth state); refresh token was set as an HttpOnly cookie
      setAuth(response.data);
      navigate(from, { replace: true });
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 429) {
        setError("Too many attempts. Please wait a minute and try again.");
      } else {
        setError("Invalid email or password");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell>
      <h1>Welcome back</h1>
      <p className="lede">Sign in to pick up where the credits left off.</p>

      {error && (
        <Alert variant="danger" dismissible onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Form onSubmit={handleSubmit}>
        <Form.Group className="mb-3" controlId="email">
          <Form.Label>Email</Form.Label>
          <Form.Control
            type="email"
            placeholder="you@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            autoFocus
            required
          />
        </Form.Group>

        <Form.Group className="mb-4" controlId="password">
          <Form.Label>Password</Form.Label>
          <Form.Control
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </Form.Group>

        <Button type="submit" variant="brand" size="lg" className="w-100" disabled={loading}>
          {loading ? (
            <>
              <Spinner animation="border" size="sm" />
              Signing in...
            </>
          ) : (
            "Sign in"
          )}
        </Button>
      </Form>

      <p className="auth-switch">
        New to {BRAND.name}? <Link to="/register">Create an account</Link>
      </p>
    </AuthShell>
  );
}

export default Login;
