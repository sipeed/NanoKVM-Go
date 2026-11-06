import Cookies from 'js-cookie';

const COOKIE_TOKEN_KEY = 'nano-kvm-token';
const AUTH_MARKER_KEY = 'nano-kvm-auth';

export function existToken() {
  const token = Cookies.get(AUTH_MARKER_KEY) || Cookies.get(COOKIE_TOKEN_KEY);
  return !!token;
}

export function getToken() {
  const token = Cookies.get(COOKIE_TOKEN_KEY);
  if (!token) return null;

  return token;
}

export function setToken(token: string) {
  void token;
  Cookies.set(AUTH_MARKER_KEY, '1', {
    expires: 30,
    sameSite: 'strict',
    secure: window.location.protocol === 'https:'
  });
}

export function removeToken() {
  Cookies.remove(AUTH_MARKER_KEY);
  Cookies.remove(COOKIE_TOKEN_KEY);
}
