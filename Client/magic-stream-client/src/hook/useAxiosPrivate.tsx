import {useEffect, useMemo, useRef} from 'react';
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';

import useAuth from './useAuth';
import { refreshSession } from '../api/axiosConfig';

const apiUrl = import.meta.env.VITE_API_BASE_URL;

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
    _retry?: boolean;
}

const useAxiosPrivate = () =>{

    const axiosAuth = useMemo(() => axios.create({
        baseURL: apiUrl,
        withCredentials: true, // sends the HttpOnly refresh_token cookie
    }), []);


    const {auth, setAuth} = useAuth();

    // Interceptors read the token through a ref so the axios instance stays stable
    const accessToken = useRef(auth?.access_token);

    useEffect(() => {
        accessToken.current = auth?.access_token;
    }, [auth]);

     useEffect(() => {
        const requestInterceptor = axiosAuth.interceptors.request.use(config => {
            if (accessToken.current && !config.headers.Authorization) {
                config.headers.Authorization = `Bearer ${accessToken.current}`;
            }
            return config;
        });

        const responseInterceptor = axiosAuth.interceptors.response.use(
        response => response,
        async (error: AxiosError) => {
            const originalRequest = error.config as RetryableRequestConfig | undefined;

            if (!originalRequest || error.response?.status !== 401 || originalRequest._retry) {
                return Promise.reject(error);
            }

            originalRequest._retry = true;

            try {
                // Access token missing or expired: get a new one via the refresh cookie
                const user = await refreshSession();
                accessToken.current = user.access_token;
                setAuth(user);

                originalRequest.headers.Authorization = `Bearer ${user.access_token}`;
                return axiosAuth(originalRequest);
            } catch (refreshError) {
                if (
                    axios.isAxiosError(refreshError) &&
                    (refreshError.response?.status === 401 ||
                        refreshError.response?.status === 403)
                ) {
                    console.error('❌ Refresh token has expired or is invalid.');
                    setAuth(null); // Clear auth state
                }
                return Promise.reject(refreshError);
            }
        }
        );

        return () => {
            axiosAuth.interceptors.request.eject(requestInterceptor);
            axiosAuth.interceptors.response.eject(responseInterceptor);
        };
    }, [axiosAuth, setAuth]);

    return axiosAuth;
}

export default useAxiosPrivate;
