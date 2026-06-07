import Link from 'next/link';
import Button from '@/components/Button';
import GlassCard from '@/components/GlassCard';

export default function NotFound() {
  return (
    <div className="flex h-full min-h-[60vh] w-full flex-col items-center justify-center p-4">
      <GlassCard className="max-w-md w-full text-center space-y-6 py-12">
        <div className="mx-auto flex items-center justify-center text-6xl font-black text-brand-bg-glass text-stroke">
          404
        </div>
        
        <div>
          <h2 className="text-xl font-bold text-white tracking-tight">Page Not Found</h2>
          <p className="mt-2 text-sm text-brand-text-secondary px-4">
            The page you&apos;re looking for doesn&apos;t exist or has been moved.
          </p>
        </div>
        
        <div className="pt-4 flex justify-center">
          <Link href="/">
            <Button variant="primary">
              Return Home
            </Button>
          </Link>
        </div>
      </GlassCard>
    </div>
  );
}
