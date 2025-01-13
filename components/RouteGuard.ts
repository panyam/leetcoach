import React, { useContext, useState, useEffect } from 'react';
import { useRouter } from 'next/router';
import { UserContext } from "./UserContext";
import Auth from "./Auth";

export default function RouteGuard({ children }: any) {
  const router = useRouter();
  const [authorized, setAuthorized] = useState(false);
  const {loggedInUser, setLoggedInUser} = useContext(UserContext) as any;

  useEffect(() => {
    if (!router.isReady) return;

    // on initial load - run auth check 
    authCheck(router.asPath);

    // on route change start - hide page content by setting authorized to false  
    const hideContent = () => setAuthorized(false);
    router.events.on('routeChangeStart', hideContent);

    // on route change complete - run auth check 
    router.events.on('routeChangeComplete', authCheck)

    // unsubscribe from events in useEffect return function
    return () => {
      router.events.off('routeChangeStart', hideContent);
      router.events.off('routeChangeComplete', authCheck);
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [router.isReady]);

  function authCheck(url: string) {
    // redirect to login page if accessing a private page and not logged in 
    const publicPaths = ["/login", "/"];
    const privatePaths = ["<list_of_routes_to_be_protected>"];
    const path = url.split('?')[0];
    const loggedInUser = new Auth().loggedInUser;
    const isLoggedIn = loggedInUser != null && (loggedInUser.id || "").trim() !== "";
    const needsLogin = !publicPaths.includes(path) && !isLoggedIn && privatePaths.includes(path);
    setAuthorized(!needsLogin);
    if (needsLogin) {
      router.push({
        pathname: '/login',
        query: { callbackURL: router.asPath }
      });
    } else {
      setLoggedInUser(loggedInUser);
    }
  }

  return (authorized && children);
}
