import {useEffect, useMemo, useRef} from 'react';
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';

import useAuth from './useAuth';

const apiUrl = import.meta.env.VITE_API_BASE_URL;

type FailedQueueEntry = {
    resolve: () => void;
    reject: (reason?: unknown) => void;
};

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
    _retry?: boolean;
}

const useAxiosPrivate = () =>{

    const axiosAuth = useMemo(() => axios.create({
        baseURL: apiUrl,
        withCredentials: true, // important for HTTP-only cookies
    }), []);


    const {setAuth} = useAuth();

    const isRefreshing = useRef(false);
    const failedQueue = useRef<FailedQueueEntry[]>([]);

     useEffect(() => {
        // Helper to process queued requests after token refresh
        const processQueue = (error: unknown = null) => {
            failedQueue.current.forEach(prom => {
                if (error) {
                prom.reject(error);
                } else {
                prom.resolve();
                }
            });

            failedQueue.current = [];
        };

        const responseInterceptor = axiosAuth.interceptors.response.use(
        response => response,
        async (error: AxiosError) => {
            console.log('⚠ Interceptor caught error:', error);
            const originalRequest = error.config as RetryableRequestConfig | undefined;

            if (!originalRequest) {
                return Promise.reject(error);
            }

        if (originalRequest.url?.includes('/refresh') && error.response?.status === 401) {
            //edge case where the refresh token is invalid or expired
            console.error('❌ Refresh token has expired or is invalid.');
            return Promise.reject(error); // fail directly, no retry
        }

            if (error.response?.status === 401 && !originalRequest._retry) {
                originalRequest._retry = true;

                if (isRefreshing.current) {
                return new Promise<void>((resolve, reject) => {
                failedQueue.current.push({ resolve, reject });
                })
                .then(() => axiosAuth(originalRequest))
                .catch(err => Promise.reject(err));
            }

            isRefreshing.current = true;

            return new Promise((resolve, reject) => {
                axiosAuth
                .post('/refresh')
                .then(() => {

                    processQueue(null);

                axiosAuth(originalRequest)
                    .then(resolve)
                    .catch(reject);

                })
                .catch(refreshError => {

                        processQueue(refreshError);

                        if (
                            axios.isAxiosError(refreshError) &&
                            (refreshError.response?.status === 401 ||
                                refreshError.response?.status === 403)
                        ) {
                            localStorage.removeItem('user');
                            setAuth(null); // Clear auth state
                        }
                        reject(refreshError); // fail the original promise chain
                })
                .finally(() => {
                        isRefreshing.current = false;
                });
            });
            }

            return Promise.reject(error);
        }
        );

        return () => axiosAuth.interceptors.response.eject(responseInterceptor);
    }, [axiosAuth, setAuth]);

    return axiosAuth;
}

export default useAxiosPrivate;
