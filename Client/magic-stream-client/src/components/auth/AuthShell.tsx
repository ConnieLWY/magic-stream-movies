import type { ReactNode } from "react";
import { BRAND } from "../../brand";
import { LogoMark } from "../brand/Logo";

interface AuthShellProps {
  wide?: boolean;
  children: ReactNode;
}

function AuthShell({ wide, children }: AuthShellProps) {
  return (
    <div className="auth-wrap">
      <aside className="auth-art">
        <LogoMark size={56} />
        <h2>{BRAND.tagline}</h2>
        <p>{BRAND.description}</p>
      </aside>
      <div className="auth-panel">
        <div className={`auth-card${wide ? " wide" : ""}`}>{children}</div>
      </div>
    </div>
  );
}

export default AuthShell;
