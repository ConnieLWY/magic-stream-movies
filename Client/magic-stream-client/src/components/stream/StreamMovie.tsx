import { useParams, useNavigate } from "react-router-dom";
import ReactPlayer from "react-player";
import Container from "react-bootstrap/Container";
import Button from "react-bootstrap/Button";
import Alert from "react-bootstrap/Alert";

function StreamMovie() {
  const { yt_id } = useParams();
  const navigate = useNavigate();

  if (!yt_id) {
    return (
      <Container className="py-4">
        <Alert variant="secondary" className="text-center">
          No video selected
        </Alert>
      </Container>
    );
  }

  return (
    <Container className="py-4">
      <Button
        variant="outline-secondary"
        size="sm"
        className="mb-3"
        onClick={() => navigate(-1)}
      >
        &larr; Back
      </Button>
      <div className="ratio ratio-16x9 rounded overflow-hidden shadow-sm bg-black">
        <ReactPlayer
          src={`https://www.youtube.com/watch?v=${yt_id}`}
          controls
          playing
          width="100%"
          height="100%"
        />
      </div>
    </Container>
  );
}

export default StreamMovie;
