// Local prize-wheel component, derived from react-prize-wheel-advanced@1.0.10.
// Perf change vs upstream: all shadowBlur passes (disc halo, per-segment text
// shadows, rim glow, hub shadow) are baked once into two offscreen canvases at
// config-change time; each animation frame now only blits the rotated disc and
// the static overlay. Visual hierarchy (halo / glow / depth) is preserved.
import React, {
    forwardRef,
    useCallback,
    useEffect,
    useImperativeHandle,
    useMemo,
    useRef,
    useState,
} from 'react';

export interface PrizeWheelHandle {
    click: () => void;
}

export interface PrizeWheelProps {
    segments?: string[];
    segColors?: string[];
    onFinished?: (index: number) => void;
    primaryColor?: string;
    contrastColor?: string;
    buttonText?: string;
    spinDuration?: number;
    size?: number;
}

function lighten(hex: string, amount: number): string {
    const num = parseInt(hex.replace('#', ''), 16);
    const r = Math.min(255, (num >> 16) + amount);
    const g = Math.min(255, ((num >> 8) & 0xff) + amount);
    const b = Math.min(255, (num & 0xff) + amount);
    return `rgb(${r},${g},${b})`;
}

function darken(hex: string, amount: number): string {
    const num = parseInt(hex.replace('#', ''), 16);
    const r = Math.max(0, (num >> 16) - amount);
    const g = Math.max(0, ((num >> 8) & 0xff) - amount);
    const b = Math.max(0, (num & 0xff) - amount);
    return `rgb(${r},${g},${b})`;
}

function fitText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number, maxFontSize = 14, minFontSize = 5): number {
    let fs = maxFontSize;
    ctx.font = `bold ${fs}px system-ui, sans-serif`;
    while (ctx.measureText(text).width > maxWidth && fs > minFontSize) {
        fs -= 0.5;
        ctx.font = `bold ${fs}px system-ui, sans-serif`;
    }
    return fs;
}

