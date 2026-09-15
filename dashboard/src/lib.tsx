import { useEffect, useRef, useState, type ReactNode } from "react";

/** Deteksi preferensi reduced-motion pengguna. */
export function usePrefersReducedMotion(): boolean {
  const [prm, setPrm] = useState(
    () => typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const onChange = (e: MediaQueryListEvent) => setPrm(e.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);
  return prm;
}

/**
 * Wrapper reveal-on-scroll. Tambah class `.in` saat elemen masuk viewport.
 * CSS menangani transisi; pada reduced-motion elemen langsung tampil.
 */
export function Reveal({
  children,
  className = "",
  delay = 0,
  lines = false,
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
  lines?: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) {
            el.classList.add("in");
            io.disconnect();
          }
        });
      },
      { threshold: 0.15, rootMargin: "0px 0px -6% 0px" }
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);
  return (
    <div
      ref={ref}
      className={`${lines ? "lines" : "rv"} ${className}`}
      style={delay ? { transitionDelay: `${delay}ms` } : undefined}
    >
      {children}
    </div>
  );
}

const GLYPHS = "█▓▒░<>/\\{}[]#*+=—";

/** Efek scramble-decode: teks "terurai" dari glyph acak menuju target. */
export function useScramble(target: string): string {
  const prm = usePrefersReducedMotion();
  const [out, setOut] = useState(target);
  useEffect(() => {
    if (prm) {
      setOut(target);
      return;
    }
    let raf = 0;
    let start: number | null = null;
    const dur = Math.min(850, 300 + target.length * 12);
    const tick = (t: number) => {
      if (start === null) start = t;
      const p = Math.min(1, (t - start) / dur);
      const keep = Math.floor(p * target.length);
      let s = target.slice(0, keep);
      for (let i = keep; i < target.length; i++) {
        const c = target[i];
        s += c === " " ? " " : GLYPHS[Math.floor(Math.random() * GLYPHS.length)];
      }
      setOut(p < 1 ? s : target);
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [target, prm]);
  return out;
}

/** Salin teks ke clipboard dengan fallback untuk konteks non-secure. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    try {
      const ta = document.createElement("textarea");
      ta.value = text;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      document.body.removeChild(ta);
      return true;
    } catch {
      return false;
    }
  }
}
