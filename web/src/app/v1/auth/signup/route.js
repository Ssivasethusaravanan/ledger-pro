import { NextResponse } from 'next/server';

export async function POST(request) {
  try {
    const body = await request.json();
    
    const backendRes = await fetch('https://ledger-pro-api.onrender.com/v1/auth/signup', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    const data = await backendRes.json();
    const response = NextResponse.json(data, { status: backendRes.status });

    // Forward the Set-Cookie header but strip the Secure flag
    // so it works on local network IPs (like 192.168.x.x) during development
    const setCookieHeader = backendRes.headers.get('set-cookie');
    if (setCookieHeader) {
      let newCookie = setCookieHeader
        .replace(/;\s*Secure/ig, '')
        .replace(/;\s*SameSite=Strict/ig, '; SameSite=Lax');
      
      response.headers.set('Set-Cookie', newCookie);
    }

    return response;
  } catch (error) {
    console.error('Signup proxy error:', error);
    return NextResponse.json({ error: 'Internal Server Error' }, { status: 500 });
  }
}
