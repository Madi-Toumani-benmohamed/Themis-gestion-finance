"use client"

import * as React from "react"

export interface HalftonePalette {
  background: string
  ink: string
  warm: string
}
export type HalftonePreset = "night" | "paper" | "ember" | "cobalt"
export interface FooterLink { label: string; href?: string }
export interface FooterColumn { title: string; links: FooterLink[] }
export interface HalftoneWordmarkFooterProps {
  wordmark?: string
  label?: string
  meta?: string
  columns?: FooterColumn[]
  backToTop?: boolean
  preset?: HalftonePreset
  palette?: { background?: string; ink?: string; warm?: string }
  dotGap?: number
  radius?: number
  crop?: number
  grain?: number
  interactive?: boolean
  fontFamily?: string
  fontWeight?: number
  monoFamily?: string
  height?: string
  className?: string
}

const PRESETS: Record<HalftonePreset, HalftonePalette> = {
  night: { background: "#0c0c0b", ink: "#e9e4da", warm: "#e9762b" },
  paper: { background: "#efebe3", ink: "#17150f", warm: "#c2361c" },
  ember: { background: "#160a06", ink: "#f2c48c", warm: "#ff5a1f" },
  cobalt: { background: "#0d1b4d", ink: "#dfe7ff", warm: "#ffcd3c" },
}
const SANS_STACK = '"Inter Tight", "Inter Variable", "Inter", "Helvetica Neue", Arial, system-ui, sans-serif'
const MONO_STACK = '"JetBrains Mono", "SF Mono", ui-monospace, Menlo, Consolas, monospace'
const DEFAULT_COLUMNS: FooterColumn[] = [
  { title: "Work", links: [{ label: "Index" }, { label: "Archive" }, { label: "Journal" }] },
  { title: "Studio", links: [{ label: "About" }, { label: "Process" }, { label: "Careers" }] },
  { title: "Contact", links: [{ label: "hello@halden.co" }, { label: "+47 22 00 41 00" }] },
]

export function hwfHex(hex: string): [number, number, number] | null {
  const match = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec((hex || "").trim())
  if (!match) return null
  const full = match[1].length === 3 ? match[1].split("").map((c) => c + c).join("") : match[1]
  return [parseInt(full.slice(0, 2), 16), parseInt(full.slice(2, 4), 16), parseInt(full.slice(4, 6), 16)]
}
export function hwfRadius(coverage: number, gap: number) {
  return Math.sqrt(Math.max(0, Math.min(1, coverage))) * gap * 0.56
}
export function hwfStep(x: number, y: number, vx: number, vy: number, hx: number, hy: number, px: number, py: number, r: number) {
  const dx = x - px, dy = y - py, d2 = dx * dx + dy * dy
  if (r > 0 && d2 < r * r) {
    const d = Math.sqrt(d2) + 0.001, f = (1 - d / r) * 2.6
    vx += (dx / d) * f; vy += (dy / d) * f
  }
  vx = (vx + (hx - x) * 0.05) * 0.84
  vy = (vy + (hy - y) * 0.05) * 0.84
  return [x + vx, y + vy, vx, vy]
}
export function hwfWarmth(x: number, y: number, hx: number, hy: number, span: number) {
  return span > 0 ? Math.min(1, Math.hypot(x - hx, y - hy) / span) : 0
}
export function hwfFitSize(measured: number, width: number) {
  return measured > 0 ? (width / measured) * 100 : 0
}

