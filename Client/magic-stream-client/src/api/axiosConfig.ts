import axios from 'axios'
import type { User } from '../types/user'

const apiUrl = import.meta.env.VITE_API_BASE_URL


export const axiosClient =  axios.create({
    baseURL: apiUrl,
    headers:{'Content-Type': 'application/json'},
    withCredentials: true // sends the HttpOnly refresh_token cookie
})

let refreshInFlight: Promise<User> | null = null

// Exchanges the HttpOnly refresh_token cookie for a new access token (plus user info).
// The server rotates the refresh token on every call, so concurrent callers must share
// one request; a second request with the old cookie would be rejected as revoked.
export function refreshSession(): Promise<User> {
    if (!refreshInFlight) {
        refreshInFlight = axiosClient
            .post<User>('/auth/refresh')
            .then(response => response.data)
            .finally(() => {
                refreshInFlight = null
            })
    }

    return refreshInFlight
}
