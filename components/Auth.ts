import axios from "axios";

const isBrowser = typeof window !== "undefined";

/**
 * Proxy/Entry point to the front end
 */
export default class Auth {
  seatLocations: any;
  constructor(public readonly hostname: string = "http://localhost:3000") {
    this.hostname = hostname;
  }

  // Need a better way than this
  isSuperUser(user: any): boolean {
    return user.email === "sri.panyam@gmail.com";
  }

  makeUrl(path: string): string {
    return `${this.hostname}${path}`;
  }

  logout(callback: (resp: any, err: Error | null) => void) {
    const url = this.makeUrl("/auth/logout/");
    axios
      .get(url)
      .then((resp) => {
        document.cookie = "loggedInUser=; SameSite=None; Secure; Path=/;";
        if (callback) {
          callback(resp, null);
        }
      })
      .catch((err) => {
        if (callback) {
          callback(null, err);
        }
      });
  }

  isLoggedIn(): boolean {
    return this.loggedInUser.id != null;
  }

  get loggedInUser(): any {
    if (typeof document !== "undefined") {
      const cookieValue = document.cookie
        .split("; ")
        .find((row) => row.startsWith("loggedInUser="))
        ?.split("=")[1];
      let out = cookieValue || null;
      if (out && out != null && out.trim() !== "") {
        const parsed = JSON.parse(decodeURIComponent(out));
        parsed.created_at = new Date(parsed.created_at);
        parsed.updated_at = new Date(parsed.updated_at);
        return parsed;
      }
    }
    return { id: null };
  }

  loginUrl(provider: string, callbackUrl?: string) {
    callbackUrl = callbackUrl || (isBrowser ? window.location.href : "/");
    return this.makeUrl(`/auth/${provider}/?callbackURL=${callbackUrl}`);
  }
}
