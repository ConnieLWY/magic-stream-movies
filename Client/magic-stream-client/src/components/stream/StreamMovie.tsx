import { useParams, useNavigate } from "react-router-dom";
import ReactPlayer from "react-player";
import Container from "react-bootstrap/Container";
import { ArrowLeftIcon } from "../brand/Icons";

function StreamMovie() {
  const { yt_id } = useParams();
  const navigate = useNavigate();

  if (!yt_id) {
    return (
      <Container className="section">
        <div className="empty-state">No video selected</div>
      </Container>
    );
  }

  return (
    <Container className="theater">
      <button
        type="button"
        className="btn btn-ghost btn-sm mb-4"
        onClick={() => navigate(-1)}
      >
        <ArrowLeftIcon size={14} /> Back
      </button>
      <div className="theater-stage ratio ratio-16x9">
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
