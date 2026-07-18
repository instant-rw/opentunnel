import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatRelativeTime(value: string): string {
  const elapsed = Date.now() - new Date(value).getTime();
  const seconds = Math.max(0, Math.floor(elapsed / 1000));

  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

export function decodeBody(base64?: string): string {
  if (!base64) return "";
  try {
    return decodeURIComponent(
      Array.from(atob(base64))
        .map(
          (character) =>
            `%${character.charCodeAt(0).toString(16).padStart(2, "0")}`,
        )
        .join(""),
    );
  } catch {
    return "[Binary body]";
  }
}
