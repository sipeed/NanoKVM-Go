import axios, { AxiosHeaders, AxiosInstance, AxiosRequestConfig } from 'axios';

import { expireAuthSession } from '@/lib/auth-session.ts';
import { getBaseUrl } from '@/lib/service.ts';

type Response = {
  code: number;
  msg: string;
  data: any;
};

function logRequestError(error: unknown) {
  if (axios.isAxiosError(error)) {
    const requestUrl = error.config?.url?.split(/[?#]/, 1)[0];
    console.log('HTTP request failed', {
      code: error.code,
      status: error.response?.status,
      method: error.config?.method?.toUpperCase(),
      url: requestUrl
    });
    return;
  }

  if (error instanceof Error) {
    console.log('HTTP request failed', { name: error.name });
    return;
  }

  console.log('HTTP request failed');
}

class Http {
  private instance: AxiosInstance;

  constructor() {
    const baseURL = getBaseUrl('http');

    this.instance = axios.create({
      baseURL,
      withCredentials: true,
      timeout: 60 * 1000
    });

    this.setInterceptors();
  }

  private setInterceptors() {
    this.instance.interceptors.request.use((config) => {
      if (config.headers) {
        config.headers.Accept = 'application/json';
      }

      return config;
    });

    this.instance.interceptors.response.use(
      (response) => {
        return response.data;
      },
      (error) => {
        logRequestError(error);
        const code = error.response?.status;
        if (code === 401) {
          expireAuthSession();
        }
        return Promise.reject(error);
      }
    );
  }

  public get(url: string, params?: any, headers?: AxiosHeaders): Promise<Response> {
    return this.instance.request({
      method: 'get',
      url,
      params,
      headers
    });
  }

  public post(url: string, data?: any, config?: AxiosRequestConfig): Promise<Response> {
    return this.instance.request({
      method: 'post',
      url,
      data,
      ...config
    });
  }

  public delete(url: string, data?: any): Promise<Response> {
    return this.instance.request({
      method: 'delete',
      url,
      data
    });
  }

  public request(config: AxiosRequestConfig): Promise<Response> {
    return this.instance.request(config);
  }
}

export const http = new Http();
