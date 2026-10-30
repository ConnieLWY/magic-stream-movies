import { useParams, useNavigate } from "react-router-dom";
import Container from "react-bootstrap/Container";
import { ArrowLeftIcon } from "../brand/Icons";

// YouTube video IDs are exactly 11 characters from this set
const YOUTUBE_ID = /^[A-Za-z0-9_-]{11}$/;

function StreamMovie() {
  const { yt_id: youTubeId } = useParams();
  const navigate = useNavigate();

  if (!youTubeId || !YOUTUBE_ID.test(youTubeId)) {
    return (
      <Container className="section">
        <div className="empty-state">This trailer link is not valid.</div>
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
        {/* Plain YouTube embed: no player library needed for a single provider */}
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${youTubeId}?autoplay=1&rel=0`}
          title="Trailer"
          allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
          allowFullScreen
          referrerPolicy="strict-origin-when-cross-origin"
        />
      </div>
    </Container>
  );
}

export default StreamMovie;
