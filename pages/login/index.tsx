import React from "react";
import { withRouter, useRouter } from 'next/router';
import Image from "next/image";
import Head from "next/head";
import Header from "../../components/Header";
import styles from "./view.module.scss";
import SigninButtonGoogle from "../../public/static/images/googleSigninButtons/btn_google_signin_dark_normal_web@2x.png"
import SigninButtonGithub from "../../public/static/images/signin_with_github.png"
import BaseComponent from "../../components/BaseComponent";

class LoginPage extends BaseComponent<any>{
  render() {
    const router = this.props.router; // useRouter();
    const callbackURL = router.query.callbackURL || "/";
    console.log("Router: ", router, callbackURL);
    const shouldRedirectIfLoggedIn = false;
    if (shouldRedirectIfLoggedIn) {
      // This check will ensure that we will never show the login page unless logged out
      // redirect to home if already logged in
      if (this.isLoggedIn) {
        router.push(callbackURL);
      }
    }
    return (
      <main>
        <Header showLoginButton={false}
                styles={styles} showSearchButton ={false} />
        <div style={{textAlign: "center"}}>
          <div className={styles.authProviderDiv}>
            <button className={styles.authProviderButton}>
              <a href={`/auth/google?callbackURL=${callbackURL}`}>
                <Image width={300} height={75} alt="Sign in with Google" src={SigninButtonGoogle} />
              </a>
            </button>
          </div>

          <div className={styles.authProviderDiv}>
            <button className={styles.authProviderButton}>
              <a href={`/auth/github?callbackURL=${callbackURL}`}>
                <Image width={300} height={75} alt="Sign in with Github" src={SigninButtonGithub} />
              </a>
            </button>
          </div>

          {false ? 
          <div className={styles.authProviderDiv}>
            <button className={styles.authProviderButton}>
              <a href="/auth/facebook?callbackURL={{callbackURL}}"><img className="img" src="/static/images/continue_with_facebook.png" width="300" alt="Continue with Facebook"/></a>
            </button>
          </div>
          :null}
        </div>
      </main>
    );
  }
}

export default withRouter(LoginPage);
