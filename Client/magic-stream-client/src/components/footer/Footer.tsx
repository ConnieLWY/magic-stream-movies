import Container from "react-bootstrap/Container";
import { BRAND } from "../../brand";
import Logo from "../brand/Logo";

function Footer() {
  return (
    <footer className="site-footer">
      <Container>
        <Logo size={22} />
        <span>
          © {new Date().getFullYear()} {BRAND.name}. {BRAND.tagline}
        </span>
      </Container>
    </footer>
  );
}

export default Footer;
