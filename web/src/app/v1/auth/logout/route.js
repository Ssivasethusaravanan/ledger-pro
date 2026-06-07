import { NextResponse } from 'next/server';

export async function POST(request) {
  try {
    const backendRes = await fetch('https://ledger-pro-api.onrender.com/v1/auth/logout', {
      method: 'POST',
      headers: {
        'Cookie': request.headers.get('cookie') || ''
      }
    });

    const response = new NextResponse(null, { status: backendRes.status });

    const setCookieHeader = backendRes.headers.get('set-cookie');
    if (setCookieHeader) {
      let newCookie = setCookieHeader
        .replace(/;\s*Secure/ig, '')
        .replace(/;\s*SameSite=Strict/ig, '; SameSite=Lax');
      
      response.headers.set('Set-Cookie', newCookie);
    }

    return response;
  } catch (error) {
    console.error('Logout proxy error:', error);
    return new NextResponse(null, { status: 500 });
  }
}