const HWF_CSS = `
.hwf-root{position:relative;width:100%;overflow:hidden;isolation:isolate;background:var(--hwf-bg);color:var(--hwf-ink);-webkit-font-smoothing:antialiased}
.hwf-root canvas{position:absolute;inset:0;width:100%;height:100%;display:block;max-width:none}
.hwf-top{position:relative;z-index:1;display:grid;grid-template-columns:minmax(0,1.25fr) repeat(3,minmax(0,1fr));gap:24px;padding:clamp(22px,3.4vw,40px) clamp(20px,3.4vw,44px) 0;font-family:var(--hwf-mono);font-size:10.5px;letter-spacing:.18em;text-transform:uppercase;line-height:1.75}
.hwf-top--single{grid-template-columns:minmax(0,1fr)}
.hwf-brand strong{display:block;font-weight:500}.hwf-brand span{opacity:.65}
.hwf-col h3{margin:0;font:inherit;opacity:.55;font-weight:500}.hwf-col ul{list-style:none;margin:0;padding:0}
.hwf-col a,.hwf-up{color:inherit;text-decoration:none;opacity:.78;background:none;border:0;padding:0;font:inherit;letter-spacing:inherit;text-transform:inherit;cursor:pointer;background-image:linear-gradient(currentColor,currentColor);background-size:0% 1px;background-repeat:no-repeat;background-position:0 100%;transition:opacity .25s ease,background-size .35s cubic-bezier(.2,.7,.1,1)}
.hwf-col a:hover,.hwf-up:hover{opacity:1;background-size:100% 1px}.hwf-col a:focus-visible,.hwf-up:focus-visible{outline:2px solid currentColor;outline-offset:4px;opacity:1}
.hwf-rule{position:relative;z-index:1;display:flex;justify-content:flex-end;align-items:center;gap:16px;margin:clamp(18px,2.6vw,30px) clamp(20px,3.4vw,44px) 0;padding-top:12px;border-top:1px solid color-mix(in srgb,var(--hwf-ink) 24%,transparent);font-family:var(--hwf-mono);font-size:10.5px;letter-spacing:.18em;text-transform:uppercase}
.hwf-sr{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%);white-space:nowrap}
@media(max-width:720px){.hwf-top{grid-template-columns:minmax(0,1fr) minmax(0,1fr);row-gap:18px}.hwf-top--single{grid-template-columns:minmax(0,1fr)}.hwf-brand{grid-column:1/-1}}
@media(prefers-reduced-motion:reduce){.hwf-col a,.hwf-up{transition:none}}
`

type DotPointerEvent = PointerEvent
type DotState = { x: number; y: number; vx: number; vy: number; hx: number; hy: number; r: number }

