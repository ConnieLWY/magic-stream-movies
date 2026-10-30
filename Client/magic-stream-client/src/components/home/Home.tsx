import { useEffect, useState } from 'react'
import Container from 'react-bootstrap/Container'
import Spinner from 'react-bootstrap/Spinner'
import { axiosClient } from '../../api/axiosConfig'
import Movies from '../movies/Movies'
import type { MovieModel } from '../../types/movie'


interface HomeProps {
    updateMovieReview?: (imdb_id: string) => void
}

function Home({ updateMovieReview }: HomeProps) {
    const [movies, setMovies] = useState<MovieModel[]>([])
    const [loading, setLoading] = useState(false)
    const [message, setMessage] = useState("")

    useEffect(() => {
        const fetchMovies = async () => {
            setLoading(true)
            setMessage("")
            try {
                const response = await axiosClient.get<MovieModel[]>('/movies')
                setMovies(response.data ?? [])

                if (!response.data || response.data.length === 0) {
                    setMessage('There are currently no movies available')
                }

            } catch (error) {
                console.error("Error fetching movies: ", error)
                setMessage("Error fetching movies")
            } finally {
                setLoading(false)
            }
        }

        fetchMovies()
    }, [])

  return (
    <Container className="py-4">
        {loading ? (
            <div className="d-flex justify-content-center my-5">
                <Spinner animation="border" role="status" />
            </div>
        ) : (
            <Movies movies={movies} message={message} updateMovieReview={updateMovieReview} />
        )}
    </Container>
  )
}

export default Home
