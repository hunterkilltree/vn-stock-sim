import Link from "next/link";

export default function Home() {
  return (
    <div className="flex flex-1 flex-col items-center justify-center bg-zinc-50 font-sans dark:bg-black">
      <main className="flex w-full max-w-xl flex-col items-center gap-6 px-8 py-32 text-center">
        <h1 className="text-3xl font-semibold tracking-tight text-black dark:text-zinc-50">
          VN Stock Sim
        </h1>
        <p className="text-lg text-zinc-600 dark:text-zinc-400">
          Trade the past before you trade the future. Simulation-first analysis and
          paper trading for Vietnamese stocks (HOSE, HNX, UPCOM).
        </p>
        <Link
          href="/stocks"
          className="rounded-full bg-foreground px-6 py-3 text-sm font-medium text-background transition-colors hover:bg-[#383838] dark:hover:bg-[#ccc]"
        >
          Browse stocks
        </Link>
      </main>
    </div>
  );
}
