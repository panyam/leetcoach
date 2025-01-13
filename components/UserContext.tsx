
import React, { useContext, createContext, useState, useEffect } from 'react';
import Auth from "./Auth";

export const UserContext = createContext({});
export function UserContextProvider(props: any) {
  const [loggedInUser, setLoggedInUser] = useState(null);
  // console.log("LI, SLIU:", loggedInUser, setLoggedInUser, props);
  useEffect(() => {
    // console.log("auth.LoggedInUser: ", new Auth().loggedInUser);
    const user = new Auth().loggedInUser;
    setLoggedInUser(user);
  }, []);

  return (
    <UserContext.Provider value={{loggedInUser, setLoggedInUser}}>
      {props.children}
    </UserContext.Provider>
  );
}
