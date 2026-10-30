import { useId } from "react";
import { BRAND } from "../../brand";

interface LogoMarkProps {
  size?: number;
}

export function LogoMark({ size = 32 }: LogoMarkProps) {
  const gradientId = useId();

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      aria-hidden="true"
    >
      <defs>
        <linearGradient
          id={gradientId}
          x1="0"
          y1="0"
          x2="64"
          y2="64"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#FFC15E" />
          <stop offset="1" stopColor="#FF5E3A" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="16" fill={`url(#${gradientId})`} />
      <path
        d="M23 19.5v25c0 1.6 1.7 2.5 3 1.7l20-12.5c1.3-.8 1.3-2.6 0-3.4l-20-12.5c-1.3-.8-3 .1-3 1.7z"
        fill="#14101C"
      />
      <path
        d="M49 7.5c.7 4.2 2.3 5.8 6.5 6.5-4.2.7-5.8 2.3-6.5 6.5-.7-4.2-2.3-5.8-6.5-6.5 4.2-.7 5.8-2.3 6.5-6.5z"
        fill="#14101C"
      />
    </svg>
  );
}

interface LogoProps {
  size?: number;
}

function Logo({ size = 32 }: LogoProps) {
  return (
    <span className="logo">
      <LogoMark size={size} />
      <span className="logo-word" style={{ fontSize: size * 0.72 }}>
        {BRAND.name}
      </span>
    </span>
  );
}

export default Logo;