export default function HalftoneWordmarkFooter({
  wordmark = "halden", label = "Halden Studio", meta = "©2026 · Oslo / Lisbon", columns = DEFAULT_COLUMNS,
  backToTop = true, preset = "night", palette, dotGap = 6, radius = 80, crop = 0.2,
  grain = 0.5, interactive = true, fontFamily = SANS_STACK, fontWeight = 800,
  monoFamily = MONO_STACK, height = "560px", className = "",
}: HalftoneWordmarkFooterProps) {
  const colors: HalftonePalette = {
    background: palette?.background ?? PRESETS[preset].background,
    ink: palette?.ink ?? PRESETS[preset].ink,
    warm: palette?.warm ?? PRESETS[preset].warm,
  }
  const rootRef = React.useRef<HTMLElement | null>(null)
  const canvasRef = React.useRef<HTMLCanvasElement | null>(null)
  const opts = React.useRef({ colors, dotGap, radius, crop, grain, interactive, fontFamily, fontWeight, wordmark })
  opts.current = { colors, dotGap, radius, crop, grain, interactive, fontFamily, fontWeight, wordmark }
  const key = [wordmark, dotGap, crop, grain, fontFamily, fontWeight, colors.background, colors.ink, colors.warm].join("|")

  React.useEffect(() => {
    const root = rootRef.current, canvas = canvasRef.current
    if (!root || !canvas) return
    const ctx = canvas.getContext("2d")
    if (!ctx) return
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches
    let w = 0, h = 0, dpr = 1, px = -1e5, py = -1e5, raf = 0, visible = true, awake = true
    let ring = { x: 0, y: 0, t: -1 }, dots: DotState[] = [], grainTile: HTMLCanvasElement | null = null
    const ink = hwfHex(opts.current.colors.ink) || [233, 228, 218]
    const warm = hwfHex(opts.current.colors.warm) || [233, 118, 43]
    const layout = () => {
      const o = opts.current, rect = root.getBoundingClientRect()
      w = Math.max(1, Math.round(rect.width)); h = Math.max(1, Math.round(rect.height))
      dpr = Math.min(window.devicePixelRatio || 1, 2)
      canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr)
      const mask = document.createElement("canvas"); mask.width = w; mask.height = h
      const m = mask.getContext("2d", { willReadFrequently: true }); if (!m) return
      // Canvas does not resolve CSS custom properties in its font shorthand.
      const family = o.fontFamily.includes("var(") ? getComputedStyle(root).fontFamily : o.fontFamily
      m.font = `${o.fontWeight} 100px ${family}`
      const word = o.wordmark || " ", fit = hwfFitSize(m.measureText(word).width, w * 1.02)
      m.font = `${o.fontWeight} ${fit}px ${family}`
      const box = m.measureText(word), cap = box.actualBoundingBoxAscent || fit * 0.72
      m.fillStyle = "#000"; m.textBaseline = "alphabetic"
      m.fillText(word, (w - box.width) / 2, h + cap * Math.max(0, Math.min(0.6, o.crop)))
      const img = m.getImageData(0, 0, w, h).data, gap = Math.max(3, o.dotGap), cols = Math.ceil(w / gap), rows = Math.ceil(h / gap)
      dots = []
      for (let gy = 0; gy < rows; gy++) for (let gx = 0; gx < cols; gx++) {
        let sum = 0
        for (let sy = 0; sy < 3; sy++) for (let sx = 0; sx < 3; sx++) {
          const x = Math.min(w - 1, Math.floor(gx * gap + ((sx + 0.5) * gap) / 3))
          const y = Math.min(h - 1, Math.floor(gy * gap + ((sy + 0.5) * gap) / 3))
          sum += img[(y * w + x) * 4 + 3]
        }
        const coverage = sum / (9 * 255)
        if (coverage < 0.06) continue
        const hx = gx * gap + gap / 2, hy = gy * gap + gap / 2
        dots.push({ x: hx, y: hy, vx: 0, vy: 0, hx, hy, r: hwfRadius(coverage, gap) })
      }
      grainTile = document.createElement("canvas"); grainTile.width = grainTile.height = 256
      const g = grainTile.getContext("2d")
      if (g) {
        const id = g.createImageData(256, 256)
        for (let i = 0; i < id.data.length; i += 4) {
          const value = Math.random() * 255
          id.data[i] = id.data[i + 1] = id.data[i + 2] = value
          id.data[i + 3] = Math.random() < 0.5 ? 0 : 255
        }
        g.putImageData(id, 0, 0)
      }
      awake = true
    }
    const draw = (now: number) => {
      const o = opts.current
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0); ctx.fillStyle = o.colors.background; ctx.fillRect(0, 0, w, h)
      const pushRadius = o.interactive && !reduced ? Math.max(0, o.radius) : 0
      const age = ring.t < 0 ? 1e9 : now - ring.t, ringR = age * 0.9
      let moving = false
      for (const dot of dots) {
        if (!reduced) {
          if (age < 900) {
            const dx = dot.hx - ring.x, dy = dot.hy - ring.y, d = Math.hypot(dx, dy) + 0.001
            const band = 1 - Math.min(1, Math.abs(d - ringR) / 26)
            if (band > 0) { const f = band * 3.2 * (1 - age / 900); dot.vx += dx / d * f; dot.vy += dy / d * f }
          }
          const next = hwfStep(dot.x, dot.y, dot.vx, dot.vy, dot.hx, dot.hy, px, py, pushRadius)
          ;[dot.x, dot.y, dot.vx, dot.vy] = next
          if (Math.abs(dot.vx) + Math.abs(dot.vy) > 0.02 || Math.abs(dot.x - dot.hx) + Math.abs(dot.y - dot.hy) > 0.05) moving = true
          else { dot.x = dot.hx; dot.y = dot.hy; dot.vx = 0; dot.vy = 0 }
        }
        const k = hwfWarmth(dot.x, dot.y, dot.hx, dot.hy, 28)
        ctx.fillStyle = k > 0.02 ? `rgb(${Math.round(ink[0] + (warm[0] - ink[0]) * k)},${Math.round(ink[1] + (warm[1] - ink[1]) * k)},${Math.round(ink[2] + (warm[2] - ink[2]) * k)})` : o.colors.ink
        ctx.beginPath(); ctx.arc(dot.x, dot.y, dot.r, 0, 6.2832); ctx.fill()
      }
      if (grainTile && o.grain > 0) {
        ctx.globalAlpha = Math.min(1, o.grain) * 0.09; ctx.globalCompositeOperation = "overlay"
        const pattern = ctx.createPattern(grainTile, "repeat")
        if (pattern) { ctx.fillStyle = pattern; ctx.fillRect(0, 0, w, h) }
        ctx.globalAlpha = 1; ctx.globalCompositeOperation = "source-over"
      }
      awake = moving || age < 900 || px > -1e4
    }
    const loop = (now: number) => { raf = 0; if (!visible) return; draw(now); if (awake) raf = requestAnimationFrame(loop) }
    const wake = () => { awake = true; if (!raf && visible) raf = requestAnimationFrame(loop) }
    const toLocal = (event: DotPointerEvent) => { const rect = canvas.getBoundingClientRect(); return [event.clientX - rect.left, event.clientY - rect.top] }
    const onMove = (event: DotPointerEvent) => { if (!opts.current.interactive) return; [px, py] = toLocal(event); wake() }
    const onLeave = () => { px = -1e5; py = -1e5; wake() }
    const onDown = (event: DotPointerEvent) => {
      if (!opts.current.interactive || reduced || (event.target as HTMLElement).closest("a,button")) return
      const [x, y] = toLocal(event); ring = { x, y, t: performance.now() }; wake()
    }
    layout(); draw(performance.now()); wake()
    root.addEventListener("pointermove", onMove); root.addEventListener("pointerleave", onLeave); root.addEventListener("pointerdown", onDown)
    const resize = new ResizeObserver(() => { layout(); wake() }); resize.observe(root)
    const observer = new IntersectionObserver(entries => { visible = entries[0]?.isIntersecting ?? true; if (visible) wake() }); observer.observe(root)
    let alive = true
    document.fonts?.ready?.then(() => { if (alive) { layout(); wake() } })
    return () => {
      alive = false; cancelAnimationFrame(raf); resize.disconnect(); observer.disconnect()
      root.removeEventListener("pointermove", onMove); root.removeEventListener("pointerleave", onLeave); root.removeEventListener("pointerdown", onDown)
    }
  }, [key])

  return <footer ref={rootRef} className={`hwf-root ${className}`.trim()} style={{ height, "--hwf-bg": colors.background, "--hwf-ink": colors.ink, "--hwf-mono": monoFamily } as React.CSSProperties}>
    <style>{HWF_CSS}</style>
    <canvas ref={canvasRef} aria-hidden="true" />
    <span className="hwf-sr">{wordmark}</span>
    <div className={`hwf-top${columns.length ? "" : " hwf-top--single"}`}>
      <div className="hwf-brand"><strong>{label}</strong><span>{meta}</span></div>
      {columns.map(column => <nav className="hwf-col" key={column.title} aria-label={column.title}>
        <h3>{column.title}</h3><ul>{column.links.map(link => <li key={link.label}><a href={link.href || "#"}>{link.label}</a></li>)}</ul>
      </nav>)}
    </div>
    <div className="hwf-rule">{backToTop && <button type="button" className="hwf-up" onClick={() => window.scrollTo({ top: 0, behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth" })}>Remonter en haut ↑</button>}</div>
  </footer>
}