export const PrizeWheel = forwardRef<PrizeWheelHandle, PrizeWheelProps>(function PrizeWheel({
    segments = [],
    segColors = [],
    onFinished,
    primaryColor = '#d4af37',
    contrastColor = '#ffffff',
    buttonText = 'SPIN',
    spinDuration = 5,
    size = 500,
}, ref) {
    const canvasRef = useRef<HTMLCanvasElement | null>(null);
    const angleRef = useRef(-Math.PI / 2);
    const spinningRef = useRef(false);
    const rafRef = useRef<number | null>(null);
    const [spinning, setSpinning] = useState(false);
    const CENTER = size / 2;
    const RADIUS = CENTER - 16;
    const BTN_R = Math.max(28, size * 0.08);

    // Pre-render rotating disc and static overlay once per config; shadows are
    // paid here (one time) instead of on every animation frame.
    const layers = useMemo(() => {
        const disc = document.createElement('canvas');
        disc.width = size;
        disc.height = size;
        const d = disc.getContext('2d');
        const overlay = document.createElement('canvas');
        overlay.width = size;
        overlay.height = size;
        const o = overlay.getContext('2d');
        const n = segments.length;
        if (!d || !o || n === 0) return { disc, overlay };
        const arc = (2 * Math.PI) / n;

        d.shadowColor = 'rgba(0,0,0,0.55)';
        d.shadowBlur = 24;
        d.beginPath();
        d.arc(CENTER, CENTER, RADIUS + 5, 0, 2 * Math.PI);
        d.fillStyle = '#0f172a';
        d.fill();
        d.shadowBlur = 0;
        for (let i = 0; i < n; i++) {
            const start = i * arc;
            const end = start + arc;
            const color = segColors[i % segColors.length] || '#888';
            d.beginPath();
            d.moveTo(CENTER, CENTER);
            d.arc(CENTER, CENTER, RADIUS, start, end);
            d.closePath();
            d.fillStyle = color;
            d.fill();
            d.strokeStyle = 'rgba(0,0,0,0.3)';
            d.lineWidth = 1.5;
            d.stroke();
            d.save();
            d.beginPath();
            d.moveTo(CENTER, CENTER);
            d.arc(CENTER, CENTER, RADIUS - 2, start, end);
            d.closePath();
            d.clip();
            d.translate(CENTER, CENTER);
            d.rotate(start + arc / 2);
            d.textAlign = 'right';
            d.fillStyle = contrastColor;
            d.shadowColor = 'rgba(0,0,0,0.7)';
            d.shadowBlur = 4;
            const label = segments[i] || '';
            const availableWidth = RADIUS - BTN_R - 18;
            const baseFontMax = n <= 8 ? 15 : n <= 20 ? 13 : n <= 40 ? 11 : 9;
            const fs = fitText(d, label, availableWidth, baseFontMax);
            d.fillText(label, RADIUS - 10, fs / 3);
            d.restore();
        }
        d.shadowBlur = 0;
        d.strokeStyle = 'rgba(255,255,255,0.12)';
        d.lineWidth = 1;
        for (let i = 0; i < n; i++) {
            const a = i * arc;
            d.beginPath();
            d.moveTo(CENTER, CENTER);
            d.lineTo(CENTER + RADIUS * Math.cos(a), CENTER + RADIUS * Math.sin(a));
            d.stroke();
        }

        o.beginPath();
        o.arc(CENTER, CENTER, RADIUS, 0, 2 * Math.PI);
        o.strokeStyle = primaryColor;
        o.lineWidth = 5;
        o.shadowColor = primaryColor;
        o.shadowBlur = 10;
        o.stroke();
        o.shadowBlur = 0;
        o.beginPath();
        o.arc(CENTER, CENTER, RADIUS - 6, 0, 2 * Math.PI);
        o.strokeStyle = 'rgba(255,255,255,0.08)';
        o.lineWidth = 1;
        o.stroke();
        const grad = o.createRadialGradient(CENTER - BTN_R * 0.3, CENTER - BTN_R * 0.3, BTN_R * 0.05, CENTER, CENTER, BTN_R);
        grad.addColorStop(0, lighten(primaryColor, 55));
        grad.addColorStop(0.6, primaryColor);
        grad.addColorStop(1, darken(primaryColor, 30));
        o.beginPath();
        o.arc(CENTER, CENTER, BTN_R, 0, 2 * Math.PI);
        o.fillStyle = grad;
        o.shadowColor = 'rgba(0,0,0,0.6)';
        o.shadowBlur = 16;
        o.fill();
        o.shadowBlur = 0;
        o.beginPath();
        o.arc(CENTER, CENTER, BTN_R, 0, 2 * Math.PI);
        o.strokeStyle = 'rgba(255,255,255,0.85)';
        o.lineWidth = 2.5;
        o.stroke();
        const btnFontSize = Math.min(14, BTN_R * 0.42);
        o.fillStyle = '#0f172a';
        o.font = `bold ${btnFontSize}px system-ui, sans-serif`;
        o.textAlign = 'center';
        o.textBaseline = 'middle';
        o.fillText(buttonText, CENTER, CENTER);
        return { disc, overlay };
    }, [segments, segColors, primaryColor, contrastColor, buttonText, size, CENTER, RADIUS, BTN_R]);

    const draw = useCallback((angle: number) => {
        const canvas = canvasRef.current;
        if (!canvas || segments.length === 0) return;
        const ctx = canvas.getContext('2d');
        if (!ctx) return;
        ctx.clearRect(0, 0, size, size);
        ctx.save();
        ctx.translate(CENTER, CENTER);
        ctx.rotate(angle);
        ctx.drawImage(layers.disc, -CENTER, -CENTER);
        ctx.restore();
        ctx.drawImage(layers.overlay, 0, 0);
    }, [layers, segments.length, size, CENTER]);

    useEffect(() => {
        draw(angleRef.current);
    }, [draw]);

    const spin = useCallback(() => {
        if (spinningRef.current || segments.length < 2) return;
        spinningRef.current = true;
        setSpinning(true);
        const n = segments.length;
        const arc = (2 * Math.PI) / n;
        const winnerIndex = Math.floor(Math.random() * n);
        const winnerMidAngle = winnerIndex * arc + arc / 2;
        const startAngle = angleRef.current;
        const TAU = 2 * Math.PI;
        const currentNorm = ((startAngle % TAU) + TAU) % TAU;
        const targetNorm = ((-Math.PI / 2 - winnerMidAngle) % TAU + TAU) % TAU;
        let forwardArc = targetNorm - currentNorm;
        if (forwardArc <= 0) forwardArc += TAU;
        const minFullSpins = Math.max(6, spinDuration * 2);
        const delta = forwardArc + minFullSpins * TAU;
        const startTime = performance.now();
        const durationMs = spinDuration * 1e3;
        const animate = (now: number) => {
            const elapsed = now - startTime;
            const t = Math.min(elapsed / durationMs, 1);
            const eased = 1 - Math.pow(1 - t, 4);
            angleRef.current = startAngle + delta * eased;
            draw(angleRef.current);
            if (t < 1) {
                rafRef.current = requestAnimationFrame(animate);
            } else {
                spinningRef.current = false;
                setSpinning(false);
                try {
                    if (onFinished && typeof onFinished === 'function') {
                        onFinished(winnerIndex);
                    }
                } catch (error) {
                    console.error('Error in onFinished callback:', error);
                }
            }
        };
        rafRef.current = requestAnimationFrame(animate);
    }, [segments.length, spinDuration, draw, onFinished]);

    useImperativeHandle(ref, () => ({ click: spin }), [spin]);
    useEffect(() => () => {
        if (rafRef.current) cancelAnimationFrame(rafRef.current);
    }, []);

    if (segments.length === 0) return null;
    const ptrW = Math.round(size * 0.074);
    const ptrH = Math.round(size * 0.118);
    const ballR = Math.round(ptrW * 0.5);
    const overlap = Math.round(ptrH * 0.2);
    const half = ptrW / 2;
    const tipY = ptrH;
    const bodyPath = [
        `M ${half} ${tipY}`,
        `C ${ptrW * 0.15} ${ptrH * 0.62}, 3 ${ptrH * 0.35}, 3 ${ballR}`,
        `A ${half - 3} ${half - 3} 0 0 1 ${ptrW - 3} ${ballR}`,
        `C ${ptrW - 3} ${ptrH * 0.35}, ${ptrW * 0.85} ${ptrH * 0.62}, ${half} ${tipY}`,
        'Z',
    ].join(' ');
    return (
        <div style={{ position: 'relative', display: 'inline-flex', flexDirection: 'column', alignItems: 'center', userSelect: 'none' }}>
            <div style={{ position: 'relative', zIndex: 20, marginBottom: -overlap, filter: `drop-shadow(0 5px 10px rgba(0,0,0,0.65)) drop-shadow(0 2px 4px rgba(0,0,0,0.4))`, lineHeight: 0 }}>
                <svg width={ptrW} height={ptrH} viewBox={`0 0 ${ptrW} ${ptrH}`} xmlns="http://www.w3.org/2000/svg" overflow="visible">
                    <defs>
                        <linearGradient id="ptrBody" x1="0%" y1="0%" x2="100%" y2="100%">
                            <stop offset="0%" stopColor={lighten(primaryColor, 65)} />
                            <stop offset="40%" stopColor={primaryColor} />
                            <stop offset="100%" stopColor={darken(primaryColor, 45)} />
                        </linearGradient>
                        <radialGradient id="ptrShine" cx="38%" cy="28%" r="58%">
                            <stop offset="0%" stopColor="rgba(255,255,255,0.6)" />
                            <stop offset="100%" stopColor="rgba(255,255,255,0)" />
                        </radialGradient>
                    </defs>
                    <path d={bodyPath} fill="url(#ptrBody)" stroke={darken(primaryColor, 25)} strokeWidth={1.5} strokeLinejoin="round" />
                    <ellipse cx={half} cy={ballR * 0.82} rx={ptrW * 0.26} ry={ballR * 0.36} fill="url(#ptrShine)" />
                    <circle cx={half} cy={ballR} r={half - 3} fill="none" stroke="rgba(255,255,255,0.45)" strokeWidth={1.2} />
                    <circle cx={half * 0.72} cy={ballR * 0.58} r={ptrW * 0.07} fill="rgba(255,255,255,0.7)" />
                </svg>
            </div>
            <canvas
                ref={canvasRef}
                width={size}
                height={size}
                onClick={spin}
                style={{ borderRadius: '50%', cursor: spinning ? 'default' : 'pointer', display: 'block', maxWidth: '100%' }}
            />
        </div>
    );
});

export default PrizeWheel;
