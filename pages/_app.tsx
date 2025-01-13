import "../styles/globals.scss";
import type { AppProps } from "next/app";
import { UserContextProvider } from '../components/UserContext'
import RouteGuard from "../components/RouteGuard";

function MyApp({ Component, pageProps }: AppProps) {
  return <UserContextProvider>
    <RouteGuard>
      <Component {...pageProps} />
    </RouteGuard>
  </UserContextProvider>;
}

export default MyApp;
