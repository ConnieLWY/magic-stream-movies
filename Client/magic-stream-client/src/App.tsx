import "./App.css";
import Header from "./components/header/Header";
import Home from "./components/home/Home";
import { Route, Routes, useNavigate } from "react-router-dom";
import Register from "./components/register/Register";
import Login from "./components/login/Login";
import RequiredAuth from "./components/RequiredAuth";
import Recommended from "./components/recommended/Recommended";
import Review from "./components/review/Review";
import { axiosClient } from "./api/axiosConfig";
import useAuth from "./hook/useAuth";

function App() {
  const navigate = useNavigate();
  const { auth, setAuth } = useAuth();

  const updateMovieReview = (imdb_id: string) => {
    navigate(`/review/${imdb_id}`);
  };

  const handleLogout = async () => {
    try {
      const response = await axiosClient.post("/logout", {
        user_id: auth?.user_id,
      });
      console.log(response.data);
      setAuth(null);
      // localStorage.removeItem('user');
      console.log("User logged out");
    } catch (error) {
      console.error("Error logging out:", error);
    }
  };

  return (
    <>
      <Header handleLogout={handleLogout} />
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
        </Route>
      </Routes>
    </>
  );
}

export default App;
