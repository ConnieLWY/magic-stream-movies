import {useEffect, useMemo, useRef} from 'react';
import axios from 'axios';

import useAuth from './useAuth';

const apiUrl = import.meta.env.VITE_API_BASE_URL;

type FailedQueueEntry = {
    resolve: () => void;
    reject: (reason?: unknown) => void;
};

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
        async error => {
            console.log('⚠ Interceptor caught error:', error);
            const originalRequest = error.config;

        if (originalRequest.url.includes('/refresh') && error.response.status === 401) {
            //edge case where the refresh token is invalid or expired
            console.error('❌ Refresh token has expired or is invalid.');
            return Promise.reject(error); // fail directly, no retry
        }

            if (error.response && error.response.status === 401 && !originalRequest._retry) {

                if (isRefreshing.current) {
                return new Promise<void>((resolve, reject) => {
                failedQueue.current.push({ resolve, reject });
                })
                .then(() => axiosAuth(originalRequest))
                .catch(err => Promise.reject(err));
            }

            originalRequest._retry = true;
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

                        localStorage.removeItem('user');
                        setAuth(null); // Clear auth state
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
