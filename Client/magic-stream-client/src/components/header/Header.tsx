import Container from "react-bootstrap/Container";
import Nav from "react-bootstrap/Nav";
import Navbar from "react-bootstrap/Navbar";
import Button from "react-bootstrap/Button";
import { Link, NavLink } from "react-router-dom";
import useAuth from "../../hook/useAuth";
import Logo from "../brand/Logo";

interface Props {
  handleLogout: () => void;
}

function Header({ handleLogout }: Props) {
  const { auth } = useAuth();

  return (
    <header className="site-header">
      <Navbar expand="md" variant="dark" collapseOnSelect>
        <Container>
          <Navbar.Brand as={Link} to="/" aria-label="Home">
            <Logo size={30} />
          </Navbar.Brand>
          <Navbar.Toggle aria-controls="main-navbar" />
          <Navbar.Collapse id="main-navbar">
            <Nav className="me-auto gap-md-1">
              <Nav.Link as={NavLink} to="/" end eventKey="home">
                Discover
              </Nav.Link>
              <Nav.Link as={NavLink} to="/recommended" eventKey="recommended">
                For you
              </Nav.Link>
            </Nav>
            <Nav className="header-actions align-items-md-center gap-2 flex-row">
              {auth ? (
                <>
                  <span className="user-chip me-2">
                    <span className="avatar" aria-hidden="true">
                      {auth.first_name?.charAt(0).toUpperCase()}
                    </span>
                    {auth.first_name}
                  </span>
                  <Button variant="ghost" size="sm" onClick={handleLogout}>
                    Sign out
                  </Button>
                </>
              ) : (
                <>
                  <Link to="/login" className="btn btn-ghost btn-sm">
                    Sign in
                  </Link>
                  <Link to="/register" className="btn btn-brand btn-sm">
                    Join free
                  </Link>
                </>
              )}
            </Nav>
          </Navbar.Collapse>
        </Container>
      </Navbar>
    </header>
  );
}

export default Header;
