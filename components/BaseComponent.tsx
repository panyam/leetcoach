import React from "react";
import { useRouter, withRouter } from "next/router";
import Auth from "./Auth";

class BaseComponent<PropType> extends React.Component<PropType>{
  get isLoggedIn(): boolean {
    const loggedInUser = new Auth().loggedInUser;
    return loggedInUser != null && (loggedInUser.id || "").trim() !== "";
  }

  get loggedInUser(): any {
    return new Auth().loggedInUser;
  }
}

export default (BaseComponent);
