import { NextResponse } from 'next/server';

export function middleware(request) {
  const sessionToken = request.cookies.get('session_token');
  const { pathname } = request.nextUrl;

  // Paths that don't require authentication
  const publicPaths = ['/login', '/signup'];
  
  const isPublicPath = publicPaths.some(p => pathname.startsWith(p));

  // If trying to access a protected route without a token, redirect to login
  if (!sessionToken && !isPublicPath) {
    const loginUrl = new URL('/login', request.url);
    return NextResponse.redirect(loginUrl);
  }

  // If trying to access login while already authenticated, redirect to dashboard
  if (sessionToken && isPublicPath) {
    const dashboardUrl = new URL('/', request.url);
    return NextResponse.redirect(dashboardUrl);
  }

  return NextResponse.next();
}

// Only run middleware on app routes, exclude static files, images, and API routes
export const config = {
  matcher: [
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};
