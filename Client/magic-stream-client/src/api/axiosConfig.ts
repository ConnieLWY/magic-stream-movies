import axios from 'axios'

const apiUrl = import.meta.env.VITE_API_BASE_URL


export const axiosClient =  axios.create({
    baseURL: apiUrl,
    headers:{'Content-Type': 'application/json'},
    withCredentials: true
})
