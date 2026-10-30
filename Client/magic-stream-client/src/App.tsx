import Header from "./components/header/Header";
import Footer from "./components/footer/Footer";
import Home from "./components/home/Home";
import { Route, Routes, useNavigate } from "react-router-dom";
import Register from "./components/register/Register";
import Login from "./components/login/Login";
import RequiredAuth from "./components/RequiredAuth";
import Recommended from "./components/recommended/Recommended";
import Review from "./components/review/Review";
import StreamMovie from "./components/stream/StreamMovie";
import { axiosClient } from "./api/axiosConfig";
import useAuth from "./hook/useAuth";

function App() {
  const navigate = useNavigate();
  const { setAuth } = useAuth();

  const updateMovieReview = (imdb_id: string) => {
    navigate(`/review/${imdb_id}`);
  };

  const handleLogout = async () => {
    try {
      // Server identifies the user from the refresh_token cookie and clears it
      await axiosClient.post("/auth/logout");
    } catch (error) {
      console.error("Error logging out:", error);
    } finally {
      setAuth(null);
    }
  };

  return (
    <>
      <Header handleLogout={handleLogout} />
      <main className="app-main">
        <Routes>
          <Route
            path="/"
            element={<Home updateMovieReview={updateMovieReview} />}
          ></Route>
          <Route path="/register" element={<Register />}></Route>
          <Route path="/login" element={<Login />}></Route>
          <Route element={<RequiredAuth />}>
            <Route path="/recommended" element={<Recommended />}></Route>
            <Route path="/review/:imdb_id" element={<Review />}></Route>
            <Route path="/stream/:yt_id" element={<StreamMovie />}></Route>
          </Route>
        </Routes>
      </main>
      <Footer />
    </>
  );
}

export default App;
